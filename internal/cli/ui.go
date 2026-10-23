package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			if err := platform.Current().OpenApp(exe); err != nil {
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
	p := platform.Current()
	appSuffix := p.AppSuffix()

	var dirs []string
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		dirs = append(dirs,
			dir,
			filepath.Join(dir, "..", "..", "desktop", "build", "bin"),
			filepath.Join(dir, "..", "..", "desktop", "build"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs,
			cwd,
			filepath.Join(cwd, "build", "bin"),
			filepath.Join(cwd, "build"),
			filepath.Join(cwd, "desktop", "build", "bin"),
			filepath.Join(cwd, "desktop", "build"),
		)
	}
	if appSuffix != "" {
		dirs = append(dirs, "/Applications")
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, "Applications"))
		}
	}

	var names []string
	if appSuffix != "" {
		names = append(names, "devlok-gui"+appSuffix, "DevLok"+appSuffix)
	}
	names = append(names, "devlok-gui"+p.ExeSuffix())

	for _, dir := range dirs {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			info, err := os.Stat(candidate)
			if err != nil {
				continue
			}
			if !info.IsDir() || strings.HasSuffix(candidate, ".app") {
				return candidate, nil
			}
		}
	}
	return "", devlokerr.New(devlokerr.CodeDependencyMissing, "GUI DevLok tidak ditemukan.", guiBuildHint())
}

func guiBuildHint() string {
	if platform.Current().AppSuffix() != "" {
		return "Jalankan scripts/build-macos.sh, atau wails build di folder desktop."
	}
	return "Jalankan wails build di folder desktop."
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
