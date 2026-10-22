package update

import (
	"encoding/json"
	"os"
	"path/filepath"

	"devlok/internal/devlokerr"
	"devlok/internal/usercfg"
)

const DefaultIntervalMinutes = 360

type Preferences struct {
	Schema                  int  `json:"schema"`
	AutoUpdateApp           bool `json:"auto_update_app"`
	AutoUpdateRuntime       bool `json:"auto_update_runtime"`
	AutoUpdateRuntimeSeries bool `json:"auto_update_runtime_series"`
	IntervalMinutes         int  `json:"interval_minutes"`
}

func PreferencesPath(stateDir string) string {
	return filepath.Join(usercfg.Root(stateDir), "preferences.json")
}

func DefaultPreferences() Preferences {
	return Preferences{Schema: 1, IntervalMinutes: DefaultIntervalMinutes}
}

func LoadPreferences(stateDir string) (Preferences, error) {
	data, err := os.ReadFile(PreferencesPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultPreferences(), nil
		}
		return Preferences{}, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Preferensi update tidak dapat dibaca.", "")
	}
	prefs := DefaultPreferences()
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Preferences{}, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Preferensi update tidak valid.", "Periksa config/preferences.json.")
	}
	if prefs.Schema != 1 {
		return Preferences{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Schema preferensi update tidak didukung.", "")
	}
	if prefs.IntervalMinutes < 5 || prefs.IntervalMinutes > 10080 {
		prefs.IntervalMinutes = DefaultIntervalMinutes
	}
	return prefs, nil
}

func SavePreferences(stateDir string, prefs Preferences) error {
	if prefs.IntervalMinutes < 5 || prefs.IntervalMinutes > 10080 {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Interval update harus antara 5 dan 10080 menit.", "")
	}
	prefs.Schema = 1
	path := PreferencesPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder konfigurasi tidak dapat dibuat.", "")
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Preferensi update tidak dapat ditulis.", "")
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Preferensi update tidak dapat dipindahkan.", "")
	}
	return nil
}
