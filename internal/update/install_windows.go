//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"devlok/internal/devlokerr"
	"devlok/internal/hidden"
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
	script := fmt.Sprintf("@echo off\r\ntimeout /t 3 /nobreak >nul\r\nstart \"\" %s\r\ndel \"%%~f0\" >nul 2>&1\r\n", line)
	if err := runScript(script); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Installer update tidak dapat dijadwalkan.", "")
	}
	return nil
}

func quoteCmd(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func ScheduleRollback(appDir string) error {
	backup := RollbackBackupDir(appDir)
	if _, err := os.Stat(filepath.Join(backup, "devlok-gui.exe")); err != nil {
		return devlokerr.New(devlokerr.CodeDependencyMissing, "Backup aplikasi sebelumnya tidak ditemukan.", "Rollback hanya tersedia setelah upgrade melalui installer DevLok.")
	}
	script := fmt.Sprintf("@echo off\r\ntimeout /t 3 /nobreak >nul\r\ntaskkill /IM devlok-gui.exe /F >nul 2>&1\r\ntimeout /t 2 /nobreak >nul\r\ncopy /Y %s %s >nul\r\nstart \"\" %s\r\ndel \"%%~f0\" >nul 2>&1\r\n",
		quoteCmd(filepath.Join(backup, "*")),
		quoteCmd(appDir),
		quoteCmd(filepath.Join(appDir, "devlok-gui.exe")),
	)
	if err := runScript(script); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Rollback tidak dapat dijadwalkan.", "")
	}
	return nil
}

func runScript(script string) error {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("devlok-update-%d.cmd", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("cmd.exe", "/c", path)
	hidden.Hide(cmd)
	if err := cmd.Start(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}
