package envarchive

import (
	"archive/zip"
	"os"
	"path/filepath"

	"devlok/internal/config"
	"devlok/internal/devlokerr"
)

func Export(projectDir, outPath string) (string, error) {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return "", err
	}
	manifestPath := filepath.Join(abs, config.FileName)
	m, err := config.Load(manifestPath)
	if err != nil {
		return "", err
	}
	if outPath == "" {
		outPath = "devlok-" + m.Name + ".zip"
	}
	outAbs, err := filepath.Abs(outPath)
	if err != nil {
		return "", err
	}
	if samePath(outAbs, manifestPath) {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "Output tidak boleh menimpa manifest.", "")
	}
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder output tidak dapat dibuat.", "")
	}
	f, err := os.OpenFile(outAbs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Arsip export tidak dapat ditulis.", "")
	}
	zw := zip.NewWriter(f)
	add := func(name, src string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	if err := add(config.FileName, manifestPath); err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(outAbs)
		return "", err
	}
	lockPath := filepath.Join(abs, config.LockFileName)
	if _, err := os.Stat(lockPath); err == nil {
		if err := add(config.LockFileName, lockPath); err != nil {
			_ = zw.Close()
			_ = f.Close()
			_ = os.Remove(outAbs)
			return "", err
		}
	}
	seed := filepath.Join(abs, "db", "seed.sql")
	if _, err := os.Stat(seed); err == nil {
		if err := add("db/seed.sql", seed); err != nil {
			_ = zw.Close()
			_ = f.Close()
			_ = os.Remove(outAbs)
			return "", err
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		_ = os.Remove(outAbs)
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return outAbs, nil
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}
