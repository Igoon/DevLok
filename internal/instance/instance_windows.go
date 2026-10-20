//go:build windows

package instance

import (
	"golang.org/x/sys/windows"
)

func Acquire(scope string) (*Lock, error) {
	name, err := windows.UTF16PtrFromString(mutexName(scope))
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(h)
		return nil, ErrAlreadyRunning
	}
	if err != nil {
		return nil, err
	}
	return &Lock{release: func() error {
		return windows.CloseHandle(h)
	}}, nil
}
