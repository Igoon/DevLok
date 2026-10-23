package brew

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"devlok/internal/devlokerr"
)

func binary() string {
	if p, err := exec.LookPath("brew"); err == nil {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew", "/home/linuxbrew/.linuxbrew/bin/brew"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return "brew"
}

func Prefix(ctx context.Context, formula string) (string, error) {
	args := []string{"--prefix"}
	if formula != "" {
		args = append(args, formula)
	}
	out, err := exec.CommandContext(ctx, binary(), args...).Output()
	if err != nil {
		hint := "Instal Homebrew."
		if formula != "" {
			hint = "Jalankan brew install " + formula + "."
		}
		return "", devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Homebrew tidak dapat menemukan "+formula+".", hint)
	}
	return strings.TrimSpace(string(out)), nil
}

func Version(ctx context.Context, formula string) string {
	out, err := exec.CommandContext(ctx, binary(), "list", "--versions", formula).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return ""
	}
	return strings.Split(fields[len(fields)-1], "_")[0]
}

func Series(v string) string {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return strings.TrimSpace(v)
}

func InstalledFormula(ctx context.Context, name, requested string) string {
	if series := Series(requested); series != "" {
		candidate := name + "@" + series
		if Version(ctx, candidate) != "" {
			return candidate
		}
	}
	return name
}

func InstalledVersioned(ctx context.Context, name string) []string {
	out, err := exec.CommandContext(ctx, binary(), "list", "--versions").Output()
	if err != nil {
		return nil
	}
	prefix := name + "@"
	var versions []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], prefix) {
			continue
		}
		versions = append(versions, strings.Split(fields[len(fields)-1], "_")[0])
	}
	return versions
}
