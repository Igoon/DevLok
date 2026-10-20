package instance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrAlreadyRunning = errors.New("instance already running")

type Lock struct {
	release func() error
}

func (l *Lock) Close() error {
	if l == nil || l.release == nil {
		return nil
	}
	return l.release()
}

func mutexName(scope string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(scope)))
	return "Local\\DevLok-" + hex.EncodeToString(sum[:8])
}
