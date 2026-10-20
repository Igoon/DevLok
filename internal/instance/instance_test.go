package instance

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSingleInstance(t *testing.T) {
	scope := fmt.Sprintf("devlok-test-%d", time.Now().UnixNano())
	first, err := Acquire(scope)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := Acquire(scope); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second acquire should fail with ErrAlreadyRunning, got %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("release: %v", err)
	}
	third, err := Acquire(scope)
	if err != nil {
		t.Fatalf("third acquire after release: %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("release third: %v", err)
	}
}

func TestDifferentScopes(t *testing.T) {
	base := time.Now().UnixNano()
	a, err := Acquire(fmt.Sprintf("devlok-a-%d", base))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Acquire(fmt.Sprintf("devlok-b-%d", base))
	if err != nil {
		t.Fatalf("different scope should not conflict: %v", err)
	}
	defer b.Close()
}
