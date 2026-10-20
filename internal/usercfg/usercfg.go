package usercfg

import (
	"os"
	"path/filepath"

	"devlok/internal/devlokerr"
)

func Root(stateDir string) string {
	return filepath.Join(stateDir, "config")
}

func Dir(stateDir, kind string) string {
	return filepath.Join(Root(stateDir), kind)
}

func Path(stateDir, kind, name string) string {
	return filepath.Join(Dir(stateDir, kind), name)
}

func Ensure(path, content string) error {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder konfigurasi pengguna tidak dapat dibuat.", "")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi pengguna tidak dapat ditulis.", "")
	}
	return nil
}

func Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}
