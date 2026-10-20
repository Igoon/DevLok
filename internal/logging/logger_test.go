package logging

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRedact(t *testing.T) {
	in := map[string]any{
		"password": "secret",
		"user":     "siso",
		"nested": map[string]any{
			"api_token": "abc",
			"port":      3306,
		},
		"items": []any{
			map[string]any{"ca_key": "pem"},
			"plain",
		},
	}
	out := Redact(in)
	if out["password"] != "[REDACTED]" {
		t.Fatalf("password not redacted: %v", out["password"])
	}
	if out["user"] != "siso" {
		t.Fatalf("user altered: %v", out["user"])
	}
	nested := out["nested"].(map[string]any)
	if nested["api_token"] != "[REDACTED]" {
		t.Fatalf("nested token not redacted: %v", nested["api_token"])
	}
	if nested["port"] != 3306 {
		t.Fatalf("port altered: %v", nested["port"])
	}
	items := out["items"].([]any)
	if items[0].(map[string]any)["ca_key"] != "[REDACTED]" {
		t.Fatalf("slice token not redacted")
	}
	if items[1] != "plain" {
		t.Fatalf("plain value altered")
	}
}

func TestLoggerWritesJSONAndRotates(t *testing.T) {
	dir := t.TempDir()
	l, err := New(Options{Dir: dir, Service: "test", MaxBytes: 400, MaxFiles: 2})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		l.Info("tick", "hello", map[string]any{"i": i, "password": "x"})
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "test.jsonl")
	f, err := os.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	count := 0
	for sc.Scan() {
		var e entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("invalid json line: %v", err)
		}
		if e.Level != "info" || e.Service != "test" {
			t.Fatalf("unexpected entry: %+v", e)
		}
		if e.Fields["password"] != "[REDACTED]" {
			t.Fatalf("password not redacted in log")
		}
		count++
	}
	if count == 0 {
		t.Fatal("no entries in current log")
	}
	if _, err := os.Stat(base + ".1"); err != nil {
		t.Fatalf("expected rotated file: %v", err)
	}
	info, err := os.Stat(base)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 400 {
		t.Fatalf("current log exceeds max size: %d", info.Size())
	}
}
