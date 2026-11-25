package main

import (
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/config"
)

func TestSettingsSaveAndReload(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := NewApp("test", "test")
	if got := a.Settings(); got.Config.Language != "id" || got.Path == "" {
		t.Fatalf("default settings = %+v", got)
	}
	c := a.Settings().Config
	c.Theme = config.ThemeLight
	c.Language = config.LanguageEN
	c.LargeBytes = 512 << 20
	c.StaleDays = 30
	c.Excluded = []string{"~/Downloads"}
	saved, err := a.SaveSettings(c)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Config.Theme != config.ThemeLight || saved.Config.Language != config.LanguageEN || saved.Config.StaleDays != 30 {
		t.Fatalf("saved = %+v", saved)
	}
	if len(saved.Config.Excluded) != 1 {
		t.Fatalf("excluded = %v", saved.Config.Excluded)
	}
	reloaded := NewApp("test", "test")
	if got := reloaded.Settings(); got.Config.Theme != config.ThemeLight || got.Config.LargeBytes != 512<<20 {
		t.Fatalf("reloaded = %+v", got)
	}
	reset, err := reloaded.ResetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if reset.Config.Theme != config.ThemeSystem || reset.Config.Language != config.LanguageID {
		t.Fatalf("reset = %+v", reset)
	}
}

func TestExcludedPath(t *testing.T) {
	ex := []string{"/home/u/Downloads"}
	for _, p := range []string{"/home/u/Downloads", "/home/u/Downloads/a.zip", "/home/u/Downloads-old"} {
		want := p != "/home/u/Downloads-old"
		if got := excludedPath(p, ex); got != want {
			t.Fatalf("excludedPath(%q) = %v, mau %v", p, got, want)
		}
	}
}
