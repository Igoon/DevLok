package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

func newUICommand() *cobra.Command {
	var browser bool
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Buka GUI DevLok",
		RunE: func(cmd *cobra.Command, args []string) error {
			if browser {
				return devlokerr.New(devlokerr.CodeUnsupported, "Mode browser belum tersedia.", "Gunakan GUI native: dl ui")
			}
			exe, err := findGUI()
			if err != nil {
				return err
			}
			requestShow()
			c := exec.Command(exe)
			c.Stdout = nil
			c.Stderr = nil
			if err := c.Start(); err != nil {
				return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "GUI tidak dapat dijalankan.", "")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "GUI dibuka:", exe)
			return nil
		},
	}
	cmd.Flags().BoolVar(&browser, "browser", false, "Buka mode browser")
	return cmd
}

func findGUI() (string, error) {
	var candidates []string
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		candidates = append(candidates,
			filepath.Join(dir, "devlok-gui.exe"),
			filepath.Join(dir, "..", "..", "desktop", "build", "bin", "devlok-gui.exe"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "devlok-gui.exe"),
			filepath.Join(cwd, "build", "bin", "devlok-gui.exe"),
			filepath.Join(cwd, "desktop", "build", "bin", "devlok-gui.exe"),
		)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", devlokerr.New(devlokerr.CodeDependencyMissing, "devlok-gui.exe tidak ditemukan.", "Jalankan wails build di folder desktop.")
}

func requestShow() {
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	dir := filepath.Join(state, "state")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "show.request"), []byte("1"), 0o600)
}
