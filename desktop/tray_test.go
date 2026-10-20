//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteNginxQuickReplacesManagedBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	a := &App{}
	if err := a.writeNginxQuick("demo", "128M", "300"); err != nil {
		t.Fatal(err)
	}
	if err := a.writeNginxQuick("demo", "64M", "120"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".devlok", "config", "nginx", "demo.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, nginxQuickStart) != 1 {
		t.Fatalf("managed block tidak idempotent:\n%s", text)
	}
	if !strings.Contains(text, "client_max_body_size 64m;") || !strings.Contains(text, "fastcgi_read_timeout 120s;") {
		t.Fatalf("nilai quick settings tidak tersimpan:\n%s", text)
	}
	if strings.Contains(text, "128m") || strings.Contains(text, "300s") {
		t.Fatalf("nilai lama masih tersisa:\n%s", text)
	}
}

func TestStripPHPQuickBlock(t *testing.T) {
	in := "memory_limit = 128M\n" + phpQuickStart + "\nmemory_limit = 512M\n" + phpQuickEnd + "\nmax_execution_time = 300\n"
	out := stripPHPQuickBlock(in)
	if strings.Contains(out, phpQuickStart) || strings.Contains(out, "512M") {
		t.Fatalf("block tidak terhapus:\n%s", out)
	}
	if !strings.Contains(out, "memory_limit = 128M") || !strings.Contains(out, "max_execution_time = 300") {
		t.Fatalf("konten pengguna hilang:\n%s", out)
	}
}
