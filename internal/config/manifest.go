package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"devlok/internal/devlokerr"
	"gopkg.in/yaml.v3"
)

const (
	SchemaVersion = 1
	MaxSize       = 1 << 20
	FileName      = ".devlok.yaml"
	LockFileName  = ".devlok.lock"
)

type Manifest struct {
	Schema     int               `yaml:"schema" json:"schema"`
	Name       string            `yaml:"name" json:"name"`
	Web        Web               `yaml:"web" json:"web"`
	Runtime    map[string]string `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Services   map[string]string `yaml:"services,omitempty" json:"services,omitempty"`
	DB         Database          `yaml:"db,omitempty" json:"db"`
	Ports      PortsConfig       `yaml:"ports,omitempty" json:"ports,omitempty"`
	Components map[string]bool   `yaml:"components,omitempty" json:"components,omitempty"`
	Env        map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Extensions map[string]any    `yaml:",inline" json:"-"`
}

type PortsConfig struct {
	HTTP  int `yaml:"http,omitempty" json:"http,omitempty"`
	HTTPS int `yaml:"https,omitempty" json:"https,omitempty"`
	MySQL int `yaml:"mysql,omitempty" json:"mysql,omitempty"`
	PHP   int `yaml:"php,omitempty" json:"php,omitempty"`
}

type Web struct {
	Server     string         `yaml:"server,omitempty" json:"server"`
	Root       string         `yaml:"root,omitempty" json:"root"`
	Domain     string         `yaml:"domain,omitempty" json:"domain,omitempty"`
	TLD        string         `yaml:"tld,omitempty" json:"tld"`
	Aliases    []string       `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	PHP        string         `yaml:"php,omitempty" json:"php"`
	HTTPS      *bool          `yaml:"https,omitempty" json:"https"`
	Extensions map[string]any `yaml:",inline" json:"-"`
}

type Database struct {
	Name       string         `yaml:"name,omitempty" json:"name"`
	Charset    string         `yaml:"charset,omitempty" json:"charset"`
	Collation  string         `yaml:"collation,omitempty" json:"collation"`
	Seed       string         `yaml:"seed,omitempty" json:"seed,omitempty"`
	BackupDir  string         `yaml:"backup_dir,omitempty" json:"backup_dir,omitempty"`
	Extensions map[string]any `yaml:",inline" json:"-"`
}

func (m *Manifest) HTTPSEnabled() bool {
	if m.Web.HTTPS == nil {
		return true
	}
	return *m.Web.HTTPS
}

func (m *Manifest) BaseDomain() string {
	if m.Web.Domain != "" {
		return strings.ToLower(m.Web.Domain)
	}
	return m.Name + "." + m.Web.TLD
}

func (m *Manifest) Domains() []string {
	out := []string{m.BaseDomain()}
	out = append(out, m.Web.Aliases...)
	return out
}

func (m *Manifest) ComponentEnabled(name string) bool {
	if m.Components == nil {
		return true
	}
	enabled, ok := m.Components[name]
	if !ok {
		return true
	}
	return enabled
}

var (
	nameRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	tldRe     = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)
	versionRe = regexp.MustCompile(`^[0-9]{1,2}(\.[0-9]{1,3}){0,2}$`)
	dbNameRe  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	labelRe   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, devlokerr.Wrap(devlokerr.CodeManifestNotFound, err, "Manifest tidak ditemukan.", "Jalankan dl init atau tentukan path manifest.")
		}
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidManifest, err, "Manifest tidak dapat dibaca.", "Periksa izin file.")
	}
	return Parse(data, filepath.Dir(path))
}

func Parse(data []byte, baseDir string) (*Manifest, error) {
	if len(data) > MaxSize {
		return nil, devlokerr.New(devlokerr.CodeInvalidManifest, "Manifest terlalu besar.", fmt.Sprintf("Ukuran maksimum %d byte.", MaxSize))
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidManifest, err, "YAML tidak valid.", "Periksa indentasi dan sintaks YAML.")
	}
	ApplyDefaults(&m)
	if m.Web.Root == "" && baseDir != "" {
		m.Web.Root = DetectRoot(baseDir)
	}
	if err := Validate(&m, baseDir); err != nil {
		return nil, err
	}
	return &m, nil
}

func Find(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func Marshal(m *Manifest) ([]byte, error) {
	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Manifest tidak dapat diserialisasi.", "")
	}
	return data, nil
}

func SanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "app"
	}
	if len(out) > 63 {
		out = strings.Trim(out[:63], "-")
	}
	if out[0] < 'a' || out[0] > 'z' {
		out = "app-" + out
	}
	return out
}

func DetectRoot(baseDir string) string {
	candidates := []string{"public", "web", "www", "."}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(baseDir, c, "index.php")); err == nil {
			return c
		}
	}
	return ""
}
