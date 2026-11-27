package main

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/insight"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
	"github.com/Juragan-Koding/cleanlie/internal/schedule"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunCLIScanJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.bin"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	var handled bool
	var code int
	out := captureStdout(t, func() {
		handled, code = runCLI([]string{"--scan-only", "--path", dir, "--json"})
	})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}

	var report rules.Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON tidak valid: %v\n%s", err, out)
	}
	if report.Bytes <= 0 || len(report.Groups) != 1 || report.Groups[0].ID != "scan" {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunCLIDryRunHuman(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.bin"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	var handled bool
	var code int
	out := captureStdout(t, func() {
		handled, code = runCLI([]string{"--dry-run", "--scan-only", "--path", dir})
	})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
	if !strings.Contains(out, "dry run") || !strings.Contains(out, "Hasil scan") {
		t.Fatalf("output = %s", out)
	}
}

func TestRunCLINoArgsIsGUI(t *testing.T) {
	handled, _ := runCLI(nil)
	if handled {
		t.Fatal("tanpa argumen harus membuka GUI")
	}
}

func TestRunCLIVersion(t *testing.T) {
	out := captureStdout(t, func() {
		handled, code := runCLI([]string{"--version"})
		if !handled || code != 0 {
			t.Fatalf("handled=%v code=%d", handled, code)
		}
	})
	if !strings.Contains(out, "cleanlie") {
		t.Fatalf("output = %q", out)
	}
}

func seedTrash(t *testing.T) (home, data, name, id string) {
	t.Helper()
	home = t.TempDir()
	data = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("CLEANLIE_NO_GIO", "1")

	root := filepath.Join(data, "Trash")
	if err := os.MkdirAll(filepath.Join(root, "files"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "info"), 0o700); err != nil {
		t.Fatal(err)
	}
	name = "sample"
	original := filepath.Join(home, "orig", name)
	if err := os.MkdirAll(filepath.Dir(original), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "files", name), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	info := "[Trash Info]\nPath=" + url.PathEscape(original) + "\nDeletionDate=2026-11-25T10:00:00\n"
	if err := os.WriteFile(filepath.Join(root, "info", name+".trashinfo"), []byte(info), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, data, name, trash.EntryID(root, name)
}

func TestRunCLITrashListAndRestore(t *testing.T) {
	home, _, name, id := seedTrash(t)

	out := captureStdout(t, func() {
		handled, code := runCLI([]string{"--trash-list", "--json", "--no-size"})
		if !handled || code != 0 {
			t.Fatalf("list handled=%v code=%d", handled, code)
		}
	})
	var rep trash.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("JSON tidak valid: %v\n%s", err, out)
	}
	if len(rep.Entries) != 1 || rep.Entries[0].ID != id {
		t.Fatalf("entries = %+v (id=%s)", rep.Entries, id)
	}

	out = captureStdout(t, func() {
		handled, code := runCLI([]string{"--trash-restore", id, "--yes", "--json"})
		if !handled || code != 0 {
			t.Fatalf("restore handled=%v code=%d", handled, code)
		}
	})
	var res trash.ActionResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("JSON tidak valid: %v\n%s", err, out)
	}
	if res.Restored != 1 || res.Failed != 0 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(home, "orig", name)); err != nil {
		t.Fatalf("file tidak kembali: %v", err)
	}
}

func TestRunCLIDestructiveNeedsYes(t *testing.T) {
	_, _, name, _ := seedTrash(t)

	captureStdout(t, func() {
		handled, code := runCLI([]string{"--trash-empty"})
		if !handled || code != 2 {
			t.Fatalf("empty tanpa yes: handled=%v code=%d", handled, code)
		}
	})
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files", name)); err != nil {
		t.Fatalf("trash berubah tanpa --yes: %v", err)
	}

	captureStdout(t, func() {
		handled, code := runCLI([]string{"--trash-empty", "--yes", "--json"})
		if !handled || code != 0 {
			t.Fatalf("empty dengan yes: handled=%v code=%d", handled, code)
		}
	})
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files", name)); !os.IsNotExist(err) {
		t.Fatalf("file masih ada setelah empty: %v", err)
	}
}

func TestRunCLIHistoryJSON(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	path := clean.DefaultAuditPath()
	if err := clean.AppendAudit(path, clean.Entry{
		Time: time.Now().Format(time.RFC3339), Mode: clean.ModeTrash, ID: "x", Path: "/tmp/x",
		BytesBefore: 10, Status: clean.StatusRemoved, Batch: "batch-test",
	}); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		handled, code := runCLI([]string{"--history", "--limit", "5", "--json"})
		if !handled || code != 0 {
			t.Fatalf("history handled=%v code=%d", handled, code)
		}
	})
	var page clean.SessionPage
	if err := json.Unmarshal([]byte(out), &page); err != nil {
		t.Fatalf("JSON tidak valid: %v\n%s", err, out)
	}
	if len(page.Sessions) != 1 || page.Sessions[0].Batch != "batch-test" {
		t.Fatalf("page = %+v", page)
	}
}

func TestRunCLIInsightAndReport(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	out := captureStdout(t, func() {
		handled, code := runCLI([]string{"--insight", "--json"})
		if !handled || code != 0 {
			t.Fatalf("insight handled=%v code=%d", handled, code)
		}
	})
	var rep insight.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("JSON insight tidak valid: %v\n%s", err, out)
	}
	if rep.Week.Days != 7 || rep.Version == "" {
		t.Fatalf("report = %+v", rep)
	}

	md := captureStdout(t, func() {
		handled, code := runCLI([]string{"--report", "md"})
		if !handled || code != 0 {
			t.Fatalf("report handled=%v code=%d", handled, code)
		}
	})
	if !strings.Contains(md, "Cleanlie") || !strings.Contains(md, "Tren disk") {
		t.Fatalf("md = %s", md)
	}

	captureStdout(t, func() {
		handled, code := runCLI([]string{"--report", "xml"})
		if !handled || code != 2 {
			t.Fatalf("format asing handled=%v code=%d", handled, code)
		}
	})
}

func TestRunCLIAutoCleanRequiresYes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	handled, code := runCLI([]string{"--auto-clean"})
	if !handled || code != 2 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
}

func TestRunCLIAutoClean(t *testing.T) {
	home := t.TempDir()
	state := t.TempDir()
	data := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("CLEANLIE_NO_GIO", "1")
	cache := filepath.Join(home, ".cache", "opencode", "blob")
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Dry-run tidak boleh mengubah apa pun.
	out := captureStdout(t, func() {
		if _, code := runCLI([]string{"--auto-clean", "--dry-run", "--json"}); code != 0 {
			t.Fatalf("dry-run code=%d", code)
		}
	})
	var dry clean.Result
	if err := json.Unmarshal([]byte(out), &dry); err != nil {
		t.Fatalf("json dry: %v\n%s", err, out)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("dry-run menghapus file: %v", err)
	}

	out = captureStdout(t, func() {
		if _, code := runCLI([]string{"--auto-clean", "--yes", "--json"}); code != 0 {
			t.Fatalf("auto-clean code=%d", code)
		}
	})
	var res clean.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if res.Mode != clean.ModeTrash || len(res.Items) == 0 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("file masih ada: %v", err)
	}
	if res.Batch == "" || !strings.HasPrefix(res.Batch, "auto-") {
		t.Fatalf("batch = %q", res.Batch)
	}
	sessions, err := clean.ReadSessions("", 10)
	if err != nil || len(sessions.Sessions) == 0 {
		t.Fatalf("audit = %v %v", sessions, err)
	}
}

func TestRunCLISchedule(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("CLEANLIE_NO_SYSTEMCTL", "1")

	out := captureStdout(t, func() {
		if _, code := runCLI([]string{"--schedule-install", "--dry-run"}); code != 0 {
			t.Fatalf("dry-run code=%d", code)
		}
	})
	if !strings.Contains(out, "ExecStart=") || !strings.Contains(out, "OnCalendar=weekly") {
		t.Fatalf("unit dry-run:\n%s", out)
	}

	out = captureStdout(t, func() {
		if _, code := runCLI([]string{"--schedule-install", "--interval", "daily", "--json"}); code != 0 {
			t.Fatalf("install code=%d", code)
		}
	})
	var st schedule.Status
	if err := json.Unmarshal([]byte(out), &st); err != nil || !st.Installed || st.Interval != "daily" {
		t.Fatalf("status install: %+v %v\n%s", st, err, out)
	}
	for _, name := range []string{"cleanlie-clean.service", "cleanlie-clean.timer"} {
		if _, err := os.Stat(filepath.Join(cfg, "systemd", "user", name)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	out = captureStdout(t, func() {
		if _, code := runCLI([]string{"--schedule-remove", "--json"}); code != 0 {
			t.Fatalf("remove code=%d", code)
		}
	})
	var rm schedule.Status
	if err := json.Unmarshal([]byte(out), &rm); err != nil || rm.Installed {
		t.Fatalf("status remove: %+v %v\n%s", rm, err, out)
	}
}
