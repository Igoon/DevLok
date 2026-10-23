//go:build darwin

package doctor

import (
	"context"
	"strings"
	"time"

	"devlok/internal/brew"
	"devlok/internal/platform"
)

func platformChecks(plat platform.Platform) []Check {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out := []Check{{
		Name:    "platform",
		Status:  StatusOK,
		Message: "macOS; runtime dikelola Homebrew (nginx, php, mysql).",
	}}
	if _, err := brew.Prefix(ctx, ""); err != nil {
		out = append(out, Check{Name: "homebrew", Status: StatusFail, Message: "Homebrew tidak ditemukan.", Hint: "Instal dari https://brew.sh."})
	} else {
		out = append(out, Check{Name: "homebrew", Status: StatusOK, Message: "Homebrew tersedia."})
	}
	for _, formula := range []string{"nginx", "php", "mysql"} {
		out = append(out, checkBrewFormula(ctx, formula))
	}
	return out
}

func checkBrewFormula(ctx context.Context, formula string) Check {
	name := "brew_" + formula
	prefix, err := brew.Prefix(ctx, formula)
	if err != nil {
		return Check{
			Name:    name,
			Status:  StatusFail,
			Message: "Formula Homebrew tidak ditemukan: " + formula,
			Hint:    "Jalankan brew install " + formula + ".",
		}
	}
	version := brew.Version(ctx, formula)
	if version == "" {
		version = "?"
	}
	return Check{Name: name, Status: StatusOK, Message: strings.TrimSpace(formula + " " + version + " (" + prefix + ")")}
}
