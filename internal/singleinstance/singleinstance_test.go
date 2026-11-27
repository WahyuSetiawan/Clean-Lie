package singleinstance

import (
	"errors"
	"os"
	"testing"
)

func TestAcquireSecondFails(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	first, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	if _, err := Acquire(); !errors.Is(err, ErrRunning) {
		t.Fatalf("acquire kedua = %v, mau ErrRunning", err)
	}
}

func TestReleaseAllowsReacquire(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	first, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	path := first.Path()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire()
	if err != nil {
		t.Fatalf("acquire setelah release: %v", err)
	}
	defer second.Release()
}
