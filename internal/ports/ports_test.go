package ports

import (
	"net"
	"os"
	"testing"
)

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

func TestCheckAvailable(t *testing.T) {
	port := freePort(t)
	st := Check(port)
	if !st.Available {
		t.Fatalf("port %d should be available: %+v", port, st)
	}
}

func TestCheckOccupiedOwner(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	st := Check(port)
	if st.Available {
		t.Fatalf("port %d should be occupied", port)
	}
	if st.PID == 0 {
		t.Skip("owner lookup not supported on this platform")
	}
	if st.PID != os.Getpid() {
		t.Fatalf("owner pid = %d want %d", st.PID, os.Getpid())
	}
}

func TestPick(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port
	alt := freePort(t)
	got, err := Pick(busy, alt)
	if err != nil {
		t.Fatal(err)
	}
	if got != alt {
		t.Fatalf("got %d want %d", got, alt)
	}
}
