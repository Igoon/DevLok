//go:build !windows && !darwin

package privileged

import "devlok/internal/devlokerr"

func SyncHosts([]string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Helper hosts hanya tersedia di Windows.", "")
}

func RemoveHosts() error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Helper hosts hanya tersedia di Windows.", "")
}

func WriteResolver(string, string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Resolver /etc/resolver hanya tersedia di macOS.", "")
}

func RemoveResolver(string) error {
	return devlokerr.New(devlokerr.CodeUnsupported, "Resolver /etc/resolver hanya tersedia di macOS.", "")
}
