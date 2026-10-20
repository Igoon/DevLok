//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"devlok/internal/config"
)

func TestComponentsRoundtrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	c, err := a.Components(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Database || !c.PHPMyAdmin {
		t.Fatalf("default components should be on: %+v", c)
	}
	if err := a.SetComponents(dir, false, true); err != nil {
		t.Fatal(err)
	}
	c, err = a.Components(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Database || c.PHPMyAdmin {
		t.Fatalf("both should be off: %+v", c)
	}
	if err := a.SetComponents(dir, true, true); err != nil {
		t.Fatal(err)
	}
	c, _ = a.Components(dir)
	if !c.Database || !c.PHPMyAdmin {
		t.Fatalf("both should be on: %+v", c)
	}
}

func TestMySQLVersionSelection(t *testing.T) {
	a := NewApp()
	versions, err := a.MySQLVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) < 2 {
		t.Fatalf("expected multiple MySQL versions, got %v", versions)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: "dbver"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetMySQLVersion(dir, "9.4.0"); err != nil {
		t.Fatal(err)
	}
	m, err := loadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Services["mysql"] != "9.4.0" {
		t.Fatalf("mysql version = %q", m.Services["mysql"])
	}
}

func TestServiceVersionSelection(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: "svcver"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetServiceVersion(dir, "nginx", "1.30.6"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetServiceVersion(dir, "php", "8.3"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetServiceVersion(dir, "node", "22"); err != nil {
		t.Fatal(err)
	}
	m, err := loadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Runtime["nginx"] != "1.30.6" {
		t.Fatalf("nginx = %q", m.Runtime["nginx"])
	}
	if m.Web.PHP != "8.3.35" || m.Runtime["php"] != "8.3.35" {
		t.Fatalf("php = %q / %q", m.Web.PHP, m.Runtime["php"])
	}
	if m.Runtime["node"] != "22.23.3" {
		t.Fatalf("node = %q", m.Runtime["node"])
	}
}

func TestProjectConfigSave(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: "cfgapp"}); err != nil {
		t.Fatal(err)
	}
	err := a.SaveProjectConfig(dir, ProjectConfigInput{
		Domain:     "app.local",
		Root:       "public",
		HTTPS:      true,
		DBName:     "cfgapp",
		Aliases:    []string{"www.app.local"},
		HTTPPort:   8091,
		HTTPSPort:  8451,
		MySQLPort:  3307,
		PHPPort:    19001,
		Database:   true,
		PHPMyAdmin: true,
		Versions: map[string]string{
			"nginx": "1.30.6",
			"php":   "8.4",
			"mysql": "9.4",
			"node":  "22",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := loadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.BaseDomain() != "app.local" || m.Ports.HTTP != 8091 || m.Ports.MySQL != 3307 {
		t.Fatalf("config not saved: %+v", m)
	}
	if len(m.Web.Aliases) != 1 || m.Web.Aliases[0] != "www.app.local" {
		t.Fatalf("aliases = %v", m.Web.Aliases)
	}
	if m.Services["mysql"] != "9.4.0" || m.Web.PHP != "8.4.26" || m.Runtime["nginx"] != "1.30.6" {
		t.Fatalf("versions = %+v", m)
	}
	view, err := a.ProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if view.Domain != "app.local" || view.HTTPPort != 8091 || view.Versions["mysql"] != "9.4.0" {
		t.Fatalf("view = %+v", view)
	}
}

func TestImportLaragonBackend(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	laragon := t.TempDir()
	project := filepath.Join(laragon, "www", "demo")
	if err := os.MkdirAll(filepath.Join(project, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	confDir := filepath.Join(laragon, "etc", "nginx", "sites-enabled")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "server {\n server_name demo.test;\n root \"" + filepath.ToSlash(filepath.Join(project, "public")) + "\";\n}\n"
	if err := os.WriteFile(filepath.Join(confDir, "auto.demo.test.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	sites, err := a.ScanLaragon(laragon)
	if err != nil || len(sites) != 1 {
		t.Fatalf("scan: %v %v", sites, err)
	}
	results, err := a.ImportLaragon(laragon, []string{"demo"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Action != "create" {
		t.Fatalf("results = %+v", results)
	}
	if _, err := os.Stat(filepath.Join(project, config.FileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".devlok", "state", "projects.json")); err != nil {
		t.Fatal("project not adopted")
	}
}
