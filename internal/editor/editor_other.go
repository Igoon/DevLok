//go:build !windows && !darwin

package editor

import (
	"os"
	"os/exec"

	"devlok/internal/devlokerr"
)

func Open(path string) error {
	if path == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Path editor kosong.", "")
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		if err := exec.Command(ed, path).Start(); err == nil {
			return nil
		}
	}
	for _, name := range []string{"code", "codium", "gedit", "xdg-open"} {
		if bin, err := exec.LookPath(name); err == nil {
			if err := exec.Command(bin, path).Start(); err == nil {
				return nil
			}
		}
	}
	return devlokerr.New(devlokerr.CodeDependencyMissing, "Editor tidak ditemukan.", "Set $EDITOR atau instal VS Code.")
}
