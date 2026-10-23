//go:build darwin

package ports

import (
	"net"
	"os/exec"
	"strconv"
	"strings"
)

func listeningOn(port int) bool {
	if err := exec.Command(lsofPath(), "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN").Run(); err == nil {
		return true
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

func ownerOf(port int) (Owner, bool) {
	out, err := exec.Command(lsofPath(), "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-Fpc").Output()
	if err != nil {
		return Owner{}, false
	}
	pid, command := parseLsof(string(out))
	if pid <= 0 {
		return Owner{}, false
	}
	return Owner{PID: pid, Process: command, Path: processPath(pid)}, true
}

func parseLsof(out string) (int, string) {
	pid := 0
	command := ""
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			if pid == 0 {
				if v, err := strconv.Atoi(line[1:]); err == nil {
					pid = v
				}
			}
		case 'c':
			if command == "" {
				command = line[1:]
			}
		}
	}
	return pid, command
}

func processPath(pid int) string {
	out, err := exec.Command(psPath(), "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func lsofPath() string {
	if p, err := exec.LookPath("lsof"); err == nil {
		return p
	}
	return "/usr/sbin/lsof"
}

func psPath() string {
	if p, err := exec.LookPath("ps"); err == nil {
		return p
	}
	return "/bin/ps"
}
