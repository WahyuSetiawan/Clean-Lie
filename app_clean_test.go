package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

func TestCleanWithoutScan(t *testing.T) {
	a := NewApp("test", "x")
	if _, err := a.Clean(CleanRequest{IDs: []string{"x"}}); err == nil {
		t.Fatal("mau error karena belum scan")
	}
}

func TestCleanPermanentRequiresConfirm(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	root := filepath.Join(home, ".cache")
	path := filepath.Join(root, "jit")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "f"), make([]byte, 32*1024), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewApp("test", "x")
	a.scanItems = map[string]rules.Item{
		"jit": {ID: "jit", Path: path, Root: root, Label: "jit", Bytes: 1},
	}

	if _, err := a.Clean(CleanRequest{IDs: []string{"jit"}, Mode: "permanent"}); err == nil {
		t.Fatal("hapus permanen tanpa konfirmasi harus ditolak")
	}

	res, err := a.Clean(CleanRequest{IDs: []string{"jit"}, Mode: "permanent", ConfirmPermanent: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Freed <= 0 || len(res.Items) != 1 || res.Items[0].Status != clean.StatusRemoved {
		t.Fatalf("result = %+v", res)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path masih ada: %v", err)
	}
}

func TestCleanUnknownID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := NewApp("test", "x")
	a.scanItems = map[string]rules.Item{"known": {ID: "known", Path: filepath.Join(home, "x"), Root: home}}

	if _, err := a.Clean(CleanRequest{IDs: []string{"tidak-ada"}, Mode: "trash"}); err == nil {
		t.Fatal("ID tak dikenal harus ditolak")
	}
}

func TestCleanDryRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".cache")
	path := filepath.Join(root, "keep")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	a := NewApp("test", "x")
	a.scanItems = map[string]rules.Item{
		"keep": {ID: "keep", Path: path, Root: root},
	}
	res, err := a.Clean(CleanRequest{IDs: []string{"keep"}, Mode: "trash", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Status != clean.StatusPlanned {
		t.Fatalf("status = %s", res.Items[0].Status)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry run menghapus: %v", err)
	}
}
