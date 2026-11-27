package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

func trashHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("CLEANLIE_NO_GIO", "1")
	return home
}

func trashEntry(t *testing.T, home, name string) string {
	t.Helper()
	trash := filepath.Join(home, ".local", "share", "Trash")
	if err := os.MkdirAll(filepath.Join(trash, "files"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(trash, "info"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trash, "files", name), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(home, name)
	body := "[Trash Info]\nPath=" + original + "\nDeletionDate=2026-01-01T00:00:00\n"
	if err := os.WriteFile(filepath.Join(trash, "info", name+".trashinfo"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return original
}

func TestTrashListRestoreDelete(t *testing.T) {
	home := trashHome(t)
	original := trashEntry(t, home, "one")
	a := NewApp("test", "x")

	rep, err := a.TrashList()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 1 || rep.Entries[0].Name != "one" {
		t.Fatalf("list salah: %+v", rep)
	}
	id := rep.Entries[0].ID

	if _, err := a.TrashDelete([]string{id}, false, false); err == nil {
		t.Fatal("delete tanpa konfirmasi harus error")
	}
	planned, err := a.TrashRestore([]string{id}, true)
	if err != nil || planned.Restored != 1 {
		t.Fatalf("dry run restore salah: %+v %v", planned, err)
	}
	if _, err := os.Stat(original); err == nil {
		t.Fatal("dry run restore memindahkan file")
	}

	res, err := a.TrashRestore([]string{id}, false)
	if err != nil || res.Restored != 1 {
		t.Fatalf("restore salah: %+v %v", res, err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("file tidak kembali: %v", err)
	}
	sessions, err := a.HistorySessions(0)
	if err != nil || len(sessions.Sessions) == 0 {
		t.Fatalf("history kosong: %+v %v", sessions, err)
	}
	if sessions.Sessions[0].Action != "restore" {
		t.Fatalf("aksi sesi = %q", sessions.Sessions[0].Action)
	}

	original2 := trashEntry(t, home, "two")
	rep, err = a.TrashList()
	if err != nil {
		t.Fatal(err)
	}
	var id2 string
	for _, e := range rep.Entries {
		if e.Name == "two" {
			id2 = e.ID
		}
	}
	if id2 == "" {
		t.Fatal("entry two tidak ditemukan")
	}
	res, err = a.TrashDelete([]string{id2}, true, false)
	if err != nil || res.Deleted != 1 {
		t.Fatalf("delete salah: %+v %v", res, err)
	}
	if _, err := os.Stat(original2); err == nil {
		t.Fatal("delete tidak menghapus file")
	}
}

func TestTrashEmpty(t *testing.T) {
	home := trashHome(t)
	trashEntry(t, home, "a")
	trashEntry(t, home, "b")
	a := NewApp("test", "x")

	if _, err := a.TrashEmpty("salah", false); err == nil {
		t.Fatal("empty tanpa konfirmasi harus error")
	}
	planned, err := a.TrashEmpty("", true)
	if err != nil || planned.Deleted != 2 {
		t.Fatalf("empty dry run salah: %+v %v", planned, err)
	}
	res, err := a.TrashEmpty("KOSONGKAN", false)
	if err != nil || res.Deleted != 2 {
		t.Fatalf("empty salah: %+v %v", res, err)
	}
}

func TestUndoSession(t *testing.T) {
	home := trashHome(t)
	root := filepath.Join(home, ".cache")
	target := filepath.Join(root, "junk")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "f"), make([]byte, 32*1024), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp("test", "x")
	a.scanItems = map[string]rules.Item{
		"junk": {ID: "junk", Path: target, Root: root, Label: "junk"},
	}
	cleanRes, err := a.Clean(CleanRequest{IDs: []string{"junk"}, Mode: "trash"})
	if err != nil || cleanRes.Batch == "" || cleanRes.Items[0].Status != clean.StatusRemoved {
		t.Fatalf("clean salah: %+v %v", cleanRes, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target masih ada: %v", err)
	}

	planned, err := a.UndoSession(cleanRes.Batch, true)
	if err != nil || planned.Restored != 1 {
		t.Fatalf("undo dry run salah: %+v %v", planned, err)
	}
	res, err := a.UndoSession(cleanRes.Batch, false)
	if err != nil || res.Restored != 1 {
		t.Fatalf("undo salah: %+v %v", res, err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target tidak kembali: %v", err)
	}
}
