package direnv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func entryByPath(entries []Entry, path string) *Entry {
	for i := range entries {
		if entries[i].Path == path {
			return &entries[i]
		}
	}
	return nil
}

func TestScanFindsDirenvAndSkipsHeavyDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "Projects")

	writeFile(t, filepath.Join(root, "proj-a", DirName, "flake-profile", "bin", "x"), 64<<10)
	writeFile(t, filepath.Join(root, "proj-b", DirName, "nix", "env"), 32<<10)
	writeFile(t, filepath.Join(root, "proj-b", "node_modules", "dep", DirName, "x"), 128<<10)
	writeFile(t, filepath.Join(root, ".git", DirName, "x"), 128<<10)
	writeFile(t, filepath.Join(root, "proj-c", DirName, "x"), 16<<10)
	// Symlink ke luar tidak boleh diikuti.
	outside := filepath.Join(home, "outside.bin")
	writeFile(t, outside, 1<<20)
	if err := os.Symlink(outside, filepath.Join(root, "proj-a", DirName, "outside-link")); err != nil {
		t.Fatal(err)
	}
	// Cache global.
	writeFile(t, filepath.Join(home, ".cache", "direnv", "layouts", "x"), 16<<10)
	writeFile(t, filepath.Join(home, ".cache", "nix-direnv", "profile"), 8<<10)

	old := time.Now().Add(-30 * 24 * time.Hour)
	for _, p := range []string{
		filepath.Join(root, "proj-a", DirName),
		filepath.Join(root, "proj-b", DirName),
	} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	missing := filepath.Join(home, "tidak-ada")
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.Local)
	rep, err := Scan(context.Background(), Options{
		Roots:    []string{root, missing, root},
		Excluded: []string{filepath.Join(root, "proj-c")},
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Roots) != 2 || rep.Roots[0] != root || rep.Roots[1] != missing {
		t.Fatalf("roots = %v", rep.Roots)
	}
	if len(rep.Missing) != 1 || rep.Missing[0] != missing {
		t.Fatalf("missing = %v", rep.Missing)
	}
	if len(rep.Entries) != 4 {
		t.Fatalf("entries = %+v", rep.Entries)
	}
	if e := entryByPath(rep.Entries, filepath.Join(root, "proj-c", DirName)); e != nil {
		t.Fatalf("path excluded ikut terpindai: %+v", e)
	}
	if e := entryByPath(rep.Entries, filepath.Join(root, "proj-b", "node_modules", "dep", DirName)); e != nil {
		t.Fatalf("node_modules ikut dipindai: %+v", e)
	}
	if e := entryByPath(rep.Entries, filepath.Join(root, ".git", DirName)); e != nil {
		t.Fatalf(".git ikut dipindai: %+v", e)
	}
	a := entryByPath(rep.Entries, filepath.Join(root, "proj-a", DirName))
	if a == nil {
		t.Fatal("proj-a/.direnv tidak ditemukan")
	}
	if a.Kind != "project" || a.Project != filepath.Join(root, "proj-a") {
		t.Fatalf("proj-a = %+v", a)
	}
	if a.Active {
		t.Fatalf("proj-a harus tidak aktif: %+v", a)
	}
	if a.Bytes <= 0 || a.Bytes >= 1<<20 {
		t.Fatalf("symlink diikuti? bytes = %d", a.Bytes)
	}
	cache := entryByPath(rep.Entries, filepath.Join(home, ".cache", "direnv"))
	if cache == nil || cache.Kind != "cache" {
		t.Fatalf("cache direnv = %+v", cache)
	}
	if rep.Bytes <= 0 || rep.Files == 0 {
		t.Fatalf("total = %d/%d", rep.Bytes, rep.Files)
	}
	// Urutan: terbesar dulu.
	if rep.Entries[0].Bytes < rep.Entries[len(rep.Entries)-1].Bytes {
		t.Fatalf("tidak terurut: %+v", rep.Entries)
	}
}

func TestScanRootIsDirenv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "proj", DirName)
	writeFile(t, filepath.Join(dir, "x"), 8<<10)

	rep, err := Scan(context.Background(), Options{Roots: []string{dir}, SkipCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 1 || rep.Entries[0].Path != dir {
		t.Fatalf("entries = %+v", rep.Entries)
	}
	if rep.Entries[0].Active != true {
		t.Fatalf("folder baru harus aktif: %+v", rep.Entries[0])
	}
}

func TestScanNoRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rep, err := Scan(context.Background(), Options{SkipCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 0 || rep.Bytes != 0 || len(rep.Roots) != 0 {
		t.Fatalf("report = %+v", rep)
	}
}
