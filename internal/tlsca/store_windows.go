//go:build windows

package tlsca

import (
	"os/exec"

	"devlok/internal/hidden"
)

func trustCA(certPath string) error {
	cmd := exec.Command("certutil", "-user", "-f", "-addstore", "Root", certPath)
	hidden.Hide(cmd)
	return cmd.Run()
}

func untrustCA(thumbprint string) error {
	cmd := exec.Command("certutil", "-user", "-f", "-delstore", "Root", thumbprint)
	hidden.Hide(cmd)
	return cmd.Run()
}
