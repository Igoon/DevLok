//go:build darwin

package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rt "devlok/internal/runtime"
	"devlok/internal/supervisor"
)

func newPHPSpec(php rt.Installed, iniPath, logPath, generatedDir, projectDir string, port int) (supervisor.Spec, error) {
	fpm := filepath.Join(php.Path, "sbin", "php-fpm")
	if _, err := os.Stat(fpm); err != nil {
		fpm = filepath.Join(php.Path, "bin", "php-fpm")
	}
	if _, err := os.Stat(fpm); err != nil {
		return supervisor.Spec{}, fmt.Errorf("php-fpm Homebrew tidak ditemukan di %s", php.Path)
	}
	confDir := filepath.Join(generatedDir, "php", php.Version)
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return supervisor.Spec{}, err
	}
	confPath := filepath.Join(confDir, "php-fpm.conf")
	conf := fmt.Sprintf(`[global]
error_log = %s
daemonize = no

[www]
listen = 127.0.0.1:%d
listen.allowed_clients = 127.0.0.1
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
catch_workers_output = yes
clear_env = no
`, filepath.ToSlash(logPath), port)
	if err := os.WriteFile(confPath, []byte(conf), 0o644); err != nil {
		return supervisor.Spec{}, err
	}
	return supervisor.Spec{
		Name:       "php",
		Executable: fpm,
		Args:       []string{"--nodaemonize", "--fpm-config", confPath, "-c", iniPath},
		Dir:        projectDir,
		Health:     supervisor.Health{TCPPort: port, Timeout: 10 * time.Second},
		Policy:     supervisor.PolicyNo,
		Persistent: true,
	}, nil
}
