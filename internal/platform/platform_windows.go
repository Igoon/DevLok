//go:build windows

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
	"unsafe"

	"devlok/internal/hidden"
	"golang.org/x/sys/windows"
)

type windowsPlatform struct{}

func newPlatform() Platform { return windowsPlatform{} }

func (windowsPlatform) Name() string { return "windows" }

func (windowsPlatform) StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".devlok"), nil
}

func (p windowsPlatform) CacheDir() (string, error) {
	state, err := p.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "cache"), nil
}

func (windowsPlatform) HostsPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "drivers", "etc", "hosts")
}

func (p windowsPlatform) ReadHosts() ([]byte, error) {
	return os.ReadFile(p.HostsPath())
}

func (windowsPlatform) WriteHosts([]byte) error {
	return errors.New("managed hosts writer belum tersedia")
}

func (windowsPlatform) TrustCA(string) error {
	return errors.New("trust CA belum tersedia")
}

func (windowsPlatform) UntrustCA(string) error {
	return errors.New("untrust CA belum tersedia")
}

func (windowsPlatform) KillTree(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("pid tidak valid: %d", pid)
	}
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	hidden.Hide(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("taskkill: %w: %s", err, string(out))
	}
	return nil
}

func (windowsPlatform) KillProcessesUnder(root string) ([]int, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, err
	}
	prefix := strings.ToLower(filepath.Clean(root)) + string(os.PathSeparator)
	self := os.Getpid()
	var pids []int
	for {
		pid := int(entry.ProcessID)
		if pid > 0 && pid != self {
			if path := processPath(pid); path != "" && strings.HasPrefix(strings.ToLower(path), prefix) {
				pids = append(pids, pid)
			}
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	for _, pid := range pids {
		_ = windowsPlatform{}.KillTree(pid)
	}
	return pids, nil
}

func processPath(pid int) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func (windowsPlatform) ProcessStartTime(pid int) (time.Time, error) {
	if pid <= 0 {
		return time.Time{}, fmt.Errorf("pid tidak valid: %d", pid)
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return time.Time{}, err
	}
	defer windows.CloseHandle(h)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, creation.Nanoseconds()), nil
}

func (windowsPlatform) OpenBrowser(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	hidden.Hide(cmd)
	return cmd.Start()
}

func (windowsPlatform) OpenFolder(path string) error {
	return exec.Command("explorer.exe", path).Start()
}

func (windowsPlatform) OpenTerminal(dir string, env []string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", "cmd.exe", "/k")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	return cmd.Start()
}

func (windowsPlatform) OpenEditor(path string) error {
	if p, err := exec.LookPath("code"); err == nil {
		cmd := exec.Command("cmd.exe", "/c", p, path)
		hidden.Hide(cmd)
		return cmd.Start()
	}
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Notepad++", "notepad++.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Notepad++", "notepad++.exe"),
	} {
		if _, err := os.Stat(p); err == nil {
			return exec.Command(p, path).Start()
		}
	}
	return exec.Command("notepad.exe", path).Start()
}

func (windowsPlatform) OpenApp(path string) error {
	cmd := exec.Command(path)
	hidden.Hide(cmd)
	return cmd.Start()
}

func (windowsPlatform) FlushDNS() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	hidden.Hide(cmd)
	return cmd.Run()
}

func (windowsPlatform) ExeSuffix() string { return ".exe" }

func (windowsPlatform) AppSuffix() string { return "" }
