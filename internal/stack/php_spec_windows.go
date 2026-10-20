//go:build windows

package stack

import (
	"fmt"
	"path/filepath"
	"time"

	rt "devlok/internal/runtime"
	"devlok/internal/supervisor"
)

func newPHPSpec(php rt.Installed, iniPath, logPath, generatedDir, projectDir string, port int) (supervisor.Spec, error) {
	return supervisor.Spec{
		Name:       "php",
		Executable: filepath.Join(php.Path, "php-cgi.exe"),
		Args:       []string{"-b", fmt.Sprintf("127.0.0.1:%d", port), "-c", iniPath},
		Dir:        projectDir,
		Env:        []string{"PHP_FCGI_CHILDREN=4", "PHP_FCGI_MAX_REQUESTS=0"},
		Health:     supervisor.Health{TCPPort: port, Timeout: 10 * time.Second},
		Policy:     supervisor.PolicyNo,
		Persistent: true,
	}, nil
}
