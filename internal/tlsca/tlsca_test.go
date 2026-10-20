package tlsca

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCAIdempotent(t *testing.T) {
	dir := t.TempDir()
	ca, err := EnsureCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ca.Cert == nil || !ca.Cert.IsCA {
		t.Fatal("not a CA")
	}
	if _, err := os.Stat(filepath.Join(dir, "ca.crt")); err != nil {
		t.Fatal(err)
	}
	keyFile, err := os.ReadFile(filepath.Join(dir, "ca.key.dpapi"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(keyFile), "PRIVATE KEY") {
		t.Fatal("CA key stored as plaintext PEM")
	}
	ca2, err := EnsureCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ca2.Thumbprint() != ca.Thumbprint() {
		t.Fatal("CA changed between loads")
	}
}

func TestIssueCertificate(t *testing.T) {
	ca, err := EnsureCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath, err := ca.Issue("siso", []string{"siso.test", "www.siso.test"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("invalid cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.DNSNames) != 2 {
		t.Fatalf("dns names = %v", cert.DNSNames)
	}
	if err := cert.CheckSignatureFrom(ca.Cert); err != nil {
		t.Fatalf("not signed by CA: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatal(err)
	}
}

func TestIssueNoDomains(t *testing.T) {
	ca, err := EnsureCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ca.Issue("x", nil); err == nil {
		t.Fatal("expected error")
	}
}
