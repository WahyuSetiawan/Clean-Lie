package clean

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

func TestValidate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".cache")

	ok := []Target{
		{Path: filepath.Join(root, "opencode"), Root: root},
		{Path: filepath.Join(root, "ms-playwright"), Root: root},
		{Path: filepath.Join(root, "a", "b"), Root: root, Depth: 2},
		{Path: root, Root: root, Whole: true},
	}
	for _, tt := range ok {
		if err := Validate(tt); err != nil {
			t.Errorf("Validate(%s) = %v, mau nil", tt.Path, err)
		}
	}

	bad := []Target{
		{Path: "", Root: root},
		{Path: "relative/x", Root: root},
		{Path: root, Root: root},                                          // root bukan item
		{Path: filepath.Join(root, "a", "b"), Root: root},                 // terlalu dalam
		{Path: filepath.Join(home, ".ssh", "id"), Root: root},             // bukan anak root
		{Path: filepath.Join(home, ".config"), Root: home},                // protected
		{Path: "/", Root: "/", Whole: true},                               // protected
		{Path: filepath.Join(home, "Documents"), Root: home, Whole: true}, // protected
		{Path: filepath.Join(root, "x"), Root: ""},                        // root invalid
	}
	for _, tt := range bad {
		if err := Validate(tt); err == nil {
			t.Errorf("Validate(%+v) = nil, mau error", tt)
		}
	}
}

func TestBlockedReason(t *testing.T) {
	procs := map[string]int{
		"brave":           100,
		"qemu-system-x86": 200,
		"chromium-headle": 300,
	}

	cases := []struct {
		path string
		want string
	}{
		{"/home/u/.cache/BraveSoftware/Brave-Browser", "Brave"},
		{"/home/u/.cache/mozilla/firefox", ""},
		{"/home/u/.android/avd/voi.avd", "emulator Android"},
		{"/home/u/.cache/ms-playwright/chromium-1", "Chromium"},
		{"/home/u/.cache/opencode", ""},
	}
	for _, c := range cases {
		got := blockedReason(c.path, procs)
		if c.want == "" && got != "" {
			t.Errorf("blockedReason(%s) = %q, mau kosong", c.path, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("blockedReason(%s) = %q, mau memuat %q", c.path, got, c.want)
		}
	}
}

func TestProcessesFrom(t *testing.T) {
	root := t.TempDir()
	for pid, name := range map[string]string{
		"10": "brave",
		"11": "firefox",
		"12": "qemu-system-x86",
	} {
		dir := filepath.Join(root, pid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// direktori bukan PID harus diabaikan
	if err := os.MkdirAll(filepath.Join(root, "bukan-pid"), 0o755); err != nil {
		t.Fatal(err)
	}

	procs := processesFrom(root)
	if len(procs) != 3 || procs["brave"] != 10 {
		t.Fatalf("procs = %v", procs)
	}
}

// TestDefaultCatalogValidates memastikan setiap item dari katalog nyata lolos
// guard (terutama item depth>1 seperti system-images). Jalankan dengan
// CLEAN_TEST_SCAN=1 karena memindai mesin.
func TestDefaultCatalogValidates(t *testing.T) {
	if os.Getenv("CLEAN_TEST_SCAN") == "" {
		t.Skip("set CLEAN_TEST_SCAN=1 untuk memindai katalog nyata")
	}
	rep, err := rules.Plan(context.Background(), rules.DefaultRules(), rules.PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, g := range rep.Groups {
		for _, it := range g.Items {
			n++
			err := Validate(Target{Path: it.Path, Root: it.Root, Whole: it.Whole, Depth: it.Depth})
			if err != nil {
				t.Errorf("%s (%s): %v", it.ID, it.Path, err)
			}
		}
	}
	if n == 0 {
		t.Fatal("tidak ada item")
	}
	t.Logf("%d item katalog valid", n)
}
