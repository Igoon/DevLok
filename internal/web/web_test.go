package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(t *testing.T) (Config, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "www", "siso")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	prefix := filepath.Join(base, "nginx")
	confDir := filepath.Join(prefix, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "fastcgi_params"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "mime.types"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		PrefixDir: prefix,
		ConfPath:  filepath.Join(base, "generated", "nginx.conf"),
		SitesDir:  filepath.Join(base, "generated", "sites"),
		LogsDir:   filepath.Join(base, "logs"),
		MimeTypes: filepath.Join(confDir, "mime.types"),
		HTTPPort:  80,
		HTTPSPort: 443,
		Servers: []Server{{
			Name:        "siso",
			Domain:      "siso.test",
			Aliases:     []string{"www.siso.test"},
			Root:        root,
			PHPUpstream: "127.0.0.1:10987",
			HTTPS:       true,
			CertFile:    filepath.Join(base, "certs", "siso.crt"),
			KeyFile:     filepath.Join(base, "certs", "siso.key"),
		}},
	}
	return cfg, base
}

func TestGenerateHTTPS(t *testing.T) {
	cfg, _ := testConfig(t)
	if err := Generate(cfg); err != nil {
		t.Fatal(err)
	}
	site, err := os.ReadFile(filepath.Join(cfg.SitesDir, "siso.conf"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(site)
	for _, want := range []string{
		"listen 443 ssl;",
		"http2 on;",
		"ssl_protocols TLSv1.2 TLSv1.3;",
		"fastcgi_pass 127.0.0.1:10987;",
		"server_name siso.test www.siso.test;",
		"return 301 https://$host$request_uri;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "TLSv1.1") || strings.Contains(text, "TLSv1 ") {
		t.Fatalf("weak TLS present:\n%s", text)
	}
	main, err := os.ReadFile(cfg.ConfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(main), filepath.ToSlash(cfg.SitesDir)+"/*.conf") {
		t.Fatalf("sites include missing:\n%s", main)
	}
}

func TestNonStandardPortsAndHTTPOnly(t *testing.T) {
	cfg, _ := testConfig(t)
	cfg.HTTPPort = 8080
	cfg.HTTPSPort = 8443
	cfg.Servers[0].HTTPS = false
	cfg.Servers[0].CertFile = ""
	cfg.Servers[0].KeyFile = ""
	if err := Generate(cfg); err != nil {
		t.Fatal(err)
	}
	site, _ := os.ReadFile(filepath.Join(cfg.SitesDir, "siso.conf"))
	text := string(site)
	if !strings.Contains(text, "listen 8080;") {
		t.Fatalf("http listener missing:\n%s", text)
	}
	if strings.Contains(text, "ssl") || strings.Contains(text, "8443") {
		t.Fatalf("unexpected https config:\n%s", text)
	}
	cfg.Servers[0].HTTPS = true
	cfg.Servers[0].CertFile = "c.crt"
	cfg.Servers[0].KeyFile = "c.key"
	cfg.HTTPSPort = 8443
	if err := Generate(cfg); err != nil {
		t.Fatal(err)
	}
	site, _ = os.ReadFile(filepath.Join(cfg.SitesDir, "siso.conf"))
	if !strings.Contains(string(site), "https://$host:8443$request_uri") {
		t.Fatalf("redirect port missing:\n%s", site)
	}
}

func TestGenerateProxy(t *testing.T) {
	cfg, _ := testConfig(t)
	cfg.Servers = []Server{{
		Name:   "mailpit",
		Domain: "mailpit.test",
		Proxy:  "127.0.0.1:8025",
		HTTPS:  false,
	}}
	if err := Generate(cfg); err != nil {
		t.Fatal(err)
	}
	site, err := os.ReadFile(filepath.Join(cfg.SitesDir, "mailpit.conf"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(site)
	for _, want := range []string{
		"server_name mailpit.test;",
		"proxy_pass http://127.0.0.1:8025;",
		"proxy_http_version 1.1;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "fastcgi_pass") {
		t.Fatalf("proxy site should not use fastcgi:\n%s", text)
	}
}

func TestCleanupStaleSite(t *testing.T) {
	cfg, _ := testConfig(t)
	if err := os.MkdirAll(cfg.SitesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(cfg.SitesDir, "old.conf")
	if err := os.WriteFile(stale, []byte("server {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale site not removed")
	}
}

func TestMissingRoot(t *testing.T) {
	cfg, _ := testConfig(t)
	cfg.Servers[0].Root = filepath.Join(t.TempDir(), "missing")
	if err := Generate(cfg); err == nil {
		t.Fatal("expected missing root error")
	}
}
