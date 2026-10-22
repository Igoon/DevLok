package update

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devlok/internal/devlokerr"
	rt "devlok/internal/runtime"
	"devlok/internal/usercfg"
	"devlok/internal/version"
)

type Config struct {
	Schema      int    `json:"schema"`
	ManifestURL string `json:"manifest_url"`
	PublicKey   string `json:"public_key"`
}

type AppRelease struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Notes       string `json:"notes,omitempty"`
	Severity    string `json:"severity,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

type AppUpdate struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Notes       string `json:"notes,omitempty"`
	Severity    string `json:"severity,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

type envelope struct {
	Schema    int    `json:"schema"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type payload struct {
	App      *AppRelease   `json:"app"`
	Runtimes []rt.Registry `json:"runtimes,omitempty"`
}

func ConfigPath(stateDir string) string {
	return filepath.Join(usercfg.Root(stateDir), "update.json")
}

func LoadConfig(stateDir string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi update tidak dapat dibaca.", "")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Konfigurasi update tidak valid.", "Periksa config/update.json.")
	}
	if cfg.Schema != 1 {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Schema konfigurasi update tidak didukung.", "")
	}
	return &cfg, nil
}

func RemoteCheck(ctx context.Context, cfg *Config) (*AppUpdate, error) {
	if cfg == nil {
		return nil, nil
	}
	p, _, err := fetchPayload(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if p.App == nil {
		return nil, nil
	}
	if p.App.Version == "" || p.App.URL == "" {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Rilis aplikasi di manifest tidak lengkap.", "")
	}
	if len(p.App.SHA256) != 64 {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "SHA256 rilis aplikasi tidak valid.", "")
	}
	if _, err := hex.DecodeString(p.App.SHA256); err != nil {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "SHA256 rilis aplikasi tidak valid.", "")
	}
	if u, err := url.Parse(p.App.URL); err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && loopbackHost(u.Hostname()))) {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "URL rilis aplikasi harus HTTPS.", "")
	}
	current := version.Current().Version
	if Compare(p.App.Version, current) <= 0 {
		return nil, nil
	}
	return &AppUpdate{
		Current:     current,
		Latest:      p.App.Version,
		URL:         p.App.URL,
		SHA256:      strings.ToLower(p.App.SHA256),
		Notes:       p.App.Notes,
		Severity:    p.App.Severity,
		PublishedAt: p.App.PublishedAt,
	}, nil
}

func configKey(cfg *Config) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cfg.PublicKey))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Public key update tidak valid.", "Gunakan base64 Ed25519 public key 32 byte.")
	}
	return key, nil
}

func fetchPayload(ctx context.Context, cfg *Config) (payload, []byte, error) {
	if strings.TrimSpace(cfg.ManifestURL) == "" {
		return payload{}, nil, devlokerr.New(devlokerr.CodeInvalidArgument, "manifest_url kosong.", "Isi URL manifest update.")
	}
	u, err := url.Parse(cfg.ManifestURL)
	if err != nil || u.Host == "" {
		return payload{}, nil, devlokerr.New(devlokerr.CodeInvalidArgument, "URL manifest update tidak valid.", "")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && loopbackHost(u.Hostname())) {
		return payload{}, nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Manifest update harus memakai HTTPS.", "HTTP hanya diizinkan untuk localhost saat pengujian.")
	}
	key, err := configKey(cfg)
	if err != nil {
		return payload{}, nil, err
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme == "https" || (req.URL.Scheme == "http" && loopbackHost(req.URL.Hostname())) {
				return nil
			}
			return fmt.Errorf("redirect update harus HTTPS")
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ManifestURL, nil)
	if err != nil {
		return payload{}, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return payload{}, nil, devlokerr.Wrap(devlokerr.CodeTimeout, err, "Manifest update tidak dapat diunduh.", "")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return payload{}, nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Manifest update mengembalikan HTTP "+fmt.Sprint(resp.StatusCode), "")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return payload{}, nil, err
	}
	if len(body) >= 1<<20 {
		return payload{}, nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Manifest update terlalu besar.", "")
	}
	p, err := verifyPayload(body, key)
	if err != nil {
		return payload{}, nil, err
	}
	return p, body, nil
}

func verifyPayload(body, key []byte) (payload, error) {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil || env.Schema != 1 {
		return payload{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Envelope manifest update tidak valid.", "")
	}
	rawPayload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return payload{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Payload manifest update tidak valid.", "")
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return payload{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Signature manifest update tidak valid.", "")
	}
	if !ed25519.Verify(ed25519.PublicKey(key), rawPayload, sig) {
		return payload{}, devlokerr.New(devlokerr.CodeChecksumMismatch, "Signature manifest update tidak cocok.", "Jangan gunakan manifest yang tidak ditandatangani publisher resmi.")
	}
	var p payload
	if err := json.Unmarshal(rawPayload, &p); err != nil {
		return payload{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Isi manifest update tidak valid.", "")
	}
	return p, nil
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
