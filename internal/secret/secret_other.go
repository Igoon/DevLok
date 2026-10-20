//go:build !windows

package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

func Seal(data []byte) ([]byte, error) {
	gcm, err := aead()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Nonce kredensial tidak dapat dibuat.", "")
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

func Unseal(data []byte) ([]byte, error) {
	gcm, err := aead()
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Data kredensial rusak.", "")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	out, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Kredensial tidak dapat didekripsi.", "")
	}
	return out, nil
}

func aead() (cipher.AEAD, error) {
	key, err := key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Cipher kredensial tidak dapat dibuat.", "")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "GCM kredensial tidak dapat dibuat.", "")
	}
	return gcm, nil
}

func key() ([]byte, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "State dir tidak dapat ditentukan.", "")
	}
	dir := filepath.Join(state, "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder secret tidak dapat dibuat.", "")
	}
	path := filepath.Join(dir, "secret.key")
	if data, err := os.ReadFile(path); err == nil {
		if len(data) != 32 {
			return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Key kredensial rusak.", "")
		}
		return data, nil
	}
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Key kredensial tidak dapat dibuat.", "")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return os.ReadFile(path)
		}
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Key kredensial tidak dapat ditulis.", "")
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Key kredensial tidak dapat ditulis.", "")
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("key kredensial: %w", err)
	}
	return data, nil
}
