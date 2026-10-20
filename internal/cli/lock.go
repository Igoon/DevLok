package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/config"
	"devlok/internal/envarchive"
	rt "devlok/internal/runtime"
)

func newLockCommand(jsonOut *bool) *cobra.Command {
	var projectDir string
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Buat .devlok.lock dari manifest dan registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, project, err := manifestFor(projectDir)
			if err != nil {
				return err
			}
			mgr, err := newRuntimeManager(false, nil)
			if err != nil {
				return err
			}
			lock, err := buildLock(m, mgr)
			if err != nil {
				return err
			}
			path := filepath.Join(project, config.LockFileName)
			if err := config.SaveLock(path, lock); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "lock", true, map[string]any{"file": path, "lock": lock})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Lockfile dibuat: %s\n", path)
			fmt.Fprintf(cmd.OutOrStdout(), "nginx %s, php %s, mysql %s\n", lock.Web.Version, lock.Runtime["php"].Version, lock.Services["mysql"].Version)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectDir, "path", "", "Folder project")
	return cmd
}

func buildLock(m *config.Manifest, mgr *rt.Manager) (*config.Lock, error) {
	nginxConstraint := m.Runtime["nginx"]
	nginx, err := mgr.Resolve("nginx", nginxConstraint)
	if err != nil {
		return nil, err
	}
	php, err := mgr.Resolve("php", m.Web.PHP)
	if err != nil {
		return nil, err
	}
	l := &config.Lock{
		Schema:      config.LockSchemaVersion,
		GeneratedAt: time.Now().UTC(),
		Platform:    mgr.OS + "/" + mgr.Arch,
		Name:        m.Name,
		Domain:      m.BaseDomain(),
		Web: config.LockedWeb{
			Server:    "nginx",
			Version:   nginx.Version.Version,
			SHA256:    nginx.Artifact.SHA256,
			URL:       nginx.Artifact.URL,
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Runtime:  map[string]config.LockedRuntime{},
		Services: map[string]config.LockedRuntime{},
	}
	l.Runtime["php"] = config.LockedRuntime{Version: php.Version.Version, SHA256: php.Artifact.SHA256, URL: php.Artifact.URL}
	if constraint, ok := m.Runtime["node"]; ok && constraint != "" {
		node, err := mgr.Resolve("node", constraint)
		if err != nil {
			return nil, err
		}
		l.Runtime["node"] = config.LockedRuntime{Version: node.Version.Version, SHA256: node.Artifact.SHA256, URL: node.Artifact.URL}
	}
	mysqlConstraint := m.Services["mysql"]
	if mysqlConstraint == "" {
		mysqlConstraint = "8.4"
	}
	mysql, err := mgr.Resolve("mysql", mysqlConstraint)
	if err != nil {
		return nil, err
	}
	l.Services["mysql"] = config.LockedRuntime{Version: mysql.Version.Version, SHA256: mysql.Artifact.SHA256, URL: mysql.Artifact.URL}
	return l, nil
}

func newExportCommand(jsonOut *bool) *cobra.Command {
	var out, projectDir string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Ekspor manifest, lockfile, dan seed tanpa rahasia",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, project, err := manifestFor(projectDir)
			if err != nil {
				return err
			}
			if out == "" {
				out = "devlok-" + m.Name + ".zip"
			}
			path, err := envarchive.Export(project, out)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "export", true, map[string]string{"file": path})
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "File zip output")
	cmd.Flags().StringVar(&projectDir, "path", "", "Folder project")
	return cmd
}

func manifestFor(projectDir string) (*config.Manifest, string, error) {
	if projectDir == "" {
		return currentManifest()
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, "", err
	}
	path := abs
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		path = filepath.Join(abs, config.FileName)
	}
	m, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	return m, filepath.Dir(path), nil
}
