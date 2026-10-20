package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuoteIdent(t *testing.T) {
	if got := quoteIdent("siso"); got != "`siso`" {
		t.Fatalf("got %s", got)
	}
	if got := quoteIdent("a`b"); got != "`a``b`" {
		t.Fatalf("got %s", got)
	}
}

func TestDBUser(t *testing.T) {
	cases := map[string]string{
		"siso":   "siso",
		"My App": "my_app",
		"123":    "db_123",
		"a-b.c":  "a_b_c",
		"":       "db_",
	}
	for in, want := range cases {
		if got := dbUser(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
	if len(dbUser(strings.Repeat("a", 80))) > 32 {
		t.Fatal("user too long")
	}
}

func TestRandomPassword(t *testing.T) {
	pw, err := randomPassword(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(pw) != 32 {
		t.Fatalf("len = %d", len(pw))
	}
	if strings.ContainsAny(pw, "'\"` ") {
		t.Fatalf("unsafe character in password: %s", pw)
	}
}

func TestCredsSealed(t *testing.T) {
	state := t.TempDir()
	m, err := New(Options{StateDir: state, Version: "8.4"})
	if err != nil {
		t.Fatal(err)
	}
	c := credentials{
		RootPassword: "rootsecret",
		Databases: map[string]DatabaseCreds{
			"siso": {Project: "siso", Name: "siso", User: "siso", Password: "dbsecret"},
		},
	}
	if err := m.saveCreds(c); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(m.credsPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "dbsecret") || strings.Contains(string(raw), "rootsecret") {
		t.Fatal("credentials stored in plaintext")
	}
	got, err := m.loadCreds()
	if err != nil {
		t.Fatal(err)
	}
	if got.RootPassword != "rootsecret" || got.Databases["siso"].Password != "dbsecret" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestInstanceState(t *testing.T) {
	state := t.TempDir()
	m, err := New(Options{StateDir: state, Version: "8.4"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.loadState(); !os.IsNotExist(err) {
		t.Fatalf("expected not exist, got %v", err)
	}
	st := InstanceState{Schema: 1, Version: "8.4", Port: 3306, DataDir: m.DataDir, Initialized: true}
	if err := m.saveState(st); err != nil {
		t.Fatal(err)
	}
	got, err := m.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != 3306 || !got.Initialized {
		t.Fatalf("unexpected state: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(state, "state", "mysql-8.4.json")); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidVersion(t *testing.T) {
	if _, err := New(Options{StateDir: t.TempDir(), Version: "x"}); err == nil {
		t.Fatal("expected version error")
	}
}
