//go:build darwin

package privileged

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
)

func SyncHosts(domains []string) error {
	return run("hosts-sync", domains)
}

func RemoveHosts() error {
	return run("hosts-remove", nil)
}

func run(operation string, domains []string) error {
	helper, err := helperPath()
	if err != nil {
		return err
	}
	args := []string{operation}
	if operation == "hosts-sync" {
		state, err := platform.Current().StateDir()
		if err != nil {
			return err
		}
		payload := filepath.Join(state, "state", "hosts-domains.txt")
		if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(payload, []byte(strings.Join(domains, "\n")+"\n"), 0o600); err != nil {
			return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Payload hosts tidak dapat ditulis.", "")
		}
		args = append(args, payload)
	}
	parts := []string{shellQuote(helper)}
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	command := strings.Join(parts, " ")
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, appleQuote(command))
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return devlokerr.New(devlokerr.CodeElevationRequired, "Hosts membutuhkan izin Administrator.", msg)
	}
	return nil
}

func helperPath() (string, error) {
	var candidates []string
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		candidates = append(candidates,
			filepath.Join(dir, "dl-helper"),
			filepath.Join(dir, "..", "..", "build", "bin", "dl-helper"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "build", "bin", "dl-helper"),
			filepath.Join(cwd, "desktop", "build", "bin", "dl-helper"),
		)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", devlokerr.New(devlokerr.CodeDependencyMissing, "dl-helper tidak ditemukan.", "Build dengan go build -o build/bin/dl-helper ./cmd/dl-helper.")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func appleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
