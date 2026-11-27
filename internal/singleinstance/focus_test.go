package singleinstance

import (
	"os"
	"testing"
	"time"
)

func TestFocusRoundTrip(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("session bus tidak tersedia")
	}
	got := make(chan struct{}, 1)
	srv, err := ServeFocus(func() { got <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if err := Activate(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("Focus tidak diterima")
	}
}
