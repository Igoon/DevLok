package update

import (
	"testing"

	rt "devlok/internal/runtime"
)

func phpRegistry(versions ...string) []rt.Registry {
	reg := rt.Registry{Schema: 1, Name: "php", Display: "PHP"}
	for _, v := range versions {
		reg.Versions = append(reg.Versions, rt.Version{
			Version:   v,
			Artifacts: map[string]rt.Artifact{"windows/amd64": {}},
		})
	}
	return []rt.Registry{reg}
}

func installedPHP(versions ...string) []rt.Installed {
	out := make([]rt.Installed, 0, len(versions))
	for _, v := range versions {
		out = append(out, rt.Installed{Name: "php", Version: v})
	}
	return out
}

func TestCheckPatchAndNewSeries(t *testing.T) {
	got := checkInstalled(
		installedPHP("8.4.26"),
		phpRegistry("8.4.26", "8.4.27", "8.5.11", "8.5.12"),
		"windows/amd64",
	)
	if len(got) != 2 {
		t.Fatalf("updates = %#v, want 2", got)
	}
	if got[0].Kind != KindPatch || got[0].Series != "8.4" || got[0].Installed != "8.4.26" || got[0].Latest != "8.4.27" {
		t.Fatalf("patch update = %#v", got[0])
	}
	if got[1].Kind != KindNew || got[1].Series != "8.5" || got[1].Installed != "8.4.26" || got[1].Latest != "8.5.12" {
		t.Fatalf("new series update = %#v", got[1])
	}
}

func TestCheckPatchPerInstalledVersion(t *testing.T) {
	got := checkInstalled(
		installedPHP("8.4.26", "8.5.11"),
		phpRegistry("8.4.26", "8.4.27", "8.5.11", "8.5.12"),
		"windows/amd64",
	)
	if len(got) != 2 {
		t.Fatalf("updates = %#v, want 2", got)
	}
	if got[0].Kind != KindPatch || got[0].Installed != "8.5.11" || got[0].Latest != "8.5.12" {
		t.Fatalf("first patch = %#v", got[0])
	}
	if got[1].Kind != KindPatch || got[1].Installed != "8.4.26" || got[1].Latest != "8.4.27" {
		t.Fatalf("second patch = %#v", got[1])
	}
}

func TestCheckNoUpdates(t *testing.T) {
	got := checkInstalled(
		installedPHP("8.4.26", "8.4.27", "8.5.12"),
		phpRegistry("8.4.26", "8.4.27", "8.5.11", "8.5.12"),
		"windows/amd64",
	)
	if len(got) != 0 {
		t.Fatalf("updates = %#v, want none", got)
	}
}

func TestCheckPatchUsesHighestInstalledPerSeries(t *testing.T) {
	got := checkInstalled(
		installedPHP("8.4.26", "8.4.27"),
		phpRegistry("8.4.26", "8.4.27", "8.4.28"),
		"windows/amd64",
	)
	if len(got) != 1 {
		t.Fatalf("updates = %#v, want 1", got)
	}
	if got[0].Kind != KindPatch || got[0].Installed != "8.4.27" || got[0].Latest != "8.4.28" {
		t.Fatalf("patch update = %#v", got[0])
	}
}

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
		{"0.1.0-dev", "0.1.0", -1},
		{"0.2.0-dev", "0.1.0", 1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Fatalf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
