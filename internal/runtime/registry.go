package runtime

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"devlok/internal/devlokerr"
)

//go:embed registry/*.json
var registryFS embed.FS

const registrySchema = 1

type Registry struct {
	Schema   int       `json:"schema"`
	Name     string    `json:"name"`
	Display  string    `json:"display"`
	Versions []Version `json:"versions"`
}

type Version struct {
	Version   string              `json:"version"`
	Artifacts map[string]Artifact `json:"artifacts"`
	StripRoot *bool               `json:"strip_root,omitempty"`
}

type Artifact struct {
	URL         string   `json:"url"`
	SHA256      string   `json:"sha256"`
	Size        int64    `json:"size,omitempty"`
	Archive     string   `json:"archive"`
	Filename    string   `json:"filename,omitempty"`
	Executables []string `json:"executables"`
}

type Resolved struct {
	Runtime  Registry
	Version  Version
	Artifact Artifact
	Platform string
}

const (
	ArchiveZip   = "zip"
	ArchiveTarGz = "tar.gz"
	ArchiveFile  = "file"
	defaultOS    = "windows"
	defaultArch  = "amd64"
)

var (
	nameRe     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	versionRe  = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}$`)
	platformRe = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+$`)
	sha256Re   = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func LoadEmbedded() ([]Registry, error) {
	entries, err := registryFS.ReadDir("registry")
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Registry runtime tidak dapat dibaca.", "")
	}
	out := make([]Registry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := registryFS.ReadFile(path.Join("registry", e.Name()))
		if err != nil {
			return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Registry tidak dapat dibaca: "+e.Name(), "")
		}
		reg, err := ParseRegistry(data)
		if err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, nil
}

func ParseRegistry(data []byte) (Registry, error) {
	var reg Registry
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&reg); err != nil {
		return Registry{}, devlokerr.Wrap(devlokerr.CodeInternal, err, "Registry runtime tidak valid.", "Periksa registry/*.json.")
	}
	if err := validateRegistry(reg); err != nil {
		return Registry{}, err
	}
	return reg, nil
}

func validateRegistry(reg Registry) error {
	if reg.Schema != registrySchema {
		return devlokerr.New(devlokerr.CodeInternal, "Schema registry tidak didukung.", "Gunakan schema 1.")
	}
	if !nameRe.MatchString(reg.Name) {
		return devlokerr.New(devlokerr.CodeInternal, "Nama runtime tidak valid: "+reg.Name, "")
	}
	if len(reg.Versions) == 0 {
		return devlokerr.New(devlokerr.CodeInternal, "Runtime tidak punya versi: "+reg.Name, "")
	}
	seen := map[string]bool{}
	for _, v := range reg.Versions {
		if !versionRe.MatchString(v.Version) {
			return devlokerr.New(devlokerr.CodeInternal, "Versi tidak valid: "+v.Version, "")
		}
		if seen[v.Version] {
			return devlokerr.New(devlokerr.CodeInternal, "Versi duplikat: "+v.Version, "")
		}
		seen[v.Version] = true
		if len(v.Artifacts) == 0 {
			return devlokerr.New(devlokerr.CodeInternal, "Artifact kosong untuk "+reg.Name+" "+v.Version, "")
		}
		for platform, a := range v.Artifacts {
			if !platformRe.MatchString(platform) {
				return devlokerr.New(devlokerr.CodeInternal, "Platform tidak valid: "+platform, "")
			}
			if err := validateArtifact(reg.Name, v.Version, a); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateArtifact(name, version string, a Artifact) error {
	u, err := url.Parse(a.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return devlokerr.New(devlokerr.CodeInternal, "URL artifact tidak valid: "+name+" "+version, "Gunakan URL HTTPS.")
	}
	if !sha256Re.MatchString(a.SHA256) {
		return devlokerr.New(devlokerr.CodeInternal, "SHA256 tidak valid: "+name+" "+version, "")
	}
	switch a.Archive {
	case ArchiveZip, ArchiveTarGz, ArchiveFile:
	default:
		return devlokerr.New(devlokerr.CodeInternal, "Tipe arsip tidak didukung: "+a.Archive, "Gunakan zip, tar.gz, atau file.")
	}
	if len(a.Executables) == 0 {
		return devlokerr.New(devlokerr.CodeInternal, "Daftar executable kosong: "+name+" "+version, "")
	}
	for _, exe := range a.Executables {
		if !safeRelative(exe) {
			return devlokerr.New(devlokerr.CodeInternal, "Path executable tidak aman: "+exe, "")
		}
	}
	if a.Archive == ArchiveFile && a.Filename != "" && !safeRelative(a.Filename) {
		return devlokerr.New(devlokerr.CodeInternal, "Nama file tidak aman: "+a.Filename, "")
	}
	return nil
}

func safeRelative(p string) bool {
	if p == "" || strings.Contains(p, ":") {
		return false
	}
	clean := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return false
	}
	return true
}

func (m *Manager) resolve(name, constraint string) (Resolved, error) {
	reg, ok := m.registry(name)
	if !ok {
		return Resolved{}, devlokerr.New(devlokerr.CodeRuntimeNotFound, "Runtime tidak dikenal: "+name, "Jalankan dl runtime list.")
	}
	candidates := make([]Version, 0, len(reg.Versions))
	for _, v := range reg.Versions {
		if matchConstraint(v.Version, constraint) {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return Resolved{}, devlokerr.New(devlokerr.CodeRuntimeNotFound,
			fmt.Sprintf("Versi %s@%s tidak tersedia.", name, constraint),
			"Versi tersedia: "+strings.Join(versionStrings(reg.Versions), ", "))
	}
	sort.Slice(candidates, func(i, j int) bool {
		return compareVersions(candidates[i].Version, candidates[j].Version) > 0
	})
	key := m.platformKey()
	for _, v := range candidates {
		artifact, ok := v.Artifacts[key]
		if !ok {
			continue
		}
		return Resolved{Runtime: reg, Version: v, Artifact: artifact, Platform: key}, nil
	}
	return Resolved{}, devlokerr.New(devlokerr.CodeRuntimeNotFound,
		fmt.Sprintf("%s@%s belum tersedia untuk %s.", name, constraint, key),
		"Versi tersedia: "+strings.Join(versionStrings(reg.Versions), ", "))
}

func (m *Manager) registry(name string) (Registry, bool) {
	for _, r := range m.Registries {
		if r.Name == name {
			return r, true
		}
	}
	return Registry{}, false
}

func (m *Manager) platformKey() string {
	osName := m.OS
	if osName == "" {
		osName = defaultOS
	}
	arch := m.Arch
	if arch == "" {
		arch = defaultArch
	}
	return osName + "/" + arch
}

func matchConstraint(version, constraint string) bool {
	if constraint == "" || constraint == "latest" {
		return true
	}
	if constraint == version {
		return true
	}
	return strings.HasPrefix(version, constraint+".")
}

func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

func versionStrings(versions []Version) []string {
	out := make([]string, 0, len(versions))
	for _, v := range versions {
		out = append(out, v.Version)
	}
	sort.Slice(out, func(i, j int) bool { return compareVersions(out[i], out[j]) > 0 })
	return out
}
