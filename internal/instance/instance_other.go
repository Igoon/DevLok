//go:build !windows

package instance

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func Acquire(scope string) (*Lock, error) {
	path := filepath.Join(os.TempDir(), mutexName(scope)+".lock")
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString(strconv.Itoa(os.Getpid()))
			_ = f.Close()
			return &Lock{release: func() error {
				return os.Remove(path)
			}}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if lockHolderAlive(path) {
			return nil, ErrAlreadyRunning
		}
		_ = os.Remove(path)
	}
	return nil, ErrAlreadyRunning
}

func lockHolderAlive(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	if pid == os.Getpid() {
		return true
	}
	err = syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}
