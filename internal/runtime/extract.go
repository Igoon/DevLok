package runtime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxEntries   = 200000
	maxTotalSize = int64(4 << 30)
)

func extractArchive(archivePath, dest string, art Artifact) error {
	switch art.Archive {
	case ArchiveZip:
		return extractZip(archivePath, dest)
	case ArchiveTarGz:
		return extractTarGz(archivePath, dest)
	case ArchiveFile:
		return extractFile(archivePath, dest, art)
	default:
		return fmt.Errorf("unsupported archive: %s", art.Archive)
	}
}

func extractFile(archivePath, dest string, art Artifact) error {
	name := art.Filename
	if name == "" {
		u, err := url.Parse(art.URL)
		if err != nil {
			return fmt.Errorf("parse url: %w", err)
		}
		name = path.Base(u.Path)
	}
	target, err := safeJoin(dest, name)
	if err != nil {
		return err
	}
	return copyFile(archivePath, target, 0o644)
}

func extractZip(archivePath, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()
	if len(r.File) > maxEntries {
		return fmt.Errorf("archive has too many entries: %d", len(r.File))
	}
	var total uint64
	for _, f := range r.File {
		total += f.UncompressedSize64
		if total > uint64(maxTotalSize) {
			return fmt.Errorf("archive expands beyond limit")
		}
	}
	for _, f := range r.File {
		if err := extractZipEntry(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, dest string) error {
	name := f.Name
	target, err := safeJoin(dest, name)
	if err != nil {
		return err
	}
	mode := f.Mode()
	if mode&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink entry rejected: %s", name)
	}
	if mode&os.ModeType != 0 && !f.FileInfo().IsDir() {
		return fmt.Errorf("irregular entry rejected: %s", name)
	}
	if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") || strings.HasSuffix(name, "\\") {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %s: %w", name, err)
	}
	defer rc.Close()
	perm := os.FileMode(0o644)
	if mode.Perm()&0o111 != 0 {
		perm = 0o755
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("extract %s: %w", name, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", name, closeErr)
	}
	return nil
}

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	entries := 0
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		entries++
		if entries > maxEntries {
			return fmt.Errorf("archive has too many entries")
		}
		total += hdr.Size
		if total > maxTotalSize {
			return fmt.Errorf("archive expands beyond limit")
		}
		name := hdr.Name
		target, err := safeJoin(dest, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			perm := os.FileMode(0o644)
			if hdr.FileInfo().Mode().Perm()&0o111 != 0 {
				perm = 0o755
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return fmt.Errorf("extract %s: %w", name, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close %s: %w", name, closeErr)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %d: %s", hdr.Typeflag, name)
		}
	}
	return nil
}

func safeJoin(dest, name string) (string, error) {
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("invalid entry name")
	}
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if clean == "." {
		return dest, nil
	}
	if strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("absolute or volume entry rejected: %s", name)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal rejected: %s", name)
	}
	for _, part := range strings.Split(clean, "/") {
		if isReservedName(part) {
			return "", fmt.Errorf("reserved name rejected: %s", name)
		}
	}
	target := filepath.Join(dest, filepath.FromSlash(clean))
	rel, err := filepath.Rel(dest, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes destination: %s", name)
	}
	return target, nil
}

func isReservedName(name string) bool {
	base := strings.ToUpper(strings.TrimSuffix(name, "/"))
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 {
		prefix := base[:3]
		if (prefix == "COM" || prefix == "LPT") && base[3] >= '1' && base[3] <= '9' {
			return true
		}
	}
	return false
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func stripSingleRoot(dir string) error {
	for i := 0; i < 4; i++ {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		if len(entries) != 1 || !entries[0].IsDir() {
			return nil
		}
		root := filepath.Join(dir, entries[0].Name())
		children, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		for _, child := range children {
			from := filepath.Join(root, child.Name())
			to := filepath.Join(dir, child.Name())
			if err := os.Rename(from, to); err != nil {
				return fmt.Errorf("strip root: %w", err)
			}
		}
		if err := os.Remove(root); err != nil {
			return err
		}
	}
	return nil
}

func validateExecutables(dir string, art Artifact) error {
	for _, exe := range art.Executables {
		p, err := safeJoin(dir, exe)
		if err != nil {
			return err
		}
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("executable missing after extract: %s", exe)
		}
		if info.IsDir() {
			return fmt.Errorf("executable is a directory: %s", exe)
		}
	}
	return nil
}

func zipCommonRoot(files []*zip.File) string {
	root := ""
	for _, f := range files {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		name = strings.TrimSuffix(name, "/")
		if name == "" {
			continue
		}
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 {
			return ""
		}
		if root == "" {
			root = parts[0]
		} else if root != parts[0] {
			return ""
		}
	}
	return root
}
