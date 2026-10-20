//go:build !windows

package supervisor

import (
	"os/exec"
	"syscall"
)

type procHandle struct {
	pid int
}

func newProcHandle(bool) (*procHandle, error) { return &procHandle{}, nil }

func (p *procHandle) beforeStart(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func (p *procHandle) afterStart(pid int) error {
	p.pid = pid
	return nil
}

func (p *procHandle) kill() error {
	if p.pid <= 0 {
		return nil
	}
	return syscall.Kill(-p.pid, syscall.SIGKILL)
}

func (p *procHandle) close() error { return nil }
