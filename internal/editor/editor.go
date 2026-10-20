package editor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

func Open(path string) error {
	if path == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Path editor kosong.", "")
	}
	var candidates []string
	if state, err := platform.Current().StateDir(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(state, "runtimes", "notepadpp", "*", "notepad++.exe"))
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		candidates = append(candidates, matches...)
	}
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Notepad++", "notepad++.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Notepad++", "notepad++.exe"),
	} {
		if _, err := os.Stat(p); err == nil {
			candidates = append(candidates, p)
		}
	}
	if code, err := exec.LookPath("code"); err == nil {
		dir := filepath.Dir(code)
		for _, p := range []string{
			filepath.Join(dir, "..", "Code.exe"),
			filepath.Join(dir, "..", "..", "Code.exe"),
		} {
			if _, err := os.Stat(p); err == nil {
				candidates = append(candidates, p)
				break
			}
		}
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		p := filepath.Join(local, "Programs", "Microsoft VS Code", "Code.exe")
		if _, err := os.Stat(p); err == nil {
			candidates = append(candidates, p)
		}
	}
	candidates = append(candidates, "notepad.exe")

	var last error
	for _, exe := range candidates {
		if exe == "" {
			continue
		}
		cmd := exec.Command(exe, path)
		if err := cmd.Start(); err != nil {
			last = err
			continue
		}
		return nil
	}
	if last == nil {
		last = errors.New("tidak ada editor yang ditemukan")
	}
	return devlokerr.Wrap(devlokerr.CodeDependencyMissing, last, "Editor tidak dapat dibuka.", "Instal Notepad++ atau VS Code, atau jalankan dl runtime add notepadpp.")
}
