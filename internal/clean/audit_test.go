package clean

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditAppendRead(t *testing.T) {
	home := setupHome(t)
	path := filepath.Join(home, "state", "history.jsonl")

	if got, err := ReadAudit(path, 10); err != nil || len(got) != 0 {
		t.Fatalf("file belum ada: %v %v", got, err)
	}

	for i, id := range []string{"a", "b", "c"} {
		err := AppendAudit(path, Entry{ID: id, Path: "/x/" + id, Status: StatusRemoved, Freed: int64(i + 1)})
		if err != nil {
			t.Fatal(err)
		}
	}

	all, err := ReadAudit(path, 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("all = %+v err=%v", all, err)
	}
	last, err := ReadAudit(path, 2)
	if err != nil || len(last) != 2 || last[0].ID != "b" || last[1].ID != "c" {
		t.Fatalf("last = %+v err=%v", last, err)
	}

	// baris rusak harus dilewati, bukan menggagalkan pembacaan.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{bukan json}\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	all, err = ReadAudit(path, 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("setelah baris rusak: %+v err=%v", all, err)
	}
}

func TestDefaultAuditPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	want := filepath.Join(home, ".local", "state", "cleanlie", "history.jsonl")
	if got := DefaultAuditPath(); got != want {
		t.Fatalf("path = %q, mau %q", got, want)
	}
}
