//go:build !windows

package update

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"devlok/internal/devlokerr"
)

func InstallerCommand(path string) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if runtime.GOOS == "darwin" && (ext == ".dmg" || ext == ".zip") {
		return path, nil, nil
	}
	return "", nil, devlokerr.New(devlokerr.CodeUnsupported, "Format update tidak didukung: "+ext, "Di macOS gunakan .dmg atau .zip.")
}

func openPath(path string) error {
	if err := exec.Command("open", path).Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Update tidak dapat dibuka.", "Buka file unduhan secara manual.")
	}
	return nil
}

func LaunchInstaller(path string) error {
	if _, _, err := InstallerCommand(path); err != nil {
		return err
	}
	return openPath(path)
}

func LaunchInstallerDetached(path string) error {
	if _, _, err := InstallerCommand(path); err != nil {
		return err
	}
	return openPath(path)
}

func ScheduleRollback(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Rollback otomatis tidak tersedia di platform ini.", "Pasang ulang aplikasi dari halaman rilis.")
}
