package importlaragon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"devlok/internal/config"
	"devlok/internal/devlokerr"
)

type Site struct {
	Name       string   `json:"name"`
	ProjectDir string   `json:"project_dir"`
	Domain     string   `json:"domain"`
	Root       string   `json:"root"`
	TLD        string   `json:"tld"`
	Aliases    []string `json:"aliases,omitempty"`
	PHP        string   `json:"php"`
	Source     string   `json:"source"`
}

type Result struct {
	Site    Site   `json:"site"`
	Action  string `json:"action"`
	Message string `json:"message,omitempty"`
}

var (
	serverNameRe = regexp.MustCompile(`(?m)^\s*server_name\s+([^;]+);`)
	rootRe       = regexp.MustCompile(`(?m)^\s*root\s+"?([^";]+)"?\s*;`)
)

func Plan(laragonDir string) ([]Site, error) {
	abs, err := filepath.Abs(laragonDir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Folder Laragon tidak ditemukan.", "")
	}
	files, _ := filepath.Glob(filepath.Join(abs, "etc", "nginx", "sites-enabled", "auto.*.conf"))
	sort.Strings(files)
	seen := map[string]int{}
	var out []Site
	for _, file := range files {
		site, ok := parseVhost(file)
		if !ok {
			continue
		}
		if idx, exists := seen[site.ProjectDir]; exists {
			merged := append([]string{out[idx].Domain}, out[idx].Aliases...)
			merged = append(merged, site.Domain)
			merged = append(merged, site.Aliases...)
			out[idx].Aliases = dedupeDomains(merged, out[idx].Domain)
			continue
		}
		seen[site.ProjectDir] = len(out)
		out = append(out, site)
	}
	www := filepath.Join(abs, "www")
	if entries, err := os.ReadDir(www); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(www, e.Name())
			if _, exists := seen[dir]; exists {
				continue
			}
			root := config.DetectRoot(dir)
			if root == "" {
				continue
			}
			name := config.SanitizeName(e.Name())
			out = append(out, Site{
				Name:       name,
				ProjectDir: dir,
				Domain:     name + ".test",
				Root:       root,
				TLD:        "test",
				PHP:        "8.4",
				Source:     "www-scan",
			})
			seen[dir] = len(out)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseVhost(path string) (Site, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Site{}, false
	}
	text := string(data)
	names := serverNameRe.FindStringSubmatch(text)
	root := rootRe.FindStringSubmatch(text)
	if len(names) < 2 || len(root) < 2 {
		return Site{}, false
	}
	var domains []string
	for _, f := range strings.Fields(names[1]) {
		f = strings.Trim(f, `"'`)
		if f == "" || strings.Contains(f, "*") || strings.HasPrefix(f, "_") || f == "localhost" {
			continue
		}
		if config.ValidDomain(f) {
			domains = append(domains, strings.ToLower(f))
		}
	}
	if len(domains) == 0 {
		return Site{}, false
	}
	rootPath := strings.TrimSpace(root[1])
	rootPath = strings.Trim(rootPath, `"`)
	rootPath = filepath.FromSlash(rootPath)
	projectDir := rootPath
	base := strings.ToLower(filepath.Base(filepath.Clean(rootPath)))
	if base == "public" || base == "web" || base == "www" {
		projectDir = filepath.Dir(filepath.Clean(rootPath))
	}
	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		return Site{}, false
	}
	rel, err := filepath.Rel(projectDir, rootPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Site{}, false
	}
	primary := domains[0]
	labels := strings.Split(primary, ".")
	tld := "test"
	if len(labels) > 1 {
		tld = labels[len(labels)-1]
	}
	name := config.SanitizeName(filepath.Base(projectDir))
	mainDomain := name + "." + tld
	var aliases []string
	for _, d := range domains {
		if d != mainDomain {
			aliases = append(aliases, d)
		}
	}
	return Site{
		Name:       name,
		ProjectDir: projectDir,
		Domain:     mainDomain,
		Root:       rel,
		TLD:        tld,
		Aliases:    aliases,
		PHP:        "8.4",
		Source:     filepath.Base(path),
	}, true
}

func Import(sites []Site, dryRun bool) ([]Result, error) {
	results := make([]Result, 0, len(sites))
	for _, site := range sites {
		manifestPath := filepath.Join(site.ProjectDir, config.FileName)
		if _, err := os.Stat(manifestPath); err == nil {
			results = append(results, Result{Site: site, Action: "skip", Message: "manifest sudah ada"})
			continue
		}
		if dryRun {
			results = append(results, Result{Site: site, Action: "create", Message: "dry-run"})
			continue
		}
		m := &config.Manifest{
			Schema: config.SchemaVersion,
			Name:   site.Name,
			Web: config.Web{
				Server:  "nginx",
				Root:    site.Root,
				TLD:     site.TLD,
				Aliases: site.Aliases,
				PHP:     site.PHP,
			},
			Runtime:  map[string]string{"php": site.PHP},
			Services: map[string]string{"mysql": "8.4"},
		}
		config.ApplyDefaults(m)
		if err := config.Validate(m, site.ProjectDir); err != nil {
			results = append(results, Result{Site: site, Action: "error", Message: err.Error()})
			continue
		}
		data, err := config.Marshal(m)
		if err != nil {
			results = append(results, Result{Site: site, Action: "error", Message: err.Error()})
			continue
		}
		if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
			results = append(results, Result{Site: site, Action: "error", Message: err.Error()})
			continue
		}
		results = append(results, Result{Site: site, Action: "create", Message: manifestPath})
	}
	return results, nil
}

func Summary(results []Result) string {
	created, skipped, failed := 0, 0, 0
	for _, r := range results {
		switch r.Action {
		case "create":
			created++
		case "skip":
			skipped++
		case "error":
			failed++
		}
	}
	return fmt.Sprintf("%d dibuat, %d dilewati, %d gagal", created, skipped, failed)
}

func dedupeDomains(in []string, exclude string) []string {
	seen := map[string]bool{exclude: true}
	var out []string
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" || seen[d] || !config.ValidDomain(d) {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
