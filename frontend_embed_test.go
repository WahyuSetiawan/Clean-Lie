package main

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// TestFrontendAssetsResolvable memastikan index.html hanya merujuk aset yang benar-benar
// ikut ter-embed. Flake hanya menyalin file yang di-track git, jadi aset yang lupa
// `git add` akan lolos `nix build` tetapi membuat UI blank di build produksi.
func TestFrontendAssetsResolvable(t *testing.T) {
	html, err := assets.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}

	ref := regexp.MustCompile(`(?:src|href)="([^"]+)"`)
	for _, m := range ref.FindAllStringSubmatch(string(html), -1) {
		path := m[1]
		if strings.HasPrefix(path, "#") || strings.HasPrefix(path, "data:") || strings.Contains(path, "://") {
			continue
		}
		if _, err := fs.Stat(assets, "frontend/dist/"+strings.TrimPrefix(path, "/")); err != nil {
			t.Errorf("aset dirujuk tapi tidak ter-embed: %s (%v)", path, err)
		}
	}
}
