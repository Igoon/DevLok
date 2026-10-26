package cli

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

func newUICommand() *cobra.Command {
	var browser bool
	var port int
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Buka GUI DevLok",
		RunE: func(cmd *cobra.Command, args []string) error {
			if browser {
				return runBrowserMode(cmd, port)
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
	cmd.Flags().BoolVar(&browser, "browser", false, "Buka mode browser (tanpa WebView)")
	cmd.Flags().IntVar(&port, "port", 0, "Port mode browser (0 = otomatis)")
	return cmd
}

// runBrowserMode menjalankan GUI dalam mode browser: menyajikan frontend lewat
// HTTP lokal dan membuka browser default.
func runBrowserMode(cmd *cobra.Command, port int) error {
	exe, err := findGUIBinary()
	if err != nil {
		return err
	}
	if port <= 0 {
		port = freePort()
	}
	if port <= 0 {
		return devlokerr.New(devlokerr.CodeInternal, "Tidak ada port bebas untuk mode browser.", "")
	}
	c := exec.Command(exe, "--browser", "--port", strconv.Itoa(port))
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	if err := c.Start(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Mode browser tidak dapat dijalankan.", "")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Mode browser: http://127.0.0.1:%d/\n", port)
	return nil
}

func freePort() int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer ln.Close()
	if t, ok := ln.Addr().(*net.TCPAddr); ok {
		return t.Port
	}
	return 0
}

// findGUIBinary mengembalikan path executable GUI. Untuk bundle .app macOS,
// mengembalikan binary di dalam Contents/MacOS agar argumen dapat diteruskan.
func findGUIBinary() (string, error) {
	exe, err := findGUI()
	if err != nil {
		return "", err
	}
	if platform.Current().AppSuffix() != "" && strings.HasSuffix(exe, ".app") {
		inner := filepath.Join(exe, "Contents", "MacOS", "devlok-gui")
		if _, statErr := os.Stat(inner); statErr == nil {
			return inner, nil
		}
	}
	return exe, nil
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
