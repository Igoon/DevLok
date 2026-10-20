package supervisor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"devlok/internal/platform"
)

func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("DEVLOK_HELPER_MODE")
	if mode == "" {
		return
	}
	switch mode {
	case "serve":
		addr := os.Getenv("DEVLOK_HELPER_ADDR")
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			os.Exit(2)
		}
		if pidfile := os.Getenv("DEVLOK_HELPER_CHILD_PID"); pidfile != "" {
			child := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
			child.Env = append(os.Environ(), "DEVLOK_HELPER_MODE=child")
			if err := child.Start(); err == nil {
				_ = os.WriteFile(pidfile, []byte(strconv.Itoa(child.Process.Pid)), 0o600)
			}
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				os.Exit(0)
			}
			_ = conn.Close()
		}
	case "child":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "exit":
		time.Sleep(100 * time.Millisecond)
		os.Exit(1)
	}
	os.Exit(0)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func testSpec(name string, port int) Spec {
	return Spec{
		Name:       name,
		Executable: os.Args[0],
		Args:       []string{"-test.run=TestHelperProcess"},
		Dir:        os.TempDir(),
		Env: []string{
			"DEVLOK_HELPER_MODE=serve",
			fmt.Sprintf("DEVLOK_HELPER_ADDR=127.0.0.1:%d", port),
		},
		Health: Health{TCPPort: port, Timeout: 8 * time.Second, Interval: 50 * time.Millisecond},
		Policy: PolicyNo,
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "services.json")
	m, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestStartStopHealth(t *testing.T) {
	m := newTestManager(t)
	defer func() { _ = m.Close() }()
	port := freePort(t)
	st, err := m.Start(testSpec("web", port))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if st.State != StateRunning || st.PID == 0 {
		t.Fatalf("unexpected status: %+v", st)
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		t.Fatalf("service not listening: %v", err)
	}
	_ = conn.Close()

	if _, err := m.Start(testSpec("web", port)); err == nil {
		t.Fatal("second start should fail")
	}

	if err := m.Stop("web"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	st, _ = m.Get("web")
	if st.State != StateStopped || st.PID != 0 {
		t.Fatalf("unexpected stopped status: %+v", st)
	}
	if _, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond); err == nil {
		t.Fatal("port should be closed after stop")
	}
}

func TestCloseKillsProcessTree(t *testing.T) {
	m := newTestManager(t)
	port := freePort(t)
	pidfile := filepath.Join(t.TempDir(), "child.pid")
	spec := testSpec("tree", port)
	spec.Env = append(spec.Env, "DEVLOK_HELPER_CHILD_PID="+pidfile)
	if _, err := m.Start(spec); err != nil {
		t.Fatalf("start: %v", err)
	}
	var childPID int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidfile)
		if err == nil && len(data) > 0 {
			childPID, _ = strconv.Atoi(string(data))
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("child pid file not written")
	}
	if err := m.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	plat := platform.Current()
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := plat.ProcessStartTime(childPID); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("child process %d survived Close", childPID)
}

func TestReconcileStaleState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.json")
	stale := `{"schema":1,"services":[{"name":"old","pid":999999,"state":"running","started_at":"2026-10-20T00:00:00Z","restarts":0}]}`
	if err := os.WriteFile(path, []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := m.Get("old")
	if !ok {
		t.Fatal("status missing")
	}
	if st.State != StateStopped || st.PID != 0 {
		t.Fatalf("stale process not reconciled: %+v", st)
	}
}

func TestRestartPolicy(t *testing.T) {
	m := newTestManager(t)
	defer func() { _ = m.Close() }()
	spec := Spec{
		Name:        "crash",
		Executable:  os.Args[0],
		Args:        []string{"-test.run=TestHelperProcess"},
		Env:         []string{"DEVLOK_HELPER_MODE=exit"},
		Policy:      PolicyAlways,
		MaxRestarts: 1,
		Backoff:     50 * time.Millisecond,
	}
	if _, err := m.Start(spec); err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st, _ := m.Get("crash")
		if st.Restarts >= 1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	st, _ := m.Get("crash")
	t.Fatalf("restart policy did not trigger: %+v", st)
}

func TestStartMissingExecutable(t *testing.T) {
	m := newTestManager(t)
	defer func() { _ = m.Close() }()
	_, err := m.Start(Spec{Name: "nope", Executable: filepath.Join(t.TempDir(), "missing.exe")})
	if err == nil {
		t.Fatal("expected missing executable error")
	}
}

func TestStressStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test skipped in short mode")
	}
	m := newTestManager(t)
	defer func() { _ = m.Close() }()
	port := freePort(t)
	for i := 0; i < 50; i++ {
		spec := testSpec("stress", port)
		if _, err := m.Start(spec); err != nil {
			t.Fatalf("cycle %d start: %v", i, err)
		}
		if err := m.Stop("stress"); err != nil {
			t.Fatalf("cycle %d stop: %v", i, err)
		}
		st, _ := m.Get("stress")
		if st.State != StateStopped || st.PID != 0 {
			t.Fatalf("cycle %d unexpected status: %+v", i, st)
		}
	}
}
