//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type darwinPlatform struct{}

func newPlatform() Platform { return darwinPlatform{} }

func (darwinPlatform) Name() string { return "darwin" }

func (darwinPlatform) StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".devlok"), nil
}

func (p darwinPlatform) CacheDir() (string, error) {
	state, err := p.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "cache"), nil
}

func (darwinPlatform) HostsPath() string { return "/etc/hosts" }

func (darwinPlatform) ReadHosts() ([]byte, error) {
	return os.ReadFile("/etc/hosts")
}

func (darwinPlatform) WriteHosts([]byte) error {
	return errors.New("managed hosts writer belum tersedia; gunakan helper")
}

func (darwinPlatform) TrustCA(certPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	keychain := filepath.Join(home, "Library", "Keychains", "login.keychain-db")
	cmd := exec.Command("security", "add-trusted-cert", "-r", "trustRoot", "-k", keychain, certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security add-trusted-cert: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (darwinPlatform) UntrustCA(certPath string) error {
	cmd := exec.Command("security", "remove-trusted-cert", certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security remove-trusted-cert: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (darwinPlatform) KillTree(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("pid tidak valid: %d", pid)
	}
	_ = exec.Command("pkill", "-TERM", "-P", strconv.Itoa(pid)).Run()
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (darwinPlatform) KillProcessesUnder(string) ([]int, error) {
	return nil, nil
}

func (darwinPlatform) ProcessStartTime(pid int) (time.Time, error) {
	if pid <= 0 {
		return time.Time{}, fmt.Errorf("pid tidak valid: %d", pid)
	}
	cmd := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return time.Time{}, fmt.Errorf("waktu proses %d tidak tersedia", pid)
	}
	t, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", text, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse waktu proses: %w", err)
	}
	return t, nil
}

func (darwinPlatform) OpenBrowser(url string) error {
	return exec.Command("open", url).Start()
}

func (darwinPlatform) OpenFolder(path string) error {
	return exec.Command("open", path).Start()
}

func (darwinPlatform) OpenTerminal(dir string, env []string) error {
	command := "cd " + shellQuote(dir)
	script := fmt.Sprintf(`tell application "Terminal"
do script %s
activate
end tell`, appleQuote(command))
	cmd := exec.Command("osascript", "-e", script)
	cmd.Env = append(os.Environ(), env...)
	return cmd.Start()
}

func (darwinPlatform) OpenEditor(path string) error {
	if code, err := exec.LookPath("code"); err == nil {
		return exec.Command(code, path).Start()
	}
	return exec.Command("open", "-t", path).Start()
}

func (darwinPlatform) FlushDNS() error {
	cmd := exec.Command("dscacheutil", "-flushcache")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dscacheutil: %w: %s", err, strings.TrimSpace(string(out)))
	}
	_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
	return nil
}

func (darwinPlatform) ExeSuffix() string { return "" }

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func appleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
