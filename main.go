package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// Diisi lewat -ldflags saat build (lihat Makefile / flake.nix).
var (
	version = "0.9.0-dev"
	commit  = "unknown"
)

func main() {
	if handled, code := runCLI(os.Args[1:]); handled {
		os.Exit(code)
	}

	// GTK3/WebKitGTK di Wayland (mis. Hyprland) bisa melaporkan viewport
	// negatif setelah window di-tile/resize, sehingga UI bertumpuk. Pakai
	// XWayland kecuali pengguna memilih backend lain lewat GDK_BACKEND.
	preferXWayland()

	app := NewApp(version, commit)

	err := wails.Run(&options.App{
		Title:       "Cleanlie",
		Width:       1280,
		Height:      820,
		MinWidth:    980,
		MinHeight:   620,
		Frameless:   false,
		AssetServer: &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{
			R: 11, G: 13, B: 16, A: 255,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []any{app},
	})
	if err != nil {
		log.Fatal("cleanlie: ", err)
	}
}

// preferXWayland menghindari bug viewport WebKitGTK di Wayland tanpa
// menimpa pilihan pengguna yang sudah menetapkan GDK_BACKEND.
func preferXWayland() {
	if os.Getenv("GDK_BACKEND") == "" && os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("DISPLAY") != "" {
		_ = os.Setenv("GDK_BACKEND", "x11")
	}
}
