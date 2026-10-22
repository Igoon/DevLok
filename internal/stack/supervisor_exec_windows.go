//go:build windows

package stack

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func startSupervisor(stateDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Base(exe), "dl.exe") {
		candidate := filepath.Join(filepath.Dir(exe), "dl.exe")
		if _, err := os.Stat(candidate); err != nil {
			return err
		}
		exe = candidate
	}
	cmd := exec.Command(exe, "supervise", "--state", stateDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS | windows.CREATE_NO_WINDOW,
	}
	return cmd.Start()
}
