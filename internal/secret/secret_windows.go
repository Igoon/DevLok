//go:build windows

package secret

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func Seal(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	result := unsafe.Slice(out.Data, out.Size)
	cp := make([]byte, len(result))
	copy(cp, result)
	return cp, nil
}

func Unseal(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	result := unsafe.Slice(out.Data, out.Size)
	cp := make([]byte, len(result))
	copy(cp, result)
	return cp, nil
}
