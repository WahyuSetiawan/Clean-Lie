package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c, warn := Load()
	if warn != "" {
		t.Fatalf("warning = %q", warn)
	}
	if c.Theme != ThemeSystem || c.Language != LanguageID || c.DeleteMode != DeleteTrash || !c.AutoSelectSafe || !c.Notify {
		t.Fatalf("default = %+v", c)
	}
	if c.LargeBytes != 100<<20 || c.StaleDays != 365 {
		t.Fatalf("threshold = %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c := Default()
	c.Theme = ThemeLight
	c.Language = LanguageEN
	c.LargeBytes = 500 << 20
	c.StaleDays = 30
	c.ArtifactRoots = []string{"~/Projects", "~/work"}
	c.DirenvRoots = []string{"~/Projects", "~/Projects", "~/src"}
	c.Excluded = []string{"~/Downloads", "~/Downloads", "relative"}
	saved, warns, err := Save(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings = %v", warns)
	}
	if len(saved.Excluded) != 1 || !filepath.IsAbs(saved.Excluded[0]) {
		t.Fatalf("excluded = %v", saved.Excluded)
	}
	if len(saved.DirenvRoots) != 2 || !filepath.IsAbs(saved.DirenvRoots[0]) {
		t.Fatalf("direnvRoots = %v", saved.DirenvRoots)
	}
	loaded, warn := Load()
	if warn != "" {
		t.Fatalf("warning = %q", warn)
	}
	if loaded.Theme != ThemeLight || loaded.Language != LanguageEN || loaded.LargeBytes != 500<<20 || loaded.StaleDays != 30 {
		t.Fatalf("loaded = %+v", loaded)
	}
	if _, err := os.Stat(Path()); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidConfigFallsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte("{rusak"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, warn := Load()
	if !strings.Contains(warn, "tidak valid") {
		t.Fatalf("warning = %q", warn)
	}
	if c.Theme != ThemeSystem || c.LargeBytes != 100<<20 {
		t.Fatalf("fallback = %+v", c)
	}
}

func TestNormalizeClamps(t *testing.T) {
	c, warns := Normalize(Config{Theme: "neon", Language: "xx", DeleteMode: "nuke", LargeBytes: 1, StaleDays: 99999})
	if len(warns) < 3 {
		t.Fatalf("warnings = %v", warns)
	}
	if c.Theme != ThemeSystem || c.Language != LanguageID || c.DeleteMode != DeleteTrash || c.LargeBytes != 100<<20 || c.StaleDays != 3650 {
		t.Fatalf("normalized = %+v", c)
	}
}

func TestCustomRulesNormalize(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	c := Default()
	c.CustomRules = []CustomRule{
		{Path: "~/work/cache", Risk: "safe", Depth: 3},
		{Path: "~/work/cache", Label: "duplikat"},
		{Path: "~/.ssh", Label: "secret"},
		{Path: "relative", Label: "rel"},
		{Path: "~/work/other", Risk: "unknown"},
	}
	got, warnings := Normalize(c)
	if len(got.CustomRules) != 2 {
		t.Fatalf("rules = %+v warnings=%v", got.CustomRules, warnings)
	}
	if got.CustomRules[0].Path != filepath.Join(home, "work", "cache") || got.CustomRules[0].Depth != 3 {
		t.Fatalf("rule = %+v", got.CustomRules[0])
	}
	if got.CustomRules[1].Risk != "warn" {
		t.Fatalf("risk = %+v", got.CustomRules[1])
	}
	if len(warnings) < 3 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestCustomRulesLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	c := Default()
	for i := 0; i < 40; i++ {
		c.CustomRules = append(c.CustomRules, CustomRule{Path: filepath.Join(home, "cache", string(rune('a'+i)))})
	}
	got, warnings := Normalize(c)
	if len(got.CustomRules) != 32 {
		t.Fatalf("rules = %d", len(got.CustomRules))
	}
	if len(warnings) == 0 {
		t.Fatal("harus ada warning batas")
	}
}
