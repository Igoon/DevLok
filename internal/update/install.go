package update

import (
	"fmt"
	"os"
	"os/exec"
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

func LaunchInstaller(path string) error {
	name, args, err := InstallerCommand(path)
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = filepath.Dir(path)
	if err := cmd.Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Installer update tidak dapat dijalankan.", "")
	}
	return nil
}

func LaunchInstallerDetached(path string) error {
	name, args, err := InstallerCommand(path)
	if err != nil {
		return err
	}
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		quoted = append(quoted, quoteCmd(a))
	}
	line := quoteCmd(name)
	if len(quoted) > 0 {
		line += " " + strings.Join(quoted, " ")
	}
	script := fmt.Sprintf("timeout /t 3 /nobreak >nul & start \"\" %s", line)
	cmd := exec.Command("cmd.exe", "/c", script)
	if err := cmd.Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Installer update tidak dapat dijadwalkan.", "")
	}
	return nil
}

func quoteCmd(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func RollbackBackupDir(appDir string) string {
	return filepath.Join(appDir, "backup", "previous")
}

func ScheduleRollback(appDir string) error {
	backup := RollbackBackupDir(appDir)
	if _, err := os.Stat(filepath.Join(backup, "devlok-gui.exe")); err != nil {
		return devlokerr.New(devlokerr.CodeDependencyMissing, "Backup aplikasi sebelumnya tidak ditemukan.", "Rollback hanya tersedia setelah upgrade melalui installer DevLok.")
	}
	app := quoteCmd(appDir)
	backupPattern := quoteCmd(filepath.Join(backup, "*"))
	gui := quoteCmd(filepath.Join(appDir, "devlok-gui.exe"))
	script := fmt.Sprintf(`timeout /t 3 /nobreak >nul & taskkill /IM devlok-gui.exe /F >nul 2>&1 & timeout /t 2 /nobreak >nul & copy /Y %s %s >nul & start "" %s`, backupPattern, app, gui)
	cmd := exec.Command("cmd.exe", "/c", script)
	if err := cmd.Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Rollback tidak dapat dijadwalkan.", "")
	}
	return nil
}
