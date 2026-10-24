package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
)

func signedEnvelope(t *testing.T, key ed25519.PrivateKey, app AppRelease) []byte {
	t.Helper()
	raw, err := json.Marshal(payload{App: &app})
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(key, raw)
	env, err := json.Marshal(envelope{
		Schema:    1,
		Payload:   base64.StdEncoding.EncodeToString(raw),
		Signature: base64.StdEncoding.EncodeToString(sig),
	})
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestRemoteCheck(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	app := AppRelease{
		Version:  "99.0.0",
		URL:      "https://example.com/DevLok-Setup.exe",
		SHA256:   strings.Repeat("a", 64),
		Severity: "security",
	}
	body := signedEnvelope(t, priv, app)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	cfg := &Config{Schema: 1, ManifestURL: srv.URL, PublicKey: base64.StdEncoding.EncodeToString(pub)}
	got, err := RemoteCheck(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Latest != app.Version || got.Severity != "security" {
		t.Fatalf("wrong update: %+v", got)
	}
}

func TestRemoteCheckRejectsBadSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	app := AppRelease{Version: "99.0.0", URL: "https://example.com/a.exe", SHA256: strings.Repeat("b", 64)}
	body := signedEnvelope(t, priv, app)
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	env["signature"] = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	body, _ = json.Marshal(env)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	cfg := &Config{Schema: 1, ManifestURL: srv.URL, PublicKey: base64.StdEncoding.EncodeToString(pub)}
	if _, err := RemoteCheck(context.Background(), cfg); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestRemoteCheckNoNewerVersion(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	app := AppRelease{Version: "0.0.0", URL: "https://example.com/a.exe", SHA256: strings.Repeat("c", 64)}
	body := signedEnvelope(t, priv, app)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	cfg := &Config{Schema: 1, ManifestURL: srv.URL, PublicKey: base64.StdEncoding.EncodeToString(pub)}
	got, err := RemoteCheck(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected no update, got %+v", got)
	}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	content := []byte("fake installer")
	sum := sha256.Sum256(content)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()
	rel := &AppUpdate{Latest: "9.9.9", URL: srv.URL + "/DevLok-Setup.exe", SHA256: hex.EncodeToString(sum[:])}
	dir := t.TempDir()
	path, err := Download(context.Background(), rel, dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("wrong content: %q", got)
	}
}

func TestDownloadReportsProgress(t *testing.T) {
	content := []byte(strings.Repeat("installer", 512))
	sum := sha256.Sum256(content)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()
	rel := &AppUpdate{Latest: "9.9.9", URL: srv.URL + "/DevLok-Setup.exe", SHA256: hex.EncodeToString(sum[:])}
	var lastDone, lastTotal int64
	_, err := DownloadWithProgress(context.Background(), rel, t.TempDir(), func(done, total int64) {
		lastDone = done
		lastTotal = total
	})
	if err != nil {
		t.Fatal(err)
	}
	if lastDone != int64(len(content)) {
		t.Fatalf("last progress = %d, want %d", lastDone, len(content))
	}
	if lastTotal != 0 && lastTotal != int64(len(content)) {
		t.Fatalf("total progress = %d, want %d or unknown", lastTotal, len(content))
	}
}

func TestDownloadRejectsBadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake installer"))
	}))
	defer srv.Close()
	rel := &AppUpdate{Latest: "9.9.9", URL: srv.URL + "/setup.exe", SHA256: strings.Repeat("0", 64)}
	dir := t.TempDir()
	if _, err := Download(context.Background(), rel, dir); err == nil {
		t.Fatal("expected checksum error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("partial file left behind: %v", entries)
	}
}

func TestInstallerCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		name, args, err := InstallerCommand(`C:\tmp\DevLok-Setup.exe`)
		if err != nil || name != `C:\tmp\DevLok-Setup.exe` || len(args) != 0 {
			t.Fatalf("unexpected exe command: %q %v %v", name, args, err)
		}
		name, args, err = InstallerCommand(`C:\tmp\DevLok.msi`)
		if err != nil || name != "msiexec.exe" || len(args) != 2 {
			t.Fatalf("unexpected msi command: %q %v %v", name, args, err)
		}
		if _, _, err := InstallerCommand(`C:\tmp\DevLok.zip`); err == nil {
			t.Fatal("expected unsupported format")
		}
		return
	}
	if runtime.GOOS == "darwin" {
		if name, _, err := InstallerCommand("/tmp/DevLok-macos-arm64.dmg"); err != nil || name != "/tmp/DevLok-macos-arm64.dmg" {
			t.Fatalf("unexpected dmg command: %q %v", name, err)
		}
		if _, _, err := InstallerCommand("/tmp/DevLok-Setup.exe"); err == nil {
			t.Fatal("expected exe unsupported on darwin")
		}
	}
}

func TestGitHubCheck(t *testing.T) {
	sum := strings.Repeat("a", 64)
	name := appAssetCandidates("v99.0.0")[0]
	mux := http.NewServeMux()
	mux.HandleFunc("/releases", func(w http.ResponseWriter, r *http.Request) {
		rel := []githubRelease{{
			TagName: "v99.0.0",
			Name:    "DevLok 99",
			Body:    "notes",
			Assets: []githubAsset{
				{Name: name, BrowserDownloadURL: "http://" + r.Host + "/setup.bin"},
				{Name: name + ".sha256", BrowserDownloadURL: "http://" + r.Host + "/setup.bin.sha256"},
			},
		}}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/setup.bin.sha256", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sum + "  " + name + "\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	got, err := githubCheck(context.Background(), srv.URL+"/releases")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Latest != "99.0.0" || got.SHA256 != sum || got.Notes != "notes" {
		t.Fatalf("wrong update: %+v", got)
	}
}
