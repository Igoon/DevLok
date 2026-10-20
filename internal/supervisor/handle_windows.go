//go:build windows

package supervisor

import (
	"os/exec"
	"unsafe"

	"devlok/internal/hidden"
	"golang.org/x/sys/windows"
)

type procHandle struct {
	job windows.Handle
}

func newProcHandle(persistent bool) (*procHandle, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	if !persistent {
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		if _, err := windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		); err != nil {
			_ = windows.CloseHandle(h)
			return nil, err
		}
	}
	return &procHandle{job: h}, nil
}

func (p *procHandle) beforeStart(cmd *exec.Cmd) error {
	hidden.Hide(cmd)
	return nil
}

func (p *procHandle) afterStart(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(p.job, h)
}

func (p *procHandle) kill() error {
	if p.job == 0 {
		return nil
	}
	return windows.TerminateJobObject(p.job, 1)
}

func (p *procHandle) close() error {
	if p.job == 0 {
		return nil
	}
	h := p.job
	p.job = 0
	return windows.CloseHandle(h)
}
