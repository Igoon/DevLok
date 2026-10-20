//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"devlok/internal/platform"
)

func lastProjectPath() string {
	state, err := platform.Current().StateDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(state, "state", "last-project.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveLastProject(path string) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	dir := filepath.Join(state, "state")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "last-project.txt"), []byte(path+"\n"), 0o600)
}
