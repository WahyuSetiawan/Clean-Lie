package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
)

func TestDirenvAddScanCleanRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	proj := filepath.Join(home, "Projects", "demo")
	direnvDir := filepath.Join(proj, ".direnv")
	if err := os.MkdirAll(filepath.Join(direnvDir, "nix"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(direnvDir, "nix", "env"), make([]byte, 64<<10), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(home, ".cache", "direnv", "layouts")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "x"), make([]byte, 16<<10), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewApp("test", "test")
	if _, err := a.DirenvScan(); err == nil {
		t.Fatal("scan tanpa root harus menolak")
	}
	view, err := a.DirenvAddRoot(proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Config.DirenvRoots) != 1 || view.Config.DirenvRoots[0] != proj {
		t.Fatalf("roots = %+v", view.Config.DirenvRoots)
	}
	// Menambah root yang sama tidak menduplikasi.
	if view, err = a.DirenvAddRoot(proj); err != nil || len(view.Config.DirenvRoots) != 1 {
		t.Fatalf("duplikat root: %+v %v", view.Config.DirenvRoots, err)
	}
	if _, err := a.DirenvAddRoot(filepath.Join(home, "tidak-ada")); err == nil {
		t.Fatal("folder tidak ada harus ditolak")
	}

	rep, err := a.DirenvScan()
	if err != nil {
		t.Fatal(err)
	}
	var projectID, cacheID string
	for _, e := range rep.Entries {
		switch e.Kind {
		case "project":
			projectID = e.ID
			if e.Path != direnvDir || e.Project != proj {
				t.Fatalf("entry proyek = %+v", e)
			}
		case "cache":
			cacheID = e.ID
		}
	}
	if projectID == "" || cacheID == "" {
		t.Fatalf("entry tidak lengkap: %+v", rep.Entries)
	}

	// Cache 60 detik: hasil kedua tidak berubah.
	rep2, err := a.DirenvScan()
	if err != nil || rep2.Bytes != rep.Bytes {
		t.Fatalf("cache scan: %+v %v", rep2, err)
	}

	res, err := a.DirenvClean(DirenvCleanRequest{IDs: []string{projectID, cacheID}, Mode: "trash", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 || res.Items[0].Status != clean.StatusPlanned {
		t.Fatalf("dry-run = %+v", res.Items)
	}
	if _, err := os.Stat(direnvDir); err != nil {
		t.Fatal("dry-run tidak boleh menghapus")
	}
	if _, err := a.DirenvClean(DirenvCleanRequest{IDs: []string{"direnv:/tidak/ada"}, Mode: "trash"}); err == nil {
		t.Fatal("ID asing harus ditolak")
	}

	view, err = a.DirenvRemoveRoot(proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Config.DirenvRoots) != 0 {
		t.Fatalf("root belum terhapus: %+v", view.Config.DirenvRoots)
	}
	if !a.DirenvRefresh() {
		t.Fatal("DirenvRefresh harus true")
	}
}
