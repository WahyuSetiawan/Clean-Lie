package rules

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestMacRulesValid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	seen := map[string]bool{}
	for _, r := range macRules() {
		if r.ID == "" || r.Group == "" || r.Path == "" {
			t.Errorf("rule tidak lengkap: %+v", r)
		}
		if seen[r.ID] {
			t.Errorf("id duplikat: %s", r.ID)
		}
		seen[r.ID] = true
		if !filepath.IsAbs(r.Path) {
			t.Errorf("%s: path tidak absolut: %s", r.ID, r.Path)
		}
		if r.Risk != RiskSafe && r.Risk != RiskWarn && r.Risk != RiskDanger && r.Risk != RiskBlocked {
			t.Errorf("%s: risk tidak valid: %s", r.ID, r.Risk)
		}
	}
	for _, want := range []string{"mac-caches", "mac-logs", "xcode-derived", "coresim-devices"} {
		if !seen[want] {
			t.Errorf("rule %q tidak ada", want)
		}
	}
}

func TestMacProtectedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS != "darwin" {
		t.Skip("aturan path macOS hanya aktif di darwin")
	}

	protected := []string{
		"/System/Library",
		"/Applications/Safari.app",
		"/private/var/db",
		"/opt/homebrew/bin/brew",
		filepath.Join(home, "Library"),
		filepath.Join(home, "Library", "Application Support", "Slack"),
		filepath.Join(home, "Library", "Keychains", "login.keychain-db"),
		filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs"),
	}
	for _, p := range protected {
		if !IsProtected(p) {
			t.Errorf("%q mestinya protected", p)
		}
	}

	allowed := []string{
		filepath.Join(home, "Library", "Caches", "Homebrew"),
		filepath.Join(home, "Library", "Logs", "Slack"),
		filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData"),
		filepath.Join(home, "Library", "Developer", "CoreSimulator", "Devices"),
	}
	for _, p := range allowed {
		if IsProtected(p) {
			t.Errorf("%q mestinya boleh dibersihkan", p)
		}
	}
}
