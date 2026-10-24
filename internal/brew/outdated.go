package brew

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"devlok/internal/devlokerr"
)

var managedFormulae = map[string]bool{
	"nginx": true,
	"php":   true,
	"mysql": true,
	"node":  true,
}

type OutdatedFormula struct {
	Name      string
	Installed string
	Latest    string
}

func Outdated(ctx context.Context) ([]OutdatedFormula, error) {
	out, err := exec.CommandContext(ctx, binary(), "outdated", "--formula", "--json=v2").Output()
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Homebrew tidak dapat memeriksa paket usang.", "")
	}
	var doc struct {
		Formulae []struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Output brew outdated tidak valid.", "")
	}
	var res []OutdatedFormula
	for _, f := range doc.Formulae {
		if !managedFormulae[f.Name] {
			continue
		}
		installed := ""
		if len(f.InstalledVersions) > 0 {
			installed = f.InstalledVersions[0]
		}
		res = append(res, OutdatedFormula{Name: f.Name, Installed: installed, Latest: f.CurrentVersion})
	}
	return res, nil
}

func Upgrade(ctx context.Context, formula string) (string, error) {
	out, err := exec.CommandContext(ctx, binary(), "upgrade", formula).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		return "", devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "brew upgrade "+formula+" gagal.", msg)
	}
	return msg, nil
}
