package projects

import (
	"os"
	"path/filepath"
	"testing"

	"devlok/internal/config"
)

func TestAddListRemove(t *testing.T) {
	state := t.TempDir()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, config.FileName), []byte("schema: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(state, project); err != nil {
		t.Fatal(err)
	}
	if err := Add(state, project); err != nil {
		t.Fatal(err)
	}
	list, err := List(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %v", list)
	}
	if err := Remove(state, project); err != nil {
		t.Fatal(err)
	}
	list, err = List(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list after remove = %v", list)
	}
}

func TestAddWithoutManifest(t *testing.T) {
	if err := Add(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("expected error for missing manifest")
	}
}
