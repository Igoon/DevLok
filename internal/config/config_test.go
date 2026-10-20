package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"devlok/internal/devlokerr"
)

func writeIndex(t *testing.T, dir, sub string) {
	t.Helper()
	root := filepath.Join(dir, sub)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func parseErr(t *testing.T, data string, base string) error {
	t.Helper()
	_, err := Parse([]byte(data), base)
	if err == nil {
		t.Fatal("expected error")
	}
	return err
}

func TestParseDefaultsAndRootDetection(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	data := `
schema: 1
name: siso
runtime:
  php: "8.4"
`
	m, err := Parse([]byte(data), dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Web.Root != "public" {
		t.Fatalf("root = %q", m.Web.Root)
	}
	if m.Web.Server != "nginx" || m.Web.TLD != "test" {
		t.Fatalf("defaults not applied: %+v", m.Web)
	}
	if !m.HTTPSEnabled() {
		t.Fatal("https should default true")
	}
	if m.DB.Name != "siso" {
		t.Fatalf("db name = %q", m.DB.Name)
	}
	if m.Services["mysql"] != "8.4" {
		t.Fatalf("mysql default = %q", m.Services["mysql"])
	}
	if m.Web.PHP != "8.4" {
		t.Fatalf("php default = %q", m.Web.PHP)
	}
	if m.BaseDomain() != "siso.test" {
		t.Fatalf("domain = %q", m.BaseDomain())
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	err := parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\nfoo: bar\n", dir)
	e, ok := devlokerr.As(err)
	if !ok || e.Code != devlokerr.CodeInvalidManifest {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestExtensionFieldAllowed(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	data := "schema: 1\nname: siso\nx-team: platform\nweb:\n  root: public\n  x-note: ok\n"
	if _, err := Parse([]byte(data), dir); err != nil {
		t.Fatalf("extension should be allowed: %v", err)
	}
}

func TestRootTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	err := parseErr(t, "schema: 1\nname: siso\nweb:\n  root: ../etc\n", dir)
	e, _ := devlokerr.As(err)
	if e == nil || e.Code != devlokerr.CodeInvalidManifest {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidName(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: Bad Name\nweb:\n  root: public\n", dir)
}

func TestInvalidDomainAlias(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\n  aliases: ['bad_alias']\n", dir)
}

func TestDuplicateDomain(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\n  aliases: ['siso.test']\n", dir)
}

func TestReservedEnvRejected(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\nenv:\n  PATH: /tmp\n", dir)
}

func TestUnknownRuntimeRejected(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\nruntime:\n  ruby: '3'\n", dir)
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("schema: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, ok := Find(sub)
	if !ok || found != path {
		t.Fatalf("Find = %q, %v", found, ok)
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	path, m, err := Init(InitOptions{Dir: dir, Name: "My App"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "my-app" {
		t.Fatalf("name = %q", m.Name)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Init(InitOptions{Dir: dir, Name: "My App"}); err == nil {
		t.Fatal("expected existing manifest error")
	}
	if _, _, err := Init(InitOptions{Dir: dir, Name: "My App", Force: true}); err != nil {
		t.Fatalf("force should overwrite: %v", err)
	}
}

func TestInitMissingDir(t *testing.T) {
	_, _, err := Init(InitOptions{Dir: filepath.Join(t.TempDir(), "nope")})
	if err == nil {
		t.Fatal("expected error")
	}
	var e *devlokerr.Error
	if !errors.As(err, &e) {
		t.Fatalf("wrong error type: %v", err)
	}
}

func TestDomainPortsComponents(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	data := `
schema: 1
name: siso
web:
  root: public
  domain: app.local
ports:
  http: 8081
  https: 8444
  mysql: 3307
  php: 19000
components:
  phpmyadmin: false
`
	m, err := Parse([]byte(data), dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.BaseDomain() != "app.local" || m.Domains()[0] != "app.local" {
		t.Fatalf("domain = %q", m.BaseDomain())
	}
	if m.Ports.HTTP != 8081 || m.Ports.MySQL != 3307 || m.Ports.PHP != 19000 {
		t.Fatalf("ports = %+v", m.Ports)
	}
	if m.ComponentEnabled("phpmyadmin") {
		t.Fatal("phpmyadmin should be disabled")
	}
	if !m.ComponentEnabled("mysql") {
		t.Fatal("mysql should default enabled")
	}
}

func TestInvalidPorts(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\nports:\n  http: 70000\n", dir)
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\nports:\n  http: 8080\n  https: 8080\n", dir)
}

func TestUnknownComponent(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\ncomponents:\n  redis: true\n", dir)
}

func TestInvalidDomainOverride(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir, "public")
	parseErr(t, "schema: 1\nname: siso\nweb:\n  root: public\n  domain: bad_domain\n", dir)
}
