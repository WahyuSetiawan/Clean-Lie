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
	if c.Theme != ThemeSystem || c.Language != LanguageID || c.DeleteMode != DeleteTrash || !c.AutoSelectSafe {
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
