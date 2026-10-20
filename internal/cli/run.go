package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"devlok/internal/config"
	dbpkg "devlok/internal/db"
	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	rt "devlok/internal/runtime"
)

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "run -- <command>",
		Short:              "Jalankan perintah dengan PATH dan env project",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			for len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}
			if len(args) == 0 {
				return devlokerr.New(devlokerr.CodeInvalidArgument, "Perintah kosong.", "Contoh: dl run -- php -v")
			}
			env, project, err := projectEnv()
			if err != nil {
				return err
			}
			exe := resolveExecutable(args[0], getEnv(env, "PATH"))
			c := exec.CommandContext(cmd.Context(), exe, args[1:]...)
			c.Dir = project
			c.Env = env
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return exitError{code: exitErr.ExitCode()}
				}
				return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Perintah tidak dapat dijalankan: "+args[0], "Pastikan perintah tersedia di PATH.")
			}
			return nil
		},
	}
}

func newShellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Buka terminal dengan PATH dan env project",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, project, err := projectEnv()
			if err != nil {
				return err
			}
			if err := platform.Current().OpenTerminal(project, env); err != nil {
				return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Terminal tidak dapat dibuka.", "")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Terminal dibuka di", project)
			return nil
		},
	}
}

func projectEnv() ([]string, string, error) {
	m, project, err := currentManifest()
	if err != nil {
		return nil, "", err
	}
	lock, _ := config.LoadLock(filepath.Join(project, config.LockFileName))
	phpConstraint := m.Web.PHP
	nodeConstraint := m.Runtime["node"]
	if lock != nil {
		if v := lock.Runtime["php"].Version; v != "" {
			phpConstraint = v
		}
		if v := lock.Runtime["node"].Version; v != "" {
			nodeConstraint = v
		}
	}
	mgr, err := newRuntimeManager(false, nil)
	if err != nil {
		return nil, "", err
	}
	php, _, err := mgr.Install(context.Background(), "php", phpConstraint)
	if err != nil {
		return nil, "", err
	}
	paths := []string{filepath.Join(php.Path)}
	if nodeConstraint != "" {
		node, _, err := mgr.Install(context.Background(), "node", nodeConstraint)
		if err != nil {
			return nil, "", err
		}
		paths = append(paths, node.Path)
	}
	if shim := ensureComposerShim(mgr, php); shim != "" {
		paths = append(paths, shim)
	}
	paths = append(paths, os.Getenv("PATH"))
	env := setEnv(os.Environ(), "PATH", strings.Join(paths, string(os.PathListSeparator)))
	env = setEnv(env, "DEVLOK_PROJECT", project)
	if dc, err := projectDatabase(m, project); err == nil {
		env = setEnv(env, "DB_HOST", "127.0.0.1")
		env = setEnv(env, "DB_PORT", fmt.Sprintf("%d", dc.port))
		env = setEnv(env, "DB_DATABASE", dc.name)
		env = setEnv(env, "DB_USERNAME", dc.user)
		env = setEnv(env, "DB_PASSWORD", dc.password)
	}
	return env, project, nil
}

type dbEnv struct {
	name, user, password string
	port                 int
}

func projectDatabase(m *config.Manifest, project string) (dbEnv, error) {
	version := m.Services["mysql"]
	if version == "" {
		version = "8.4"
	}
	if lock, err := config.LoadLock(filepath.Join(project, config.LockFileName)); err == nil {
		if v := lock.Services["mysql"].Version; v != "" {
			version = v
		}
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		return dbEnv{}, err
	}
	mgr, err := dbpkg.New(dbpkg.Options{StateDir: state, Version: version, Port: m.Ports.MySQL})
	if err != nil {
		return dbEnv{}, err
	}
	dc, err := mgr.Database(m.DB.Name)
	if err != nil {
		return dbEnv{}, err
	}
	return dbEnv{name: dc.Name, user: dc.User, password: dc.Password, port: mgr.Port}, nil
}

func ensureComposerShim(mgr *rt.Manager, php rt.Installed) string {
	installed, err := mgr.ListInstalled()
	if err != nil {
		return ""
	}
	composerPath := ""
	for _, in := range installed {
		if in.Name != "composer" {
			continue
		}
		p := filepath.Join(in.Path, "composer.phar")
		if _, err := os.Stat(p); err == nil {
			composerPath = p
			break
		}
	}
	if composerPath == "" {
		return ""
	}
	dir := filepath.Join(filepath.Dir(filepath.Dir(php.Path)), "..", "shims")
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	phpExe := filepath.Join(php.Path, "php.exe")
	content := "@echo off\r\n\"" + phpExe + "\" \"" + composerPath + "\" %*\r\n"
	if err := os.WriteFile(filepath.Join(dir, "composer.bat"), []byte(content), 0o644); err != nil {
		return ""
	}
	return dir
}

func setEnv(env []string, key, value string) []string {
	prefix := strings.ToUpper(key) + "="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), prefix) {
			continue
		}
		out = append(out, e)
	}
	return append(out, key+"="+value)
}

func getEnv(env []string, key string) string {
	prefix := strings.ToUpper(key) + "="
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), prefix) {
			return e[len(prefix):]
		}
	}
	return ""
}

func resolveExecutable(name, pathEnv string) string {
	if strings.ContainsAny(name, `\/`) {
		return name
	}
	pathext := getEnv(os.Environ(), "PATHEXT")
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}
	exts := strings.Split(pathext, ";")
	if filepath.Ext(name) != "" {
		exts = []string{""}
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			candidate := filepath.Join(dir, name+ext)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}
	return name
}
