//go:build !windows && !darwin

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type otherPlatform struct{}

func newPlatform() Platform { return otherPlatform{} }

func (otherPlatform) Name() string { return runtime.GOOS }

func (otherPlatform) StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".devlok"), nil
}

func (p otherPlatform) CacheDir() (string, error) {
	state, err := p.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "cache"), nil
}

func (otherPlatform) HostsPath() string { return "/etc/hosts" }

func (p otherPlatform) ReadHosts() ([]byte, error) {
	return os.ReadFile(p.HostsPath())
}

func (otherPlatform) WriteHosts([]byte) error {
	return errors.New("managed hosts writer belum tersedia")
}

func (otherPlatform) TrustCA(string) error   { return errors.New("trust CA belum tersedia") }
func (otherPlatform) UntrustCA(string) error { return errors.New("untrust CA belum tersedia") }

func (otherPlatform) KillTree(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("pid tidak valid: %d", pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func (otherPlatform) KillProcessesUnder(string) ([]int, error) { return nil, nil }

func (otherPlatform) ProcessStartTime(int) (time.Time, error) {
	return time.Time{}, errors.New("process start time belum tersedia")
}

func (otherPlatform) OpenBrowser(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}

func (otherPlatform) OpenFolder(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}

func (otherPlatform) OpenTerminal(string, []string) error {
	return errors.New("open terminal belum tersedia")
}

func (otherPlatform) OpenEditor(path string) error {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return exec.Command(editor, path).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-t", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}

func (otherPlatform) FlushDNS() error {
	if runtime.GOOS == "darwin" {
		return exec.Command("dscacheutil", "-flushcache").Run()
	}
	return nil
}

func (otherPlatform) ExeSuffix() string { return "" }
