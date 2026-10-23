package update

import (
	"path/filepath"
	"strings"

	"devlok/internal/devlokerr"
)

func InstallerCommand(path string) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".exe":
		return path, nil, nil
	case ".msi":
		return "msiexec.exe", []string{"/i", path}, nil
	default:
		return "", nil, devlokerr.New(devlokerr.CodeUnsupported, "Format update tidak didukung: "+ext, "Gunakan installer .exe atau .msi.")
	}
}

func RollbackBackupDir(appDir string) string {
	return filepath.Join(appDir, "backup", "previous")
}
