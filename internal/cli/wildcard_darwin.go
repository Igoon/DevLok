//go:build darwin

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"devlok/internal/brew"
	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	"devlok/internal/privileged"
)

const wildcardPort = 5353

func wildcardPaths() (string, string, string, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", "", "", err
	}
	return filepath.Join(state, "generated", "dnsmasq", "dnsmasq.conf"),
		filepath.Join(state, "run", "dnsmasq.pid"),
		filepath.Join(state, "logs", "dnsmasq.log"),
		nil
}

func wildcardEnable(tld string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prefix, err := brew.Prefix(ctx, "dnsmasq")
	if err != nil {
		return "", devlokerr.New(devlokerr.CodeDependencyMissing, "dnsmasq belum terpasang.", "Jalankan brew install dnsmasq.")
	}
	binary := filepath.Join(prefix, "sbin", "dnsmasq")
	conf, pidPath, logPath, err := wildcardPaths()
	if err != nil {
		return "", err
	}
	for _, dir := range []string{filepath.Dir(conf), filepath.Dir(pidPath), filepath.Dir(logPath)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder wildcard tidak dapat dibuat.", "")
		}
	}
	body := fmt.Sprintf("port=%d\nlisten-address=127.0.0.1\nbind-interfaces\nno-resolv\nno-hosts\naddress=/%s/127.0.0.1\npid-file=%s\nlog-facility=%s\n",
		wildcardPort, tld, pidPath, logPath)
	if err := os.WriteFile(conf, []byte(body), 0o644); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi dnsmasq tidak dapat ditulis.", "")
	}
	if out, err := exec.Command(binary, "--test", "--conf-file="+conf).CombinedOutput(); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Konfigurasi dnsmasq tidak valid.", strings.TrimSpace(string(out)))
	}
	stopDnsmasq(pidPath)
	if err := exec.Command(binary, "--conf-file="+conf).Run(); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "dnsmasq gagal dijalankan.", readFileTail(logPath, 5))
	}
	if err := privileged.WriteResolver(tld, fmt.Sprintf("nameserver 127.0.0.1\nport %d\n", wildcardPort)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wildcard *.%s aktif via dnsmasq (127.0.0.1:%d).", tld, wildcardPort), nil
}

func wildcardDisable(tld string) (string, error) {
	_, pidPath, _, err := wildcardPaths()
	if err != nil {
		return "", err
	}
	stopped := stopDnsmasq(pidPath)
	if err := privileged.RemoveResolver(tld); err != nil {
		return "", err
	}
	if stopped {
		return "Wildcard dihentikan dan /etc/resolver/" + tld + " dihapus.", nil
	}
	return "/etc/resolver/" + tld + " dihapus (dnsmasq tidak berjalan).", nil
}

func wildcardStatus() (WildcardStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	st := WildcardStatus{TLD: "test"}
	if _, err := brew.Prefix(ctx, "dnsmasq"); err == nil {
		st.Installed = true
	}
	_, pidPath, _, err := wildcardPaths()
	if err == nil {
		if data, err := os.ReadFile(pidPath); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 && processAlive(pid) {
				st.Running = true
				st.PID = pid
			}
		}
	}
	if _, err := os.Stat("/etc/resolver/" + st.TLD); err == nil {
		st.Resolver = true
	}
	return st, nil
}

func stopDnsmasq(pidPath string) bool {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		_ = os.Remove(pidPath)
		return false
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Signal(syscall.SIGTERM)
	}
	_ = os.Remove(pidPath)
	return true
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func readFileTail(path string, lines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}
