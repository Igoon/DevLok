//go:build !windows

package privileged

import "devlok/internal/devlokerr"

func SyncHosts([]string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Helper hosts hanya tersedia di Windows.", "")
}

func RemoveHosts() error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Helper hosts hanya tersedia di Windows.", "")
}
