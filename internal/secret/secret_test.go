package secret

import (
	"bytes"
	"testing"
)

func TestSealRoundtrip(t *testing.T) {
	in := []byte("rootsecret-dbpassword")
	sealed, err := Seal(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, in) {
		t.Fatal("sealed data contains plaintext")
	}
	out, err := Unseal(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, in) {
		t.Fatalf("roundtrip mismatch: %q", out)
	}
}
