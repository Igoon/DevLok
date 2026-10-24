package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	rt "devlok/internal/runtime"
)

type envelope struct {
	Schema    int    `json:"schema"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2:])
	case "sign":
		sign(os.Args[2:])
	case "pubkey":
		pubkey(os.Args[2:])
	case "registries":
		registries(os.Args[2:])
	case "payload":
		payloadCommand(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: devlok-release keygen -out key.txt | sign -key key.txt -payload payload.json -out updates.json | pubkey -key key.txt | registries -out runtimes.json | payload -version X -url URL -sha256 HASH -runtimes runtimes.json -out payload.json")
	os.Exit(2)
}

type releaseApp struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Notes       string `json:"notes,omitempty"`
	Severity    string `json:"severity,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

type releasePayload struct {
	App      releaseApp            `json:"app"`
	Apps     map[string]releaseApp `json:"apps,omitempty"`
	Runtimes []rt.Registry         `json:"runtimes,omitempty"`
}

func payloadCommand(args []string) {
	fs := flag.NewFlagSet("payload", flag.ExitOnError)
	version := fs.String("version", "", "app version")
	rawURL := fs.String("url", "", "installer URL")
	sha := fs.String("sha256", "", "installer SHA256")
	notes := fs.String("notes", "", "release notes")
	severity := fs.String("severity", "feature", "severity")
	published := fs.String("published-at", "", "published timestamp")
	runtimesPath := fs.String("runtimes", "", "runtimes json")
	out := fs.String("out", "", "output payload json")
	macosArmURL := fs.String("macos-arm64-url", "", "macOS arm64 asset URL")
	macosArmSHA := fs.String("macos-arm64-sha256", "", "macOS arm64 asset SHA256")
	macosIntelURL := fs.String("macos-intel-url", "", "macOS intel asset URL")
	macosIntelSHA := fs.String("macos-intel-sha256", "", "macOS intel asset SHA256")
	_ = fs.Parse(args)
	if *version == "" || *rawURL == "" || *sha == "" || *out == "" {
		fail("payload: -version, -url, -sha256, -out wajib")
	}
	var regs []rt.Registry
	if *runtimesPath != "" {
		data, err := os.ReadFile(*runtimesPath)
		if err != nil {
			fail(err.Error())
		}
		if err := json.Unmarshal(data, &regs); err != nil {
			fail(err.Error())
		}
	}
	if len(regs) == 0 {
		loaded, err := rt.LoadEmbedded()
		if err != nil {
			fail(err.Error())
		}
		regs = loaded
	}
	if *published == "" {
		*published = time.Now().UTC().Format(time.RFC3339)
	}
	base := releaseApp{
		Version:     *version,
		Notes:       *notes,
		Severity:    *severity,
		PublishedAt: *published,
	}
	p := releasePayload{
		App: func() releaseApp {
			a := base
			a.URL = *rawURL
			a.SHA256 = *sha
			return a
		}(),
		Runtimes: regs,
	}
	apps := map[string]releaseApp{}
	addApp := func(key, u, s string) {
		if u == "" || s == "" {
			return
		}
		a := base
		a.URL = u
		a.SHA256 = s
		apps[key] = a
	}
	addApp("darwin/arm64", *macosArmURL, *macosArmSHA)
	addApp("darwin/amd64", *macosIntelURL, *macosIntelSHA)
	if len(apps) > 0 {
		p.Apps = apps
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	data = append(data, '\n')
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err.Error())
	}
}

func registries(args []string) {
	fs := flag.NewFlagSet("registries", flag.ExitOnError)
	out := fs.String("out", "", "output json")
	_ = fs.Parse(args)
	regs, err := rt.LoadEmbedded()
	if err != nil {
		fail(err.Error())
	}
	data, err := json.MarshalIndent(regs, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	data = append(data, '\n')
	if *out == "" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err.Error())
	}
}

func keygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "", "path private key")
	_ = fs.Parse(args)
	if *out == "" {
		fail("keygen: -out wajib")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fail(err.Error())
	}
	if err := os.WriteFile(*out, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600); err != nil {
		fail(err.Error())
	}
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
}

func sign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", "", "private key file")
	payloadPath := fs.String("payload", "", "payload json")
	outPath := fs.String("out", "", "output envelope")
	_ = fs.Parse(args)
	if *keyPath == "" || *payloadPath == "" || *outPath == "" {
		fail("sign: -key, -payload, -out wajib")
	}
	priv, err := readKey(*keyPath)
	if err != nil {
		fail(err.Error())
	}
	payload, err := os.ReadFile(*payloadPath)
	if err != nil {
		fail(err.Error())
	}
	sig := ed25519.Sign(priv, payload)
	env := envelope{
		Schema:    1,
		Payload:   base64.StdEncoding.EncodeToString(payload),
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	data = append(data, '\n')
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fail(err.Error())
	}
}

func pubkey(args []string) {
	fs := flag.NewFlagSet("pubkey", flag.ExitOnError)
	keyPath := fs.String("key", "", "private key file")
	_ = fs.Parse(args)
	if *keyPath == "" {
		fail("pubkey: -key wajib")
	}
	priv, err := readKey(*keyPath)
	if err != nil {
		fail(err.Error())
	}
	fmt.Println(base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey)))
}

func readKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(string(bytesTrim(data)))
	if err != nil {
		return nil, err
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("panjang key tidak valid: %d", len(raw))
	}
}

func bytesTrim(data []byte) []byte {
	for len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r' || data[len(data)-1] == ' ' || data[len(data)-1] == '\t') {
		data = data[:len(data)-1]
	}
	return data
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "devlok-release:", msg)
	os.Exit(1)
}
