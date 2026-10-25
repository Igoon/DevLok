package runtime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devlok/internal/devlokerr"
)

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testRegistry(rawURL, hash string, size int64) []Registry {
	return []Registry{{
		Schema:  1,
		Name:    "tool",
		Display: "Tool",
		Versions: []Version{{
			Version: "1.0.0",
			Artifacts: map[string]Artifact{
				"windows/amd64": {
					URL:         rawURL,
					SHA256:      hash,
					Size:        size,
					Archive:     ArchiveZip,
					Executables: []string{"tool.exe"},
				},
			},
		}},
	}}
}

func newTestManager(t *testing.T, regs []Registry, client *http.Client) *Manager {
	t.Helper()
	m, err := NewManager(Options{
		Root:       filepath.Join(t.TempDir(), "runtimes"),
		Cache:      filepath.Join(t.TempDir(), "cache"),
		HTTP:       client,
		OS:         "windows",
		Arch:       "amd64",
		Registries: regs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestResolveVersions(t *testing.T) {
	m := newTestManager(t, nil, nil)
	m.Registries = []Registry{{
		Schema: 1, Name: "php", Display: "PHP",
		Versions: []Version{
			{Version: "8.4.23", Artifacts: map[string]Artifact{"windows/amd64": {}}},
			{Version: "8.4.26", Artifacts: map[string]Artifact{"windows/amd64": {}}},
			{Version: "8.5.11", Artifacts: map[string]Artifact{"windows/amd64": {}}},
			{Version: "8.3.35", Artifacts: map[string]Artifact{"windows/amd64": {}}},
		},
	}}
	res, err := m.resolve("php", "8.4")
	if err != nil {
		t.Fatal(err)
	}
	if res.Version.Version != "8.4.26" {
		t.Fatalf("got %s", res.Version.Version)
	}
	res, err = m.resolve("php", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Version.Version != "8.5.11" {
		t.Fatalf("latest got %s", res.Version.Version)
	}
	res, err = m.resolve("php", "8.3.35")
	if err != nil || res.Version.Version != "8.3.35" {
		t.Fatalf("exact: %v %v", res, err)
	}
	if _, err := m.resolve("php", "7.4"); err == nil {
		t.Fatal("expected missing version error")
	}
	if _, err := m.resolve("ruby", ""); err == nil {
		t.Fatal("expected unknown runtime error")
	}
}

func TestResolvePlatformMissing(t *testing.T) {
	m := newTestManager(t, nil, nil)
	m.Registries = []Registry{{
		Schema: 1, Name: "php",
		Versions: []Version{{Version: "8.4.26", Artifacts: map[string]Artifact{"darwin/arm64": {}}}},
	}}
	_, err := m.resolve("php", "8.4")
	if err == nil {
		t.Fatal("expected platform error")
	}
	e, ok := devlokerr.As(err)
	if !ok || e.Code != devlokerr.CodeRuntimeNotFound {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestInstallVerifyRemove(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"tool-1.0.0/tool.exe":    "binary",
		"tool-1.0.0/README.txt":  "hello",
		"tool-1.0.0/sub/data.db": "data",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "tool.zip", time.Now(), bytes.NewReader(zipData))
	}))
	defer server.Close()

	m := newTestManager(t, testRegistry(server.URL+"/tool.zip", sha(zipData), int64(len(zipData))), server.Client())
	inst, downloaded, err := m.Install(context.Background(), "tool", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Fatal("first install should download")
	}
	if _, err := os.Stat(filepath.Join(inst.Path, "tool.exe")); err != nil {
		t.Fatalf("executable missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inst.Path, "sub", "data.db")); err != nil {
		t.Fatalf("data missing: %v", err)
	}
	if err := m.Verify("tool", "1.0"); err != nil {
		t.Fatalf("verify: %v", err)
	}

	inst2, downloaded, err := m.Install(context.Background(), "tool", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if downloaded {
		t.Fatal("second install should use installed marker")
	}
	if inst2.Path != inst.Path {
		t.Fatalf("path mismatch: %s %s", inst.Path, inst2.Path)
	}

	list, err := m.ListInstalled()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
	if err := m.Remove("tool", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := m.Verify("tool", "1.0.0"); err == nil {
		t.Fatal("verify should fail after remove")
	}
}

func TestInstallFallsBackToMirror(t *testing.T) {
	zipData := buildZip(t, map[string]string{"tool-1.0.0/tool.exe": "binary"})
	fallbackHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing.zip" {
			http.NotFound(w, r)
			return
		}
		fallbackHits++
		http.ServeContent(w, r, "tool.zip", time.Now(), bytes.NewReader(zipData))
	}))
	defer server.Close()

	reg := testRegistry(server.URL+"/missing.zip", sha(zipData), int64(len(zipData)))
	artifact := reg[0].Versions[0].Artifacts["windows/amd64"]
	artifact.URLFallbacks = []string{server.URL + "/tool.zip"}
	reg[0].Versions[0].Artifacts["windows/amd64"] = artifact

	m := newTestManager(t, reg, server.Client())
	inst, downloaded, err := m.Install(context.Background(), "tool", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Fatal("install should download from fallback")
	}
	if fallbackHits != 1 {
		t.Fatalf("fallback hits = %d, want 1", fallbackHits)
	}
	if _, err := os.Stat(filepath.Join(inst.Path, "tool.exe")); err != nil {
		t.Fatalf("executable missing: %v", err)
	}
}

func TestInstallChecksumMismatch(t *testing.T) {
	zipData := buildZip(t, map[string]string{"tool-1.0.0/tool.exe": "binary"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	m := newTestManager(t, testRegistry(server.URL+"/tool.zip", strings.Repeat("0", 64), int64(len(zipData))), server.Client())
	_, _, err := m.Install(context.Background(), "tool", "1.0")
	if err == nil {
		t.Fatal("expected checksum error")
	}
	e, ok := devlokerr.As(err)
	if !ok || e.Code != devlokerr.CodeChecksumMismatch {
		t.Fatalf("wrong error: %v", err)
	}
	if list, _ := m.ListInstalled(); len(list) != 0 {
		t.Fatalf("nothing should be installed: %v", list)
	}
}

func TestInstallOfflineFromCache(t *testing.T) {
	zipData := buildZip(t, map[string]string{"tool-1.0.0/tool.exe": "binary"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	m := newTestManager(t, testRegistry(server.URL+"/tool.zip", sha(zipData), int64(len(zipData))), server.Client())
	inst, _, err := m.Install(context.Background(), "tool", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(inst.Path); err != nil {
		t.Fatal(err)
	}
	server.Close()
	m.Offline = true
	inst2, downloaded, err := m.Install(context.Background(), "tool", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if downloaded {
		t.Fatal("offline install should not download")
	}
	if _, err := os.Stat(filepath.Join(inst2.Path, "tool.exe")); err != nil {
		t.Fatal(err)
	}
}

func TestOfflineWithoutCache(t *testing.T) {
	m := newTestManager(t, testRegistry("https://example.invalid/tool.zip", strings.Repeat("a", 64), 1), nil)
	m.Offline = true
	_, _, err := m.Install(context.Background(), "tool", "1.0")
	if err == nil {
		t.Fatal("expected offline error")
	}
	e, _ := devlokerr.As(err)
	if e == nil || e.Code != devlokerr.CodeDependencyMissing {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestDownloadResume(t *testing.T) {
	zipData := buildZip(t, map[string]string{"tool-1.0.0/tool.exe": "binary-content"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "tool.zip", time.Now(), bytes.NewReader(zipData))
	}))
	defer server.Close()

	reg := testRegistry(server.URL+"/tool.zip", sha(zipData), int64(len(zipData)))
	m := newTestManager(t, reg, server.Client())
	res, err := m.resolve("tool", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	cache := m.cachePath(res)
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	half := zipData[:len(zipData)/2]
	if err := os.WriteFile(cache+".part", half, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.Install(context.Background(), "tool", "1.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache + ".part"); !os.IsNotExist(err) {
		t.Fatalf("part file should be gone: %v", err)
	}
	got, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, zipData) {
		t.Fatalf("resumed file mismatch: got %d bytes want %d", len(got), len(zipData))
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "bad.zip")
	if err := os.WriteFile(zipPath, buildZip(t, map[string]string{"../evil.txt": "evil"}), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(zipPath, dest); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("file escaped destination")
	}
}

func TestExtractZipRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "link.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	hdr := &zip.FileHeader{Name: "link"}
	hdr.SetMode(os.ModeSymlink | 0o777)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("../target"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(zipPath, t.TempDir()); err == nil {
		t.Fatal("expected symlink error")
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "tool.tar.gz")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := map[string]string{
		"tool-1.0.0/tool":    "binary",
		"tool-1.0.0/bin/run": "run",
	}
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatal(err)
	}
	if err := stripSingleRoot(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "tool")); err != nil {
		t.Fatal(err)
	}
	if err := validateExecutables(dest, Artifact{Executables: []string{"tool", "bin/run"}}); err != nil {
		t.Fatal(err)
	}
}

func TestParseRegistryRejectsBadSHA(t *testing.T) {
	data := []byte(`{"schema":1,"name":"x","display":"X","versions":[{"version":"1.0.0","artifacts":{"windows/amd64":{"url":"https://example.com/x.zip","sha256":"nope","archive":"zip","executables":["x.exe"]}}}]}`)
	if _, err := ParseRegistry(data); err == nil {
		t.Fatal("expected invalid sha error")
	}
}

func TestParseRegistryRejectsBadFallbackURL(t *testing.T) {
	data := []byte(`{"schema":1,"name":"x","display":"X","versions":[{"version":"1.0.0","artifacts":{"windows/amd64":{"url":"https://example.com/x.zip","url_fallbacks":["http://example.com/x.zip"],"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","archive":"zip","executables":["x.exe"]}}}]}`)
	if _, err := ParseRegistry(data); err == nil {
		t.Fatal("expected invalid fallback url error")
	}
}

func TestParseRegistryChannel(t *testing.T) {
	okData := []byte(`{"schema":1,"name":"x","display":"X","versions":[{"version":"1.0.0","channel":"stable","recommended":true,"artifacts":{"windows/amd64":{"url":"https://example.com/x.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","archive":"zip","executables":["x.exe"]}}}]}`)
	reg, err := ParseRegistry(okData)
	if err != nil {
		t.Fatalf("valid channel rejected: %v", err)
	}
	if reg.Versions[0].Channel != "stable" || !reg.Versions[0].Recommended {
		t.Fatalf("channel fields not parsed: %+v", reg.Versions[0])
	}
	badData := []byte(`{"schema":1,"name":"x","display":"X","versions":[{"version":"1.0.0","channel":"Stable!","artifacts":{"windows/amd64":{"url":"https://example.com/x.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","archive":"zip","executables":["x.exe"]}}}]}`)
	if _, err := ParseRegistry(badData); err == nil {
		t.Fatal("expected invalid channel error")
	}
	unknownData := []byte(`{"schema":1,"name":"x","display":"X","versions":[{"version":"1.0.0","channel":"shiny","artifacts":{"windows/amd64":{"url":"https://example.com/x.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","archive":"zip","executables":["x.exe"]}}}]}`)
	if _, err := ParseRegistry(unknownData); err == nil {
		t.Fatal("expected unknown channel error")
	}
}

func TestEmbeddedRegistriesChannels(t *testing.T) {
	regs, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) == 0 {
		t.Fatal("no embedded registries")
	}
	for _, reg := range regs {
		recommended := 0
		for _, v := range reg.Versions {
			if v.Recommended {
				recommended++
				if v.Channel == "eol" || v.Channel == "beta" || v.Channel == "rc" || v.Channel == "preview" {
					t.Fatalf("%s %s marks unstable channel as recommended", reg.Name, v.Version)
				}
			}
		}
		if recommended > 1 {
			t.Fatalf("%s has %d recommended versions", reg.Name, recommended)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("8.10.1", "8.9.9") <= 0 {
		t.Fatal("8.10.1 should be greater")
	}
	if compareVersions("8.4", "8.4.0") != 0 {
		t.Fatal("8.4 should equal 8.4.0")
	}
	if matchConstraint("8.4.26", "8.4") == false {
		t.Fatal("constraint should match")
	}
	if matchConstraint("8.4.26", "8.5") {
		t.Fatal("constraint should not match")
	}
}
