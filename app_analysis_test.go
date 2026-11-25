package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
)

func TestAnalysisTreeOverviewClean(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "proj", "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "proj", "node_modules", "pkg", "x.js"), make([]byte, 2<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp("test", "test")

	tr, err := a.AnalysisTree(home, 1)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Bytes <= 0 || len(tr.Root.Children) == 0 {
		t.Fatalf("tree kosong: %+v", tr)
	}
	tr2, err := a.AnalysisTree(home, 1)
	if err != nil || tr2.Bytes != tr.Bytes {
		t.Fatalf("cache tree: %+v %v", tr2, err)
	}

	ov, err := a.AnalysisOverview(home)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, art := range ov.Artifacts {
		if art.Name == "node_modules" {
			id = "artifact:" + art.Path
		}
	}
	if id == "" {
		t.Fatalf("artefak tidak ditemukan: %+v", ov.Artifacts)
	}

	res, err := a.AnalysisClean(AnalysisCleanRequest{IDs: []string{id}, Mode: "trash", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].Status != clean.StatusPlanned {
		t.Fatalf("dry-run = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(home, "proj", "node_modules")); err != nil {
		t.Fatal("dry-run tidak boleh menghapus")
	}
	if _, err := a.AnalysisClean(AnalysisCleanRequest{IDs: []string{"artifact:/tidak/ada"}, Mode: "trash"}); err == nil {
		t.Fatal("ID asing harus ditolak")
	}

	vol, err := a.DiskStatus()
	if err != nil || vol.Total <= 0 {
		t.Fatalf("disk = %+v %v", vol, err)
	}
}
