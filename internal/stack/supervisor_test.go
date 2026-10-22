package stack

import (
	"os"
	"testing"
	"time"

	"devlok/internal/supervisor"
)

func TestSupervisorSpecsRoundtrip(t *testing.T) {
	state := t.TempDir()
	specs := []supervisor.Spec{{
		Name:       "php",
		Executable: `C:\php\php-cgi.exe`,
		Args:       []string{"-b", "127.0.0.1:9000"},
		Health:     supervisor.Health{TCPPort: 9000},
	}}
	if err := writeSupervisorSpecs(state, specs); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSupervisorSpecs(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "php" || got[0].Executable != specs[0].Executable || got[0].Health.TCPPort != 9000 {
		t.Fatalf("specs = %+v", got)
	}
}

func TestSupervisorPaused(t *testing.T) {
	state := t.TempDir()
	if SupervisorPaused(state) {
		t.Fatal("should not be paused initially")
	}
	end := beginSupervisorPause(state)
	if !SupervisorPaused(state) {
		t.Fatal("should be paused")
	}
	end()
	if SupervisorPaused(state) {
		t.Fatal("pause should be cleared")
	}
	path := supervisorPausePath(state)
	if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-61 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if SupervisorPaused(state) {
		t.Fatal("stale pause should not block")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stale pause should be removed")
	}
}

func TestStopSupervisorWithoutDaemon(t *testing.T) {
	state := t.TempDir()
	start := time.Now()
	StopSupervisor(state)
	if time.Since(start) > 2*time.Second {
		t.Fatal("stop supervisor took too long without daemon")
	}
	if _, err := os.Stat(supervisorStopPath(state)); !os.IsNotExist(err) {
		t.Fatal("stop flag should be removed")
	}
}
