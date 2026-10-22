package update

import "testing"

func TestPreferencesRoundtrip(t *testing.T) {
	state := t.TempDir()
	def, err := LoadPreferences(state)
	if err != nil {
		t.Fatal(err)
	}
	if def.IntervalMinutes != DefaultIntervalMinutes || def.AutoUpdateApp || def.AutoUpdateRuntime {
		t.Fatalf("defaults = %+v", def)
	}
	want := Preferences{
		Schema:                  1,
		AutoUpdateApp:           true,
		AutoUpdateRuntime:       true,
		AutoUpdateRuntimeSeries: true,
		IntervalMinutes:         15,
	}
	if err := SavePreferences(state, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPreferences(state)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("prefs = %+v, want %+v", got, want)
	}
}

func TestPreferencesRejectsBadInterval(t *testing.T) {
	if err := SavePreferences(t.TempDir(), Preferences{IntervalMinutes: 1}); err == nil {
		t.Fatal("expected interval error")
	}
}
