package envarchive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExport(t *testing.T) {
	dir := t.TempDir()
	manifest := "schema: 1\nname: demo\nweb:\n  root: public\n"
	if err := os.WriteFile(filepath.Join(dir, ".devlok.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".devlok.lock"), []byte("schema: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "db", "seed.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "env.zip")
	got, err := Export(dir, out)
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(got)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	for _, want := range []string{".devlok.yaml", ".devlok.lock", "db/seed.sql"} {
		if !names[want] {
			t.Fatalf("missing %s in archive: %v", want, names)
		}
	}
	for name := range names {
		if name == ".env" || filepath.Ext(name) == ".key" {
			t.Fatalf("secret file leaked: %s", name)
		}
	}
}

func TestExportMissingManifest(t *testing.T) {
	if _, err := Export(t.TempDir(), ""); err == nil {
		t.Fatal("expected error")
	}
}
