//go:build darwin

package tlsca

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func loginKeychain() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "Library", "Keychains", "login.keychain-db")
	}
	return filepath.Join(home, "Library", "Keychains", "login.keychain-db")
}

func trustCA(certPath string) error {
	cmd := exec.Command("security", "add-trusted-cert", "-r", "trustRoot", "-k", loginKeychain(), certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security add-trusted-cert: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func untrustCA(thumbprint string) error {
	cmd := exec.Command("security", "delete-certificate", "-Z", strings.ToUpper(thumbprint), "-t", loginKeychain())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security delete-certificate: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
