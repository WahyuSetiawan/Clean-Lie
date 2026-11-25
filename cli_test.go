package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
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
