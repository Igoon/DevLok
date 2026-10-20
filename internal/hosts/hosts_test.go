package hosts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newManager(t *testing.T) (*Manager, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts")
	return &Manager{Path: path}, path
}

func TestApplyPreservesAndIsIdempotent(t *testing.T) {
	m, path := newManager(t)
	original := "127.0.0.1 localhost\n# existing\n10.0.0.5 old.local\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := m.Apply([]string{"siso.test", "www.siso.test"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || len(res.Added) != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, "10.0.0.5 old.local") {
		t.Fatal("existing entry lost")
	}
	if strings.Count(text, "127.0.0.1 siso.test") != 1 || strings.Count(text, "127.0.0.1 www.siso.test") != 1 {
		t.Fatalf("duplicate entry: %s", text)
	}
	if _, err := m.Apply([]string{"siso.test"}); err != nil {
		t.Fatal(err)
	}
}

func TestRemove(t *testing.T) {
	m, path := newManager(t)
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Apply([]string{"a.test", "b.test"}); err != nil {
		t.Fatal(err)
	}
	res, err := m.Remove()
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || len(res.Removed) != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "a.test") || strings.Contains(string(data), StartMarker) {
		t.Fatalf("block not removed: %s", data)
	}
}

func TestConflictOutsideBlock(t *testing.T) {
	m, path := newManager(t)
	if err := os.WriteFile(path, []byte("127.0.0.1 siso.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := m.Apply([]string{"siso.test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || len(res.Added) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestCRLFPreserved(t *testing.T) {
	m, path := newManager(t)
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Apply([]string{"siso.test"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "\r\n") {
		t.Fatal("CRLF not preserved")
	}
}

func TestBrokenBlock(t *testing.T) {
	m, path := newManager(t)
	if err := os.WriteFile(path, []byte(StartMarker+"\n127.0.0.1 a.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Apply([]string{"b.test"}); err == nil {
		t.Fatal("expected broken block error")
	}
}
