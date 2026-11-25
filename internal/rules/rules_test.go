package rules

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mkfile(t *testing.T, path string, size int) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, bytes.Repeat([]byte("a"), size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := ExpandHome("~"); got != home {
		t.Fatalf("~ = %q", got)
	}
	if got := ExpandHome("~/.cache/x"); got != filepath.Join(home, ".cache/x") {
		t.Fatalf("~/ = %q", got)
	}
	if got := ExpandHome("/abs"); got != "/abs" {
		t.Fatalf("abs = %q", got)
	}
}

func TestPlanDedupeByPriority(t *testing.T) {
	root := t.TempDir()
	mkfile(t, filepath.Join(root, "a", "file"), 8192)
	mkfile(t, filepath.Join(root, "b", "file"), 8192)

	list := []Rule{
		{ID: "low", Group: "app", Path: root, Depth: 1, Risk: RiskSafe, Priority: 1},
		{ID: "high", Group: "app", Path: filepath.Join(root, "a"), Whole: true, Risk: RiskWarn, Priority: 10, Note: "dari high"},
	}

	report, err := Plan(context.Background(), list, PlanOptions{Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 1 || len(report.Groups[0].Items) != 2 {
		t.Fatalf("items = %+v", report.Groups)
	}
	for _, it := range report.Groups[0].Items {
		if it.Label == "a" {
			if it.Risk != RiskWarn || it.Note != "dari high" {
				t.Fatalf("item a tidak memakai rule prioritas tinggi: %+v", it)
			}
		}
	}
	if report.Groups[0].Risk != RiskWarn {
		t.Fatalf("risk grup = %s", report.Groups[0].Risk)
	}
	if report.Bytes <= 0 {
		t.Fatal("total 0")
	}
}

func TestPlanDropsAncestor(t *testing.T) {
	root := t.TempDir()
	chromium := filepath.Join(root, "ms-playwright", "chromium-1")
	mkdir(t, chromium)
	mkfile(t, filepath.Join(chromium, "chrome"), 16384)

	list := []Rule{
		{ID: "parent", Group: "app", Path: root, Depth: 1, Risk: RiskSafe, Priority: -10},
		{ID: "child", Group: "dev.node", Path: filepath.Join(root, "ms-playwright"), Depth: 1, Risk: RiskWarn, Priority: 20},
	}

	report, err := Plan(context.Background(), list, PlanOptions{Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	var items []Item
	for _, g := range report.Groups {
		items = append(items, g.Items...)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v, mau 1 (induk dibuang)", items)
	}
	if items[0].Label != "chromium-1" || items[0].Risk != RiskWarn {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestPlanMissingRootSkipped(t *testing.T) {
	report, err := Plan(context.Background(), []Rule{
		{ID: "ghost", Group: "app", Path: "/path/yang/tidak/ada", Whole: true, Risk: RiskSafe},
	}, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 0 || report.Bytes != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestPlanCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Plan(ctx, []Rule{{ID: "x", Group: "app", Path: t.TempDir(), Whole: true}}, PlanOptions{})
	if err == nil {
		t.Fatal("mau error context")
	}
}

func TestParseJournalUsage(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"Archived and active journals take up 4.0G in the file system.", 4 << 30},
		{"take up 512.5M", int64(512.5 * (1 << 20))},
		{"0B", 0},
		{"tidak ada angka", 0},
	}
	for _, c := range cases {
		if got := parseJournalUsage(c.in); got != c.want {
			t.Errorf("parse(%q) = %d, mau %d", c.in, got, c.want)
		}
	}
}

func TestIsProtected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	protected := []string{"/", "/nix", "/nix/store", "/etc", "/home", home,
		filepath.Join(home, ".ssh"), filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, "Documents"), filepath.Join(home, ".config")}
	for _, p := range protected {
		if !IsProtected(p) {
			t.Errorf("%q mestinya protected", p)
		}
	}
	allowed := []string{
		filepath.Join(home, ".cache", "opencode"),
		filepath.Join(home, ".npm", "_cacache"),
		"/var/tmp/foo",
	}
	for _, p := range allowed {
		if IsProtected(p) {
			t.Errorf("%q mestinya boleh", p)
		}
	}
}

func TestDefaultRulesValid(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range DefaultRules() {
		if r.ID == "" || r.Group == "" || r.Path == "" {
			t.Errorf("rule tidak lengkap: %+v", r)
		}
		if seen[r.ID] {
			t.Errorf("id duplikat: %s", r.ID)
		}
		seen[r.ID] = true
		if r.Risk != RiskSafe && r.Risk != RiskWarn && r.Risk != RiskDanger && r.Risk != RiskBlocked {
			t.Errorf("risk tidak valid: %s", r.Risk)
		}
		if r.Probe == nil && !filepath.IsAbs(ExpandHome(r.Path)) {
			t.Errorf("path tidak absolut: %s", r.Path)
		}
		if r.Whole && IsProtected(r.Path) {
			t.Errorf("rule menyentuh path protected: %s (%s)", r.ID, r.Path)
		}
	}
}

func TestEnumerateDepth(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a", "b", "c"))
	mkfile(t, filepath.Join(root, "file.txt"), 100)

	if got := enumerate(root, 1, false, nil); len(got) != 1 || filepath.Base(got[0]) != "a" {
		t.Fatalf("depth 1 dirs = %v", got)
	}
	if got := enumerate(root, 2, false, nil); len(got) != 1 || filepath.Base(got[0]) != "b" {
		t.Fatalf("depth 2 = %v", got)
	}
	if got := enumerate(root, 1, true, nil); len(got) != 2 {
		t.Fatalf("depth 1 files = %v", got)
	}
}
