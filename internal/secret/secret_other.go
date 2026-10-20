//go:build !windows

package secret

func Seal(data []byte) ([]byte, error) { return data, nil }

func Unseal(data []byte) ([]byte, error) { return data, nil }
