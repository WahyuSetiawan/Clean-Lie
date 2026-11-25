package nix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"4.0G":      4 << 30,
		"3.9 GiB":   4187593113,
		"123.4 MiB": 129394278,
		"500M":      500 << 20,
		"1,5G":      int64(1.5 * (1 << 30)),
		"512B":      512,
	}
	for in, want := range cases {
		got, ok := parseSize(in)
		if !ok || got != want {
			t.Errorf("parseSize(%q) = %d,%v; mau %d", in, got, ok, want)
		}
	}
	if _, ok := parseSize("tidak ada angka"); ok {
		t.Error("string tanpa ukuran harus gagal")
	}
}

func TestParsePathInfoS(t *testing.T) {
	out := "/nix/store/aaa  123\n/nix/store/bbb\t456\nbukan-path 99\n"
	got := parsePathInfoS(out)
	if got["/nix/store/aaa"] != 123 || got["/nix/store/bbb"] != 456 {
		t.Fatalf("hasil = %#v", got)
	}
	if len(got) != 2 {
		t.Fatalf("jumlah = %d", len(got))
	}
}

func TestParseDeadAndGC(t *testing.T) {
	out := "finding garbage collector roots...\n/nix/store/a\n/nix/store/b\n"
	if got := parseDeadPaths(out); len(got) != 2 || got[0] != "/nix/store/a" {
		t.Fatalf("dead = %#v", got)
	}
	for in, want := range map[string]int{
		"603 store paths would be deleted":      603,
		"12 store paths deleted, 1.2 GiB freed": 12,
		"0 store paths deleted":                 0,
	} {
		if got := parseGCDeletedCount(in); got != want {
			t.Errorf("parseGCDeletedCount(%q) = %d, mau %d", in, got, want)
		}
	}
}

func TestGenerationLinksAndProfile(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".local/state/nix/profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	paths := map[int]string{}
	for _, n := range []int{1, 2, 3} {
		p := filepath.Join(store, "gen-"+string(rune('0'+n)))
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		paths[n] = p
		if err := os.Symlink(p, filepath.Join(dir, "home-manager-"+string(rune('0'+n))+"-link")); err != nil {
			t.Fatal(err)
		}
	}
	profile := filepath.Join(dir, "home-manager")
	if err := os.Symlink("home-manager-3-link", profile); err != nil {
		t.Fatal(err)
	}

	links, current, err := generationLinks(profile)
	if err != nil {
		t.Fatal(err)
	}
	if current != 3 || len(links) != 3 || links[0].number != 3 {
		t.Fatalf("links=%+v current=%d", links, current)
	}

	fake := func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
		switch {
		case cmd[0] == "nix-env":
			return tools.Result{ExitCode: 0, Stdout: "" +
				"    1   2026-01-01 10:00:00\n" +
				"    2   2026-02-01 10:00:00\n" +
				"    3   2026-03-01 10:00:00   (current)\n"}
		case len(cmd) > 2 && cmd[0] == "nix" && cmd[1] == "path-info" && cmd[2] == "-S":
			var b strings.Builder
			for n, p := range paths {
				b.WriteString(p + "  " + string(rune('0'+n)) + "000\n")
			}
			return tools.Result{ExitCode: 0, Stdout: b.String()}
		}
		return tools.Result{ExitCode: 0}
	}

	info, gens := loadProfile(context.Background(), profileSpec{Name: "home-manager", Path: profile}, Options{
		Home: home, Run: fake,
		Now: func() time.Time { return time.Date(2026, 11, 25, 0, 0, 0, 0, time.Local) },
	}, true)
	if info.Error != "" || info.Count != 3 || info.Current != 3 {
		t.Fatalf("info = %+v", info)
	}
	if !gens[0].Current || gens[0].Number != 3 {
		t.Fatalf("generasi pertama = %+v", gens[0])
	}
	if gens[0].ClosureBytes != 3000 || gens[0].Date != "2026-03-01 10:00:00" {
		t.Fatalf("closure/tanggal = %+v", gens[0])
	}
	if !gens[2].Suggested || gens[2].Current {
		t.Fatalf("gen lama harus disarankan: %+v", gens[2])
	}
}

func TestDeadPaths(t *testing.T) {
	fake := func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
		switch {
		case cmd[0] == "nix-store":
			return tools.Result{ExitCode: 0, Stdout: "/nix/store/a\n/nix/store/b\n"}
		case cmd[0] == "nix" && len(cmd) > 2 && cmd[1] == "path-info" && cmd[2] == "--json":
			return tools.Result{ExitCode: 0, Stdout: `{"/nix/store/a":{"narSize":100},"/nix/store/b":{"narSize":250}}`}
		}
		return tools.Result{ExitCode: 1, Stderr: "tidak terduga"}
	}
	d := DeadPaths(context.Background(), Options{Run: fake})
	if d.Error != "" || d.Count != 2 || d.Bytes != 350 {
		t.Fatalf("dead = %+v", d)
	}
}

func TestDeleteGenerationsUser(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".local/state/nix/profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"1", "2"} {
		if err := os.Symlink("/nix/store/"+n, filepath.Join(dir, "profile-"+n+"-link")); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("profile-2-link", filepath.Join(dir, "profile")); err != nil {
		t.Fatal(err)
	}
	var ran []string
	fake := func(_ context.Context, cmd []string, _ time.Duration) tools.Result {
		ran = cmd
		if cmd[0] == "nix-env" && len(cmd) > 3 && cmd[len(cmd)-1] == "--list-generations" {
			return tools.Result{ExitCode: 0, Stdout: "1 2026-01-01 00:00:00\n2 2026-01-02 00:00:00 (current)\n"}
		}
		return tools.Result{ExitCode: 0}
	}
	res, err := DeleteGenerations(context.Background(), GenRequest{
		Profile: "user", Numbers: []int{1}, Confirm: "GC",
	}, Options{Home: home, Run: fake})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.Deleted != 1 {
		t.Fatalf("res = %+v", res)
	}
	if !strings.Contains(strings.Join(ran, " "), "--delete-generations 1") {
		t.Fatalf("command = %v", ran)
	}

	if _, err := DeleteGenerations(context.Background(), GenRequest{
		Profile: "user", Numbers: []int{2}, Confirm: "GC",
	}, Options{Home: home, Run: fake}); err == nil {
		t.Fatal("generasi current harus ditolak")
	}
	if _, err := DeleteGenerations(context.Background(), GenRequest{
		Profile: "user", Numbers: []int{1}, Confirm: "gc",
	}, Options{Home: home, Run: fake}); err == nil {
		t.Fatal("konfirmasi salah harus ditolak")
	}
}

func TestCollectGarbageNeedsSudo(t *testing.T) {
	res, err := CollectGarbage(context.Background(), "GC", Options{
		Run: func(context.Context, []string, time.Duration) tools.Result {
			t.Fatal("tidak boleh jalan")
			return tools.Result{}
		},
		Agent: func() bool { return false },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.NeedsSudo || !strings.Contains(res.SudoCommand, "nix-collect-garbage") {
		t.Fatalf("res = %+v", res)
	}
}

func TestVacuumJournalValidation(t *testing.T) {
	if _, err := VacuumJournal(context.Background(), "500M; rm -rf /", Options{}); err == nil {
		t.Fatal("target injeksi harus ditolak")
	}
}

func TestRealStatus(t *testing.T) {
	if os.Getenv("NIX_TEST_REAL") == "" {
		t.Skip("set NIX_TEST_REAL=1 untuk membaca mesin nyata")
	}
	st := ReadStatus(context.Background(), Options{})
	if len(st.Generations) == 0 {
		t.Fatalf("tidak ada generasi: %+v", st)
	}
	if st.StoreBytes <= 0 {
		t.Fatalf("store size 0: %+v", st.Errors)
	}
	t.Logf("store=%d paths=%d gens=%d profiles=%d journal=%d errors=%v dur=%dms",
		st.StoreBytes, st.StorePaths, len(st.Generations), len(st.Profiles), st.JournalBytes, st.Errors, st.DurationMs)
	for _, p := range st.Profiles {
		t.Logf("profile %s: count=%d current=%d oldest=%s err=%s", p.Name, p.Count, p.Current, p.Oldest, p.Error)
	}
	d := DeadPaths(context.Background(), Options{})
	if d.Count == 0 {
		t.Fatalf("dead paths 0: %+v", d)
	}
	t.Logf("dead=%d bytes=%d source=%s err=%s dur=%dms", d.Count, d.Bytes, d.Source, d.Error, d.DurationMs)
}

func TestVacuumTargetNormalize(t *testing.T) {
	for in, want := range map[string]string{"500 MB": "500M", "1gb": "1G", "2 G": "2G", "500M": "500M"} {
		if got := normalizeVacuumTarget(in); got != want {
			t.Fatalf("normalize(%q) = %q, mau %q", in, got, want)
		}
	}
}
