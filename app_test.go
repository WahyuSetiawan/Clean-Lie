package main

import (
	"runtime"
	"testing"
	"time"
)

func TestDiagnosticsBasics(t *testing.T) {
	a := NewApp("1.2.3-test", "abc1234")
	a.startedAt = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	d := a.Diagnostics()

	if d.Version != "1.2.3-test" {
		t.Errorf("version = %q, want 1.2.3-test", d.Version)
	}
	if d.Commit != "abc1234" {
		t.Errorf("commit = %q, want abc1234", d.Commit)
	}
	if d.GoVersion != runtime.Version() {
		t.Errorf("goVersion = %q, want %q", d.GoVersion, runtime.Version())
	}
	wantPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if d.Platform != wantPlatform {
		t.Errorf("platform = %q, want %q", d.Platform, wantPlatform)
	}
	if d.Home == "" {
		t.Error("home kosong")
	}
	if d.StartedAt != "2026-11-22T12:00:00Z" {
		t.Errorf("startedAt = %q", d.StartedAt)
	}
	if !d.NixStore {
		t.Error("nixStore = false, padahal test dijalankan di NixOS")
	}
}

func TestDetectPolkitAgentShape(t *testing.T) {
	ok, detail := detectPolkitAgent()
	if detail == "" {
		t.Fatal("detail tidak boleh kosong")
	}
	if !ok && detail == "" {
		t.Fatal("jika agent tidak ada, detail harus menjelaskan fallback")
	}
}
