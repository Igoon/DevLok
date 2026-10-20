package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: devlok-release keygen -out key.txt | sign -key key.txt -payload payload.json -out updates.json | pubkey -key key.txt")
	os.Exit(2)
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
