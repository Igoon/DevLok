package tlsca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"devlok/internal/devlokerr"
	"devlok/internal/secret"
)

type CA struct {
	Dir     string
	Cert    *x509.Certificate
	CertPEM []byte
	Key     *ecdsa.PrivateKey
}

func EnsureCA(dir string) (*CA, error) {
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key.dpapi")
	certPEM, certErr := os.ReadFile(certPath)
	keySealed, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		block, _ := pem.Decode(certPEM)
		if block == nil {
			return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "CA certificate rusak.", "Hapus folder certs lalu jalankan ulang.")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "CA certificate tidak dapat dibaca.", "")
		}
		keyPEM, err := secret.Unseal(keySealed)
		if err != nil {
			return nil, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Private key CA tidak dapat dibuka.", "DPAPI hanya bisa dibuka oleh user yang membuatnya.")
		}
		key, err := parseKey(keyPEM)
		if err != nil {
			return nil, err
		}
		return &CA{Dir: dir, Cert: cert, CertPEM: certPEM, Key: key}, nil
	}
	if certErr == nil || keyErr == nil {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Pasangan CA tidak lengkap.", "Hapus folder certs lalu jalankan ulang.")
	}
	return generateCA(dir)
}

func generateCA(dir string) (*CA, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder certificate tidak dapat dibuat.", "")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Key CA tidak dapat dibuat.", "")
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "DevLok Local CA", Organization: []string{"DevLok"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "CA certificate tidak dapat dibuat.", "")
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "CA certificate tidak dapat diparse.", "")
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM, err := marshalKey(key)
	if err != nil {
		return nil, err
	}
	sealed, err := secret.Seal(keyPEM)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Private key CA tidak dapat dienkripsi.", "")
	}
	if err := os.WriteFile(filepath.Join(dir, "ca.crt"), certPEM, 0o644); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "CA certificate tidak dapat ditulis.", "")
	}
	if err := os.WriteFile(filepath.Join(dir, "ca.key.dpapi"), sealed, 0o600); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Private key CA tidak dapat ditulis.", "")
	}
	return &CA{Dir: dir, Cert: cert, CertPEM: certPEM, Key: key}, nil
}

func (ca *CA) Issue(name string, domains []string) (string, string, error) {
	if ca == nil || ca.Key == nil || ca.Cert == nil {
		return "", "", devlokerr.New(devlokerr.CodeInternal, "CA belum dimuat.", "")
	}
	domains = normalizeDomains(domains)
	if len(domains) == 0 {
		return "", "", devlokerr.New(devlokerr.CodeInvalidArgument, "Domain certificate kosong.", "")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", devlokerr.Wrap(devlokerr.CodeInternal, err, "Key certificate tidak dapat dibuat.", "")
	}
	serial, err := randomSerial()
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: domains[0], Organization: []string{"DevLok"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(0, 0, 825),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     domains,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return "", "", devlokerr.Wrap(devlokerr.CodeInternal, err, "Certificate tidak dapat dibuat.", "")
	}
	dir := filepath.Join(ca.Dir, "sites")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder certificate site tidak dapat dibuat.", "")
	}
	base := safeName(name)
	certPath := filepath.Join(dir, base+".crt")
	keyPath := filepath.Join(dir, base+".key")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM, err := marshalKey(key)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return "", "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Certificate site tidak dapat ditulis.", "")
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Key site tidak dapat ditulis.", "")
	}
	return certPath, keyPath, nil
}

func (ca *CA) Trust() error {
	if ca == nil {
		return devlokerr.New(devlokerr.CodeInternal, "CA belum dimuat.", "")
	}
	return trustCA(filepath.Join(ca.Dir, "ca.crt"))
}

func (ca *CA) Untrust() error {
	if ca == nil {
		return devlokerr.New(devlokerr.CodeInternal, "CA belum dimuat.", "")
	}
	sum := sha1.Sum(ca.Cert.Raw)
	return untrustCA(hex.EncodeToString(sum[:]))
}

func (ca *CA) Thumbprint() string {
	if ca == nil || ca.Cert == nil {
		return ""
	}
	sum := sha1.Sum(ca.Cert.Raw)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Serial certificate tidak dapat dibuat.", "")
	}
	return n, nil
}

func marshalKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Private key tidak dapat diserialisasi.", "")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func parseKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Private key CA rusak.", "")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Private key CA tidak dapat diparse.", "")
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Tipe private key CA tidak didukung.", "")
	}
	return key, nil
}

func normalizeDomains(domains []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func safeName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "site"
	}
	return b.String()
}
