//go:build darwin

package editor

import (
	"os"
	"os/exec"
	"path/filepath"

	"devlok/internal/devlokerr"
)

func Open(path string) error {
	if path == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Path editor kosong.", "")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(abs)
	isDir := statErr == nil && info.IsDir()

	for _, app := range []string{"Visual Studio Code", "Cursor", "Zed", "Sublime Text", "TextMate"} {
		if exec.Command("open", "-a", app, abs).Run() == nil {
			return nil
		}
	}
	args := []string{abs}
	if !isDir {
		args = []string{"-t", abs}
	}
	if err := exec.Command("open", args...).Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Editor tidak dapat dibuka.", "Instal VS Code, atau gunakan TextEdit/Finder.")
	}
	return nil
}
