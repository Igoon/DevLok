package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	rt "devlok/internal/runtime"
)

func signedPayloadEnvelope(t *testing.T, key ed25519.PrivateKey, p payload) []byte {
	t.Helper()
	raw, err := json.Marshal(p)
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

func testPHPRegistry() rt.Registry {
	return rt.Registry{
		Schema:  1,
		Name:    "php",
		Display: "PHP",
		Versions: []rt.Version{{
			Version: "8.4.27",
			Artifacts: map[string]rt.Artifact{
				"windows/amd64": {
					URL:         "https://example.com/php-8.4.27.zip",
					SHA256:      strings.Repeat("a", 64),
					Archive:     rt.ArchiveZip,
					Executables: []string{"php.exe", "php-cgi.exe"},
				},
			},
		}},
	}
}

func TestRemoteRegistriesAndCache(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := signedPayloadEnvelope(t, priv, payload{Runtimes: []rt.Registry{testPHPRegistry()}})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	cfg := &Config{Schema: 1, ManifestURL: server.URL, PublicKey: base64.StdEncoding.EncodeToString(pub)}
	cachePath := RegistryCachePath(t.TempDir())
	regs, err := RemoteRegistries(context.Background(), cfg, cachePath, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(regs) != 1 || regs[0].Name != "php" || regs[0].Versions[0].Version != "8.4.27" {
		t.Fatalf("registries = %+v", regs)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatal("cache not written")
	}
	server.Close()
	regs, err = RemoteRegistries(context.Background(), cfg, cachePath, time.Hour)
	if err != nil || len(regs) != 1 {
		t.Fatalf("cache read: %+v %v", regs, err)
	}
	regs, err = RemoteRegistries(context.Background(), cfg, cachePath, 0)
	if err != nil || len(regs) != 1 {
		t.Fatalf("stale fallback: %+v %v", regs, err)
	}
}

func TestRemoteRegistriesRejectsBadSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := signedPayloadEnvelope(t, priv, payload{Runtimes: []rt.Registry{testPHPRegistry()}})
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	env["signature"] = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	body, _ = json.Marshal(env)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	cfg := &Config{Schema: 1, ManifestURL: server.URL, PublicKey: base64.StdEncoding.EncodeToString(pub)}
	if _, err := RemoteRegistries(context.Background(), cfg, "", 0); err == nil {
		t.Fatal("expected signature error")
	}
}
