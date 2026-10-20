package cli

import "testing"

func TestSplitTarget(t *testing.T) {
	cases := []struct {
		in       string
		req      bool
		wantName string
		wantVer  string
		wantErr  bool
	}{
		{"php@8.4", false, "php", "8.4", false},
		{"php", false, "php", "", false},
		{"php", true, "", "", true},
		{"php@", false, "", "", true},
		{"php@8.4.26", true, "php", "8.4.26", false},
		{"bad name@8", false, "", "", true},
	}
	for _, tc := range cases {
		name, ver, err := splitTarget(tc.in, tc.req)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if name != tc.wantName || ver != tc.wantVer {
			t.Fatalf("%q: got %q %q", tc.in, name, ver)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	if got := humanBytes(512); got != "512 B" {
		t.Fatalf("got %q", got)
	}
	if got := humanBytes(1536); got != "1.5 KB" {
		t.Fatalf("got %q", got)
	}
}
