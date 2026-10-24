//go:build !darwin

package cli

import "devlok/internal/devlokerr"

func wildcardEnable(string) (string, error) {
	return "", devlokerr.New(devlokerr.CodeUnsupported, "Wildcard dnsmasq hanya tersedia di macOS.", "Di Windows tambahkan entri hosts per domain atau gunakan dnsmasq manual.")
}

func wildcardDisable(string) (string, error) {
	return "", devlokerr.New(devlokerr.CodeUnsupported, "Wildcard dnsmasq hanya tersedia di macOS.", "")
}

func wildcardStatus() (WildcardStatus, error) {
	return WildcardStatus{}, devlokerr.New(devlokerr.CodeUnsupported, "Wildcard dnsmasq hanya tersedia di macOS.", "")
}
