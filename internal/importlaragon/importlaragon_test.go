package importlaragon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "www", "demo")
	if err := os.MkdirAll(filepath.Join(project, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "public", "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	confDir := filepath.Join(dir, "etc", "nginx", "sites-enabled")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := `server {
    listen 8080;
    server_name demo.test www.demo.test;
    root "` + filepath.ToSlash(filepath.Join(project, "public")) + `";
}
`
	if err := os.WriteFile(filepath.Join(confDir, "auto.demo.test.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPlanAndImport(t *testing.T) {
	dir := fixture(t)
	sites, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 {
		t.Fatalf("sites = %+v", sites)
	}
	s := sites[0]
	if s.Name != "demo" || s.Domain != "demo.test" || s.Root != "public" {
		t.Fatalf("unexpected site: %+v", s)
	}
	manifestPath := filepath.Join(s.ProjectDir, ".devlok.yaml")
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatal("dry-run should not write")
	}
	res, err := Import(sites, true)
	if err != nil || len(res) != 1 || res[0].Action != "create" {
		t.Fatalf("dry-run result: %+v %v", res, err)
	}
	res, err = Import(sites, false)
	if err != nil || len(res) != 1 || res[0].Action != "create" {
		t.Fatalf("import result: %+v %v", res, err)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: demo") {
		t.Fatalf("manifest: %s", data)
	}
	res, _ = Import(sites, false)
	if res[0].Action != "skip" {
		t.Fatalf("second import should skip: %+v", res)
	}
}

func TestPlanSkipsWildcardOnly(t *testing.T) {
	dir := fixture(t)
	confDir := filepath.Join(dir, "etc", "nginx", "sites-enabled")
	conf := "server {\n server_name *.test;\n root \"" + filepath.ToSlash(filepath.Join(dir, "www", "demo", "public")) + "\";\n}\n"
	if err := os.WriteFile(filepath.Join(confDir, "auto.wild.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(confDir, "auto.demo.test.conf"))
	sites, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 || sites[0].Source != "www-scan" {
		t.Fatalf("expected www scan: %+v", sites)
	}
}

func TestValidDomainGuard(t *testing.T) {
	dir := fixture(t)
	confDir := filepath.Join(dir, "etc", "nginx", "sites-enabled")
	_ = os.Remove(filepath.Join(confDir, "auto.demo.test.conf"))
	if err := os.WriteFile(filepath.Join(confDir, "auto.bad.conf"), []byte("server {\n server_name bad_domain!;\n root \"/nope\";\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sites, err := Plan(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sites {
		if s.Domain == "bad_domain!" {
			t.Fatal("invalid domain accepted")
		}
	}
}
