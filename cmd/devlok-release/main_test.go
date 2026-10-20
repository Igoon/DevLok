package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestKeygenSign(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "update.key")
	payloadPath := filepath.Join(dir, "payload.json")
	outPath := filepath.Join(dir, "updates.json")
	payload := []byte(`{"app":{"version":"9.9.9"}}`)
	if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	keygen([]string{"-out", keyPath})
	sign([]string{"-key", keyPath, "-payload", payloadPath, "-out", outPath})
	priv, err := readKey(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(priv.Public().(ed25519.PublicKey), raw, sig) {
		t.Fatal("signature verification failed")
	}
}
