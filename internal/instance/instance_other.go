//go:build !windows

package instance

import (
	"os"
	"path/filepath"
)

func Acquire(scope string) (*Lock, error) {
	path := filepath.Join(os.TempDir(), mutexName(scope)+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}
	_ = f.Close()
	return &Lock{release: func() error {
		return os.Remove(path)
	}}, nil
}
