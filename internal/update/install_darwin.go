//go:build darwin

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
	if ext == ".dmg" || ext == ".zip" {
		return path, nil, nil
	}
	return "", nil, devlokerr.New(devlokerr.CodeUnsupported, "Format update tidak didukung: "+ext, "Di macOS gunakan .dmg atau .zip.")
}

func LaunchInstaller(path string) error {
	return applyMac(path)
}

func LaunchInstallerDetached(path string) error {
	return applyMac(path)
}

func ScheduleRollback(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Rollback otomatis tidak tersedia di macOS.", "Pasang ulang aplikasi dari halaman rilis.")
}

func applyMac(path string) error {
	if _, _, err := InstallerCommand(path); err != nil {
		return err
	}
	target := currentAppBundle()
	if target == "" {
		if _, err := os.Stat("/Applications/DevLok.app"); err == nil {
			target = "/Applications/DevLok.app"
		}
	}
	if target == "" || !writableDir(filepath.Dir(target)) {
		return openMac(path)
	}
	staged, err := stageApp(path)
	if err != nil || staged == "" {
		return openMac(path)
	}
	cmd := exec.Command("/bin/sh", "-c", replaceScript(os.Getpid(), staged, target))
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(filepath.Dir(staged))
		return openMac(path)
	}
	return nil
}

func openMac(path string) error {
	if err := exec.Command("open", path).Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Update tidak dapat dibuka.", "Buka file unduhan secara manual.")
	}
	return nil
}

func currentAppBundle() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	marker := ".app/Contents/MacOS/"
	i := strings.Index(exe, marker)
	if i < 0 {
		return ""
	}
	return exe[:i+len(".app")]
}

func writableDir(dir string) bool {
	probe := filepath.Join(dir, fmt.Sprintf(".devlok-probe-%d", os.Getpid()))
	if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
		return false
	}
	_ = os.Remove(probe)
	return true
}

func stageApp(path string) (string, error) {
	tmp, err := os.MkdirTemp("", "devlok-update-")
	if err != nil {
		return "", err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".zip":
		if out, err := exec.Command("ditto", "-x", "-k", path, tmp).CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmp)
			return "", fmt.Errorf("ditto unzip: %w: %s", err, strings.TrimSpace(string(out)))
		}
	case ".dmg":
		mnt := filepath.Join(tmp, "mnt")
		if err := os.MkdirAll(mnt, 0o700); err != nil {
			_ = os.RemoveAll(tmp)
			return "", err
		}
		if out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-mountpoint", mnt, path).CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmp)
			return "", fmt.Errorf("hdiutil attach: %w: %s", err, strings.TrimSpace(string(out)))
		}
		src, findErr := findApp(mnt)
		if findErr == nil {
			dst := filepath.Join(tmp, "DevLok.app")
			if out, err := exec.Command("ditto", src, dst).CombinedOutput(); err != nil {
				findErr = fmt.Errorf("ditto: %w: %s", err, strings.TrimSpace(string(out)))
			}
		}
		_ = exec.Command("hdiutil", "detach", mnt).Run()
		if findErr != nil {
			_ = os.RemoveAll(tmp)
			return "", findErr
		}
	default:
		_ = os.RemoveAll(tmp)
		return "", fmt.Errorf("format tidak didukung")
	}
	app, err := findApp(tmp)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	return app, nil
}

func findApp(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("bundel .app tidak ditemukan")
}

func replaceScript(pid int, staged, target string) string {
	tmp := filepath.Dir(staged)
	return fmt.Sprintf(`#!/bin/sh
while kill -0 %d 2>/dev/null; do sleep 0.3; done
sleep 1
bak="%s.old.$(date +%%s)"
if mv "%s" "$bak" 2>/dev/null; then
  if ditto "%s" "%s"; then
    xattr -dr com.apple.quarantine "%s" 2>/dev/null || true
    open "%s"
    rm -rf "$bak"
    rm -rf "%s"
    exit 0
  fi
  rm -rf "%s"
  mv "$bak" "%s"
fi
rm -rf "%s"
exit 1
`, pid, target, target, staged, target, target, target, tmp, target, target, tmp)
}
