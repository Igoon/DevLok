package phpcfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"devlok/internal/devlokerr"
)

var DefaultExtensions = []string{
	"mysqli",
	"pdo_mysql",
	"mbstring",
	"openssl",
	"curl",
	"fileinfo",
	"gd",
	"zip",
	"intl",
	"exif",
	"sodium",
	"sqlite3",
	"pdo_sqlite",
}

type settings struct {
	Schema     int      `json:"schema"`
	Extensions []string `json:"extensions"`
}

func Path(stateDir, version string) string {
	return filepath.Join(stateDir, "generated", "php", version, "extensions.json")
}

func Load(stateDir, version string) []string {
	data, err := os.ReadFile(Path(stateDir, version))
	if err != nil {
		return append([]string{}, DefaultExtensions...)
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return append([]string{}, DefaultExtensions...)
	}
	return normalize(s.Extensions)
}

func Save(stateDir, version string, extensions []string) error {
	path := Path(stateDir, version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder konfigurasi PHP tidak dapat dibuat.", "")
	}
	s := settings{Schema: 1, Extensions: normalize(extensions)}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi ekstensi PHP tidak dapat ditulis.", "")
	}
	return nil
}

func normalize(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
