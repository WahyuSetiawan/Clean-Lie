package analysis

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

func dirBlock(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return int64(st.Blocks) * 512
}

func writeFile(t *testing.T, path string, size int, fill byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, size)
	for i := range b {
		b[i] = fill
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanTree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "one.bin"), 256<<10, 'a')
	writeFile(t, filepath.Join(root, "b", "two.bin"), 512<<10, 'b')
	writeFile(t, filepath.Join(root, "b", "sub", "three.bin"), 128<<10, 'c')
	if err := os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".git", "objects", "x"), 256<<10, 'x')

	tr, err := ScanTree(context.Background(), Options{Root: root, Depth: 2, Now: func() time.Time { return time.Now() }})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Root.Children) != 2 {
		t.Fatalf("children = %+v", tr.Root.Children)
	}
	if tr.Root.Children[0].Name != "b" {
		t.Fatalf("urut ukuran = %+v", tr.Root.Children)
	}
	var b Node
	for _, n := range tr.Root.Children {
		if n.Name == "b" {
			b = n
		}
		if n.Name == ".git" {
			t.Fatal(".git harus dilewati")
		}
	}
	if len(b.Children) != 2 || b.Children[0].Name != "two.bin" {
		t.Fatalf("children b = %+v", b.Children)
	}
	// tree memakai allocated block; bandingkan dengan fsx.Usage (tanpa .git).
	want, files, err := fsx.Usage(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	git, _, err := fsx.Usage(context.Background(), filepath.Join(root, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	rootBlock := dirBlock(t, root)
	gitBlock := dirBlock(t, filepath.Join(root, ".git"))
	wantTree := want + rootBlock - git - gitBlock
	if diff := tr.Bytes - wantTree; diff < -(16<<10) || diff > 16<<10 || tr.Files != files-1 {
		t.Fatalf("tree = %d/%d, fsx = %d/%d (root %d, git %d)", tr.Bytes, tr.Files, wantTree, files-1, rootBlock, git)
	}
}

func TestScanOverview(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "work")
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.Local)
	old := now.Add(-400 * 24 * time.Hour)

	writeFile(t, filepath.Join(root, "large.bin"), 2<<20, 'L')
	writeFile(t, filepath.Join(root, "old", "stale.bin"), 2<<20, 'S')
	writeFile(t, filepath.Join(root, "proj-lama", "node_modules", "a.js"), 1<<20, 'a')
	writeFile(t, filepath.Join(root, "proj-lama", "node_modules", "nested", "node_modules", "b.js"), 512<<10, 'b')
	writeFile(t, filepath.Join(root, "proj-baru", "node_modules", "c.js"), 1<<20, 'c')
	writeFile(t, filepath.Join(root, ".git", "config"), 2<<20, 'g')

	for _, p := range []string{
		filepath.Join(root, "old", "stale.bin"),
		filepath.Join(root, "old"),
		filepath.Join(root, "proj-lama"),
		filepath.Join(root, "proj-lama", "node_modules"),
	} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	ov, err := ScanOverview(context.Background(), Options{Root: root, Now: func() time.Time { return now }, LargeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Large) < 2 || ov.Large[0].Path != filepath.Join(root, "large.bin") {
		t.Fatalf("large = %+v", ov.Large)
	}
	if len(ov.Stale) != 1 || ov.Stale[0].Path != filepath.Join(root, "old", "stale.bin") {
		t.Fatalf("stale = %+v", ov.Stale)
	}
	if len(ov.Artifacts) != 2 {
		t.Fatalf("artifacts = %+v", ov.Artifacts)
	}
	var lama, baru *Artifact
	for i := range ov.Artifacts {
		if ov.Artifacts[i].Project == filepath.Join(root, "proj-lama") {
			lama = &ov.Artifacts[i]
		}
		if ov.Artifacts[i].Project == filepath.Join(root, "proj-baru") {
			baru = &ov.Artifacts[i]
		}
	}
	if lama == nil || lama.Active {
		t.Fatalf("proyek lama = %+v", lama)
	}
	if baru == nil || !baru.Active {
		t.Fatalf("proyek baru = %+v", baru)
	}
	if lama.Bytes < 1<<20+512<<10 {
		t.Fatalf("nested node_modules tidak ikut dihitung: %+v", lama)
	}
}

func TestFindDuplicates(t *testing.T) {
	root := t.TempDir()
	const size = 1200 << 10
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	writeFile(t, filepath.Join(root, "a.bin"), size, 'x')
	writeFile(t, filepath.Join(root, "b.bin"), size, 'x')
	writeFile(t, filepath.Join(root, "c.bin"), size, 'x')
	writeFile(t, filepath.Join(root, "mirip.bin"), size, 'x')
	f, err := os.OpenFile(filepath.Join(root, "mirip.bin"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte{'y'}, size-1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	writeFile(t, filepath.Join(root, "kecil.bin"), 1000, 'x')
	for i, name := range []string{"a.bin", "b.bin", "c.bin"} {
		p := filepath.Join(root, name)
		tm := old.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(p, tm, tm); err != nil {
			t.Fatal(err)
		}
	}

	dups, err := FindDuplicates(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(dups.Groups) != 1 {
		t.Fatalf("groups = %+v", dups.Groups)
	}
	g := dups.Groups[0]
	if len(g.Files) != 3 || !g.Files[0].Keep || g.Files[0].Path != filepath.Join(root, "a.bin") {
		t.Fatalf("group = %+v", g)
	}
	if g.Wasted != size*2 || dups.Wasted != size*2 {
		t.Fatalf("wasted = %d/%d", g.Wasted, dups.Wasted)
	}
	if dups.Candidates != 4 {
		t.Fatalf("candidates = %d", dups.Candidates)
	}
	if dups.Hashed < 4 {
		t.Fatalf("hashed = %d", dups.Hashed)
	}
}

func TestScanTreeHardlinkAndNestedGit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "one.bin"), 1<<20, 'h')
	if err := os.MkdirAll(filepath.Join(root, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(root, "a", "one.bin"), filepath.Join(root, "b", "hard.bin")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "b", ".git", "objects", "x"), 2<<20, 'g')

	tr, err := ScanTree(context.Background(), Options{Root: root, Depth: 2})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Files != 1 {
		t.Fatalf("hardlink dihitung %d file, mau 1", tr.Files)
	}
	all, _, err := fsx.Usage(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	git, _, err := fsx.Usage(context.Background(), filepath.Join(root, "b", ".git"))
	if err != nil {
		t.Fatal(err)
	}
	rootBlock := dirBlock(t, root)
	gitBlock := dirBlock(t, filepath.Join(root, "b", ".git"))
	wantTree := all + rootBlock - git - gitBlock
	if diff := tr.Bytes - wantTree; diff < -(16<<10) || diff > 16<<10 {
		t.Fatalf("tree %d != fsx tanpa .git %d", tr.Bytes, wantTree)
	}
}

func TestScanTreeExcluded(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "one.bin"), 256<<10, 'a')
	writeFile(t, filepath.Join(root, "skip", "two.bin"), 512<<10, 'b')
	tr, err := ScanTree(context.Background(), Options{Root: root, Depth: 1, Excluded: []string{filepath.Join(root, "skip")}})
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range tr.Root.Children {
		if child.Name == "skip" {
			t.Fatalf("folder skip masih ada: %+v", child)
		}
	}
	if tr.Bytes >= 512<<10 {
		t.Fatalf("bytes = %d, file yang dikecualikan masih dihitung", tr.Bytes)
	}
}

func TestScanOverviewExcluded(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "big.bin"), 2<<20, 'a')
	writeFile(t, filepath.Join(root, "skip", "big.bin"), 3<<20, 'b')
	ov, err := ScanOverview(context.Background(), Options{
		Root: root, LargeBytes: 1 << 20,
		Excluded: []string{filepath.Join(root, "skip")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range ov.Large {
		if filepath.Dir(f.Path) == filepath.Join(root, "skip") {
			t.Fatalf("file excluded muncul: %s", f.Path)
		}
	}
}

func TestScanOverviewExtraRoot(t *testing.T) {
	root := t.TempDir()
	extra := t.TempDir()
	writeFile(t, filepath.Join(root, "big.bin"), 2<<20, 'a')
	writeFile(t, filepath.Join(extra, "proj", "node_modules", "x.bin"), 1<<20, 'b')
	ov, err := ScanOverview(context.Background(), Options{
		Root: root, LargeBytes: 1 << 20, ExtraRoots: []string{extra},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range ov.Artifacts {
		if a.Path == filepath.Join(extra, "proj", "node_modules") {
			found = true
		}
	}
	if !found {
		t.Fatalf("artefak dari extra root tidak ditemukan: %+v", ov.Artifacts)
	}
}
