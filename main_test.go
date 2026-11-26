package main

import (
	"os"
	"testing"
)

func TestPreferXWayland(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		wayland string
		display string
		want    string
	}{
		{name: "wayland dengan XWayland", wayland: "wayland-1", display: ":0", want: "x11"},
		{name: "pilihan pengguna dipertahankan", backend: "wayland", wayland: "wayland-1", display: ":0", want: "wayland"},
		{name: "tanpa XWayland", wayland: "wayland-1", want: ""},
		{name: "sesi X11", display: ":0", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GDK_BACKEND", tt.backend)
			t.Setenv("WAYLAND_DISPLAY", tt.wayland)
			t.Setenv("DISPLAY", tt.display)

			preferXWayland()

			if got := os.Getenv("GDK_BACKEND"); got != tt.want {
				t.Fatalf("GDK_BACKEND = %q, mau %q", got, tt.want)
			}
		})
	}
}
