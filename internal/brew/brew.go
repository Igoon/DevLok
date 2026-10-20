package brew

import (
	"context"
	"os/exec"
	"strings"

	"devlok/internal/devlokerr"
)

func Prefix(ctx context.Context, formula string) (string, error) {
	args := []string{"--prefix"}
	if formula != "" {
		args = append(args, formula)
	}
	out, err := exec.CommandContext(ctx, "brew", args...).Output()
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
	out, err := exec.CommandContext(ctx, "brew", "list", "--versions", formula).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return ""
	}
	return strings.Split(fields[len(fields)-1], "_")[0]
}
