//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devlok/internal/config"
	"devlok/internal/projects"
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
	if m.Services["mysql"] != "9.4.0" || m.Web.PHP != "8.4.27" || m.Runtime["nginx"] != "1.30.6" {
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

func TestRemoveRuntimeGuard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: project, Name: "removeguard"}); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if err := a.SetServiceVersion(project, "php", "8.4.26"); err != nil {
		t.Fatal(err)
	}
	saveLastProject(project)
	if err := projects.Add(filepath.Join(home, ".devlok"), project); err != nil {
		t.Fatal(err)
	}

	for _, version := range []string{"8.4.26", "8.4.25"} {
		writeRuntimeMarker(t, home, "php", version)
	}
	list, err := a.RuntimeList()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if r.Name == "php" && r.Version == "8.4.26" && (!r.InUse || len(r.UsedBy) != 1 || r.UsedBy[0] != project) {
			t.Fatalf("runtime usage = %+v", r)
		}
	}
	if _, err := a.RemoveRuntime("php", "8.4.26"); err == nil {
		t.Fatal("expected guard error for runtime in use")
	}
	if _, err := a.RemoveRuntime("php", "8.4.25"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".devlok", "runtimes", "php", "8.4.25")); !os.IsNotExist(err) {
		t.Fatal("unused runtime not removed")
	}
}

func writeRuntimeMarker(t *testing.T, home, name, version string) {
	t.Helper()
	dir := filepath.Join(home, ".devlok", "runtimes", name, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := `{"schema":1,"name":"` + name + `","version":"` + version + `","sha256":"` + strings.Repeat("a", 64) + `","os":"windows","arch":"amd64","installed_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, ".devlok-runtime.json"), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupUnusedRuntimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Init(config.InitOptions{Dir: project, Name: "cleanup"}); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if err := a.SetServiceVersion(project, "php", "8.4.26"); err != nil {
		t.Fatal(err)
	}
	saveLastProject(project)
	if err := projects.Add(filepath.Join(home, ".devlok"), project); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"8.4.25", "8.4.26", "8.5.12"} {
		writeRuntimeMarker(t, home, "php", version)
	}
	writeRuntimeMarker(t, home, "composer", "2.10.3")
	if _, err := a.CleanupUnusedRuntimes(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".devlok", "runtimes", "php", "8.4.25")); !os.IsNotExist(err) {
		t.Fatal("superseded runtime not removed")
	}
	for _, keep := range [][2]string{{"php", "8.4.26"}, {"php", "8.5.12"}, {"composer", "2.10.3"}} {
		if _, err := os.Stat(filepath.Join(home, ".devlok", "runtimes", keep[0], keep[1])); err != nil {
			t.Fatalf("runtime %s %s should remain: %v", keep[0], keep[1], err)
		}
	}
}

func TestRuntimeCacheCleanup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".devlok", "cache", "downloads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "php.zip"), []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	cache, err := a.RuntimeCache()
	if err != nil {
		t.Fatal(err)
	}
	if cache.Files != 1 || cache.Bytes != int64(len("archive")) {
		t.Fatalf("cache = %+v", cache)
	}
	if _, err := a.CleanupRuntimeCache(); err != nil {
		t.Fatal(err)
	}
	cache, err = a.RuntimeCache()
	if err != nil {
		t.Fatal(err)
	}
	if cache.Files != 0 || cache.Bytes != 0 {
		t.Fatalf("cache after cleanup = %+v", cache)
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
