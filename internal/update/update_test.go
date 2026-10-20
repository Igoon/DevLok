package update

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"8.4.26", "8.5.11", -1},
		{"8.5.11", "8.4.26", 1},
		{"8.4", "8.4.0", 0},
		{"1.30.6", "1.30.6", 0},
		{"9.4.0", "8.4.10", 1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Fatalf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
