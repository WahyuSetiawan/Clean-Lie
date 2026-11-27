package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func auditFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	return filepath.Join(dir, "cleanlie", "history.jsonl")
}

func sampleEntry(batch, id, path string) Entry {
	return Entry{
		Time:        "2026-01-01T00:00:00Z",
		Mode:        ModeTrash,
		ID:          id,
		Path:        path,
		BytesBefore: 100,
		BytesAfter:  0,
		Freed:       0,
		Status:      StatusRemoved,
		TrashPath:   "/trash/" + id,
		Batch:       batch,
	}
}

func TestNewBatchFormat(t *testing.T) {
	b := NewBatch("undo-")
	if !strings.HasPrefix(b, "undo-") || !strings.Contains(b, "-") {
		t.Fatalf("batch = %q", b)
	}
	if b == NewBatch("undo-") {
		t.Fatal("batch harus unik")
	}
}

func TestAuditBatchAndSessions(t *testing.T) {
	path := auditFixture(t)
	entries := []Entry{
		sampleEntry("b1", "a", "/tmp/a"),
		sampleEntry("b1", "b", "/tmp/b"),
		sampleEntry("b2", "c", "/tmp/c"),
	}
	if err := AppendAuditBatch(path, entries); err != nil {
		t.Fatal(err)
	}

	page, err := ReadSessions(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Sessions) != 2 {
		t.Fatalf("sessions = %d, want 2: %+v", len(page.Sessions), page.Sessions)
	}
	if page.Sessions[0].ID != "b2" {
		t.Fatalf("urutan terbaru salah: %+v", page.Sessions)
	}
	for _, s := range page.Sessions {
		if s.Removed != len(s.Items) || s.Restorable != len(s.Items) {
			t.Fatalf("hitungan sesi salah: %+v", s)
		}
		if s.RestorableBytes <= 0 {
			t.Fatalf("restorable bytes salah: %+v", s)
		}
	}

	got, ok, err := ReadSessionEntries(path, "b1")
	if err != nil || !ok || len(got) != 2 {
		t.Fatalf("session b1 = %+v ok=%v err=%v", got, ok, err)
	}
}

func TestSessionsLegacyAndRestorableRules(t *testing.T) {
	path := auditFixture(t)
	now := "2026-01-02T03:04:05Z"
	legacy := []Entry{
		{Time: now, Mode: ModeTrash, Status: StatusRemoved, Path: "/tmp/a", TrashPath: "/trash/a"},
		{Time: now, Mode: ModeTrash, Status: StatusRemoved, Path: "/tmp/b", TrashPath: "/trash/b"},
		{Time: "2026-01-02T03:04:06Z", Mode: ModeTrash, Status: StatusRemoved, Path: "/tmp/c", CommandLine: "rm -rf /tmp/c"},
		{Time: "2026-01-02T03:04:07Z", Mode: ModeTrash, Status: StatusRemoved, Path: "/tmp/d", CommandLine: "rm", TrashPath: "/trash/d"},
		{Time: "2026-01-02T03:04:08Z", Mode: ModeTrash, Status: StatusRemoved, Path: "/tmp/e", Action: "restore", TrashPath: "/trash/e"},
		{Time: "2026-01-02T03:04:09Z", Mode: ModePermanent, Status: StatusRemoved, Path: "/tmp/f"},
	}
	if err := AppendAuditBatch(path, legacy); err != nil {
		t.Fatal(err)
	}
	page, err := ReadSessions(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	// now (2 item), command (1), fallback command (1), restore (1), permanent (1)
	if len(page.Sessions) != 5 {
		t.Fatalf("sessions = %d, want 5: %+v", len(page.Sessions), page.Sessions)
	}
	var legacySession *Session
	for i := range page.Sessions {
		if page.Sessions[i].Batch == "" && len(page.Sessions[i].Items) == 2 {
			legacySession = &page.Sessions[i]
		}
	}
	if legacySession == nil {
		t.Fatalf("sesi legacy tidak ditemukan: %+v", page.Sessions)
	}
	if legacySession.ID == "" || legacySession.Restorable != 2 {
		t.Fatalf("sesi legacy salah: %+v", legacySession)
	}
	if _, ok, err := ReadSessionEntries(path, legacySession.ID); err != nil || !ok {
		t.Fatalf("undo legacy gagal: ok=%v err=%v", ok, err)
	}
}

func TestSessionsLimitAndTail(t *testing.T) {
	path := auditFixture(t)
	for i := 0; i < 4; i++ {
		e := sampleEntry(NewBatch(""), string(rune('a'+i)), "/tmp/x")
		e.Time = time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC).Format(time.RFC3339)
		if err := AppendAudit(path, e); err != nil {
			t.Fatal(err)
		}
	}
	page, err := ReadSessions(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Sessions) != 2 || !page.Truncated {
		t.Fatalf("limit salah: %+v", page)
	}
	if page.Sessions[0].ID == page.Sessions[1].ID {
		t.Fatalf("sesi duplikat: %+v", page.Sessions)
	}

	tail, truncated, err := ReadAuditTail(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(tail) != 2 {
		t.Fatalf("tail = %d truncated=%v", len(tail), truncated)
	}
}

func TestRotateAudit(t *testing.T) {
	path := auditFixture(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, auditMaxBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := RotateAudit(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("arsip tidak ada: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Size() != 0 {
		t.Fatalf("file baru salah: %v %v", fi, err)
	}
}

func TestAppendAuditSingle(t *testing.T) {
	path := auditFixture(t)
	e := sampleEntry("b", "a", "/tmp/a")
	if err := AppendAudit(path, e); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAudit(path, 0)
	if err != nil || len(got) != 1 || got[0].Batch != "b" {
		t.Fatalf("append/read salah: %+v %v", got, err)
	}
}
