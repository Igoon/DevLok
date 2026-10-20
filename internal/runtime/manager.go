package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"devlok/internal/devlokerr"
)

type Options struct {
	Root       string
	Cache      string
	Offline    bool
	HTTP       *http.Client
	OS         string
	Arch       string
	Registries []Registry
	Progress   func(Progress)
}

type Manager struct {
	Registries []Registry
	Root       string
	Cache      string
	Offline    bool
	HTTP       *http.Client
	OS         string
	Arch       string
	Progress   func(Progress)
}

type Progress struct {
	Phase      string
	Name       string
	Version    string
	Downloaded int64
	Total      int64
}

type Installed struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	SHA256      string    `json:"sha256"`
	InstalledAt time.Time `json:"installed_at"`
}

type marker struct {
	Schema    int       `json:"schema"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	SHA256    string    `json:"sha256"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Installed time.Time `json:"installed_at"`
}

const markerName = ".devlok-runtime.json"

func NewManager(opts Options) (*Manager, error) {
	if opts.Root == "" || opts.Cache == "" {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Root dan cache runtime wajib diisi.", "")
	}
	regs := opts.Registries
	if regs == nil {
		loaded, err := LoadEmbedded()
		if err != nil {
			return nil, err
		}
		regs = loaded
	}
	client := opts.HTTP
	if client == nil {
		client = &http.Client{}
	}
	osName := opts.OS
	if osName == "" {
		osName = runtime.GOOS
	}
	arch := opts.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	return &Manager{
		Registries: regs,
		Root:       opts.Root,
		Cache:      opts.Cache,
		Offline:    opts.Offline,
		HTTP:       client,
		OS:         osName,
		Arch:       arch,
		Progress:   opts.Progress,
	}, nil
}

func (m *Manager) Install(ctx context.Context, name, constraint string) (Installed, bool, error) {
	res, err := m.resolve(name, constraint)
	if err != nil {
		return Installed{}, false, err
	}
	dest := m.installDir(res.Runtime.Name, res.Version.Version)
	if mk, err := readMarker(dest); err == nil && mk.SHA256 == res.Artifact.SHA256 {
		return Installed{Name: mk.Name, Version: mk.Version, Path: dest, SHA256: mk.SHA256, InstalledAt: mk.Installed}, false, nil
	}

	archivePath := m.cachePath(res)
	downloaded := false
	if ok, err := verifyFile(archivePath, res.Artifact.SHA256); err == nil && ok {
		m.notify(Progress{Phase: "verify", Name: name, Version: res.Version.Version})
	} else {
		if m.Offline {
			return Installed{}, false, devlokerr.New(devlokerr.CodeDependencyMissing,
				"Runtime tidak ada di cache dan mode offline aktif: "+name+"@"+res.Version.Version,
				"Jalankan tanpa --offline atau unduh lebih dulu.")
		}
		_ = os.Remove(archivePath)
		m.notify(Progress{Phase: "download", Name: name, Version: res.Version.Version, Total: res.Artifact.Size})
		if err := m.download(ctx, res.Artifact.URL, archivePath, res.Artifact.SHA256, res.Artifact.Size, name, res.Version.Version); err != nil {
			return Installed{}, false, err
		}
		downloaded = true
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return Installed{}, false, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder runtime tidak dapat dibuat.", "Periksa izin state dir.")
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".tmp-*")
	if err != nil {
		return Installed{}, false, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder sementara tidak dapat dibuat.", "")
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmp)
		}
	}()

	m.notify(Progress{Phase: "extract", Name: name, Version: res.Version.Version})
	if err := extractArchive(archivePath, tmp, res.Artifact); err != nil {
		return Installed{}, downloaded, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Arsip runtime tidak dapat diekstrak.", "Hapus cache lalu coba lagi.")
	}
	if res.Version.StripRoot == nil || *res.Version.StripRoot {
		if err := stripSingleRoot(tmp); err != nil {
			return Installed{}, downloaded, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Layout arsip tidak dapat dinormalisasi.", "")
		}
	}
	if err := validateExecutables(tmp, res.Artifact); err != nil {
		return Installed{}, downloaded, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Runtime tidak lengkap setelah ekstraksi.", "Registry mungkin menunjuk arsip yang salah.")
	}
	mk := marker{
		Schema:    1,
		Name:      res.Runtime.Name,
		Version:   res.Version.Version,
		SHA256:    res.Artifact.SHA256,
		OS:        m.OS,
		Arch:      m.Arch,
		Installed: time.Now().UTC(),
	}
	if err := writeMarker(tmp, mk); err != nil {
		return Installed{}, downloaded, err
	}
	if _, err := os.Stat(dest); err == nil {
		if err := os.RemoveAll(dest); err != nil {
			return Installed{}, downloaded, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Versi lama tidak dapat diganti.", "Pastikan runtime tidak sedang dipakai.")
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		return Installed{}, downloaded, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Runtime tidak dapat dipindahkan ke folder tujuan.", "Pastikan tidak ada proses yang memakai file runtime.")
	}
	cleanup = false
	m.notify(Progress{Phase: "done", Name: name, Version: res.Version.Version})
	return Installed{Name: mk.Name, Version: mk.Version, Path: dest, SHA256: mk.SHA256, InstalledAt: mk.Installed}, downloaded, nil
}

func (m *Manager) Resolve(name, constraint string) (Resolved, error) {
	return m.resolve(name, constraint)
}

func (m *Manager) Remove(name, version string) error {
	if !nameRe.MatchString(name) || !versionRe.MatchString(version) {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Target runtime tidak valid: "+name+"@"+version, "Gunakan format nama@versi.")
	}
	dir := m.installDir(name, version)
	if _, err := readMarker(dir); err != nil {
		return devlokerr.New(devlokerr.CodeRuntimeNotFound, "Runtime belum terpasang: "+name+"@"+version, "Jalankan dl runtime list.")
	}
	if err := os.RemoveAll(dir); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Runtime tidak dapat dihapus.", "Pastikan tidak ada proses yang memakai file runtime.")
	}
	_ = os.Remove(filepath.Dir(dir))
	return nil
}

func (m *Manager) ListInstalled() ([]Installed, error) {
	entries, err := os.ReadDir(m.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Installed
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		versions, err := os.ReadDir(filepath.Join(m.Root, e.Name()))
		if err != nil {
			continue
		}
		for _, v := range versions {
			if !v.IsDir() || strings.HasPrefix(v.Name(), ".") {
				continue
			}
			mk, err := readMarker(filepath.Join(m.Root, e.Name(), v.Name()))
			if err != nil || mk.Name != e.Name() || mk.Version != v.Name() {
				continue
			}
			out = append(out, Installed{Name: mk.Name, Version: mk.Version, Path: filepath.Join(m.Root, e.Name(), v.Name()), SHA256: mk.SHA256, InstalledAt: mk.Installed})
		}
	}
	return out, nil
}

func (m *Manager) Verify(name, constraint string) error {
	res, err := m.resolve(name, constraint)
	if err != nil {
		return err
	}
	dir := m.installDir(res.Runtime.Name, res.Version.Version)
	mk, err := readMarker(dir)
	if err != nil {
		return devlokerr.New(devlokerr.CodeRuntimeNotFound, "Runtime belum terpasang: "+name+"@"+res.Version.Version, "Jalankan dl runtime add.")
	}
	if mk.SHA256 != res.Artifact.SHA256 {
		return devlokerr.New(devlokerr.CodeChecksumMismatch, "Marker runtime tidak cocok dengan registry.", "Pasang ulang runtime.")
	}
	if err := validateExecutables(dir, res.Artifact); err != nil {
		return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "File runtime hilang.", "Pasang ulang runtime.")
	}
	cache := m.cachePath(res)
	if _, err := os.Stat(cache); err == nil {
		ok, err := verifyFile(cache, res.Artifact.SHA256)
		if err != nil {
			return err
		}
		if !ok {
			return devlokerr.New(devlokerr.CodeChecksumMismatch, "Checksum arsip cache tidak cocok: "+cache, "Hapus file cache lalu pasang ulang.")
		}
	}
	return nil
}

func (m *Manager) installDir(name, version string) string {
	return filepath.Join(m.Root, name, version)
}

func (m *Manager) cachePath(res Resolved) string {
	key := strings.ReplaceAll(res.Platform, "/", "-")
	return filepath.Join(m.Cache, fmt.Sprintf("%s-%s-%s%s", res.Runtime.Name, res.Version.Version, key, archiveExt(res.Artifact)))
}

func archiveExt(a Artifact) string {
	switch a.Archive {
	case ArchiveZip:
		return ".zip"
	case ArchiveTarGz:
		return ".tar.gz"
	case ArchiveFile:
		if a.Filename != "" {
			return filepath.Ext(a.Filename)
		}
		if u, err := parseBase(a.URL); err == nil {
			return filepath.Ext(u)
		}
	}
	return ".bin"
}

func parseBase(raw string) (string, error) {
	i := strings.IndexByte(raw, '?')
	if i >= 0 {
		raw = raw[:i]
	}
	i = strings.LastIndexByte(raw, '/')
	if i < 0 {
		return raw, nil
	}
	return raw[i+1:], nil
}

func (m *Manager) download(ctx context.Context, rawURL, dest, wantSHA string, wantSize int64, name, version string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder cache tidak dapat dibuat.", "")
	}
	part := dest + ".part"
	var offset int64
	if info, err := os.Stat(part); err == nil {
		offset = info.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "URL runtime tidak valid.", "")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Runtime tidak dapat diunduh.", "Periksa koneksi, proxy, atau firewall.")
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	switch resp.StatusCode {
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	case http.StatusOK:
		offset = 0
		flags |= os.O_TRUNC
	case http.StatusRequestedRangeNotSatisfiable:
		return m.finishDownload(part, dest, wantSHA, name, version)
	default:
		return devlokerr.New(devlokerr.CodeDependencyMissing,
			fmt.Sprintf("Unduhan gagal (%d) untuk %s@%s.", resp.StatusCode, name, version),
			"Server menolak permintaan. Coba lagi atau gunakan proxy.")
	}
	f, err := os.OpenFile(part, flags, 0o600)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File cache tidak dapat ditulis.", "")
	}
	total := wantSize
	if resp.ContentLength > 0 {
		total = offset + resp.ContentLength
	}
	pw := &progressWriter{w: f, offset: offset, total: total, cb: func(done int64) {
		m.notify(Progress{Phase: "download", Name: name, Version: version, Downloaded: done, Total: total})
	}}
	_, copyErr := io.Copy(pw, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, copyErr, "Unduhan terputus.", "Jalankan ulang perintah untuk melanjutkan.")
	}
	if closeErr != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, closeErr, "File cache tidak dapat ditutup.", "")
	}
	return m.finishDownload(part, dest, wantSHA, name, version)
}

func (m *Manager) finishDownload(part, dest, wantSHA, name, version string) error {
	m.notify(Progress{Phase: "verify", Name: name, Version: version})
	ok, err := verifyFile(part, wantSHA)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Arsip runtime tidak dapat diverifikasi.", "")
	}
	if !ok {
		_ = os.Remove(part)
		return devlokerr.New(devlokerr.CodeChecksumMismatch,
			"Checksum runtime tidak cocok: "+name+"@"+version,
			"File dihapus. Coba unduh ulang; jika berulang, periksa registry atau proxy.")
	}
	_ = os.Remove(dest)
	if err := os.Rename(part, dest); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Cache runtime tidak dapat diselesaikan.", "")
	}
	return nil
}

func (m *Manager) notify(p Progress) {
	if m.Progress != nil {
		m.Progress(p)
	}
}

type progressWriter struct {
	w      io.Writer
	offset int64
	total  int64
	n      int64
	last   time.Time
	cb     func(int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.n += int64(n)
	now := time.Now()
	if p.cb != nil && (now.Sub(p.last) > 150*time.Millisecond || (p.total > 0 && p.offset+p.n >= p.total)) {
		p.last = now
		p.cb(p.offset + p.n)
	}
	return n, err
}

func verifyFile(path, wantSHA string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == strings.ToLower(wantSHA), nil
}

func writeMarker(dir string, mk marker) error {
	data, err := json.MarshalIndent(mk, "", "  ")
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeInternal, err, "Marker runtime tidak dapat dibuat.", "")
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, markerName), data, 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Marker runtime tidak dapat ditulis.", "")
	}
	return nil
}

func readMarker(dir string) (marker, error) {
	data, err := os.ReadFile(filepath.Join(dir, markerName))
	if err != nil {
		return marker{}, err
	}
	if len(data) > 64<<10 {
		return marker{}, errors.New("marker too large")
	}
	var mk marker
	if err := json.Unmarshal(data, &mk); err != nil {
		return marker{}, err
	}
	if mk.Name == "" || mk.Version == "" || mk.SHA256 == "" {
		return marker{}, errors.New("incomplete marker")
	}
	return mk, nil
}
