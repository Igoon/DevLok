package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock.yaml")
	l := &Lock{
		Schema:      LockSchemaVersion,
		GeneratedAt: time.Now().UTC().Truncate(time.Second),
		Platform:    "windows/amd64",
		Name:        "demo",
		Web: LockedWeb{
			Server:  "nginx",
			Version: "1.30.6",
			SHA256:  "abc",
			URL:     "https://example.com/nginx.zip",
		},
		Runtime:  map[string]LockedRuntime{"php": {Version: "8.4.26", SHA256: "def", URL: "https://example.com/php.zip"}},
		Services: map[string]LockedRuntime{"mysql": {Version: "8.4.10", SHA256: "ghi", URL: "https://example.com/mysql.zip"}},
	}
	if err := SaveLock(path, l); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo" || got.Runtime["php"].Version != "8.4.26" || got.Web.Version != "1.30.6" {
		t.Fatalf("unexpected lock: %+v", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestCompareLock(t *testing.T) {
	m := &Manifest{
		Name:     "demo",
		Web:      Web{PHP: "8.4"},
		Runtime:  map[string]string{"php": "8.4", "node": "22"},
		Services: map[string]string{"mysql": "8.4"},
	}
	l := &Lock{
		Name:     "demo",
		Runtime:  map[string]LockedRuntime{"php": {Version: "8.4.26"}, "node": {Version: "22.23.3"}},
		Services: map[string]LockedRuntime{"mysql": {Version: "8.4.10"}},
	}
	if w := CompareLock(m, l); len(w) != 0 {
		t.Fatalf("unexpected warnings: %v", w)
	}
	m.Web.PHP = "8.3"
	m.Runtime["php"] = "8.3"
	if w := CompareLock(m, l); len(w) != 1 {
		t.Fatalf("expected php warning, got %v", w)
	}
}

func TestLoadLockMissing(t *testing.T) {
	if _, err := LoadLock(filepath.Join(t.TempDir(), "missing")); !os.IsNotExist(err) {
		t.Fatalf("expected not exist, got %v", err)
	}
}
