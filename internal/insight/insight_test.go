package insight

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

func statePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	t.Setenv("HOME", t.TempDir())
	return filepath.Join(dir, "cleanlie", "trend.jsonl")
}

func TestRecordOnePointPerDay(t *testing.T) {
	path := statePath(t)
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	if err := Record(path, fsx.Volume{Total: 100, Free: 40, Used: 60}, now); err != nil {
		t.Fatal(err)
	}
	if err := Record(path, fsx.Volume{Total: 100, Free: 55, Used: 45}, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	points, err := Read(path, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].Free != 55 {
		t.Fatalf("points = %+v", points)
	}
	if points[0].Date != "2026-11-27" {
		t.Fatalf("date = %q", points[0].Date)
	}
}

func TestReadDedupeAndLimit(t *testing.T) {
	path := statePath(t)
	raw := `{"date":"2026-11-01","free":1}
{"date":"2026-11-02","free":2}
{"date":"2026-11-02","free":9}
{"date":"2026-11-03","free":3}
rusak
`
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	points, err := Read(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Date != "2026-11-02" || points[0].Free != 9 {
		t.Fatalf("points = %+v", points)
	}
}

func TestBuildWeek(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	sessions := []clean.Session{
		{StartedAt: "2026-11-27T10:00:00Z", Removed: 2, RemovedBytes: 300, Freed: 0, Restorable: 2, RestorableBytes: 300},
		{StartedAt: "2026-11-21T10:00:00Z", Removed: 1, RemovedBytes: 100, Freed: 50, Failed: 1},
		{StartedAt: "2026-10-01T10:00:00Z", Removed: 9, RemovedBytes: 900},
	}
	tr := trash.Report{Entries: []trash.Entry{{ID: "a"}, {ID: "b"}}, Bytes: 700}
	rep := Build("0.9.0", now, fsx.Volume{Total: 1000, Free: 400, Used: 600}, sessions, tr, []Point{{Date: "2026-11-27", Free: 400}})
	if rep.Week.Sessions != 2 || rep.Week.Removed != 3 || rep.Week.MovedBytes != 400 {
		t.Fatalf("week = %+v", rep.Week)
	}
	if rep.Week.FreedBytes != 50 || rep.Week.Failed != 1 || rep.Week.TrashEntries != 2 || rep.Week.TrashBytes != 700 {
		t.Fatalf("week = %+v", rep.Week)
	}
	if len(rep.Sessions) != 3 {
		t.Fatalf("sessions = %d", len(rep.Sessions))
	}
}

func TestRenderFormats(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	rep := Build("0.9.0", now, fsx.Volume{Total: 1000, Free: 400, Used: 600},
		[]clean.Session{{StartedAt: "2026-11-27T10:00:00Z", Removed: 1, RemovedBytes: 10}},
		trash.Report{}, []Point{{Date: "2026-11-27", Free: 400, Used: 600}})

	md, err := Render(rep, "md")
	if err != nil || !strings.Contains(md, "Cleanlie") || !strings.Contains(md, "Tren disk") {
		t.Fatalf("md = %q err=%v", md, err)
	}
	js, err := Render(rep, "json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal([]byte(js), &decoded); err != nil {
		t.Fatalf("json: %v", err)
	}
	if decoded.Week.Removed != 1 {
		t.Fatalf("decoded = %+v", decoded.Week)
	}
	csvOut, err := Render(rep, "csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(csvOut)).ReadAll()
	if err != nil || len(rows) < 4 {
		t.Fatalf("csv rows=%d err=%v", len(rows), err)
	}
	if _, err := Render(rep, "xml"); err == nil {
		t.Fatal("format asing harus ditolak")
	}
}

func TestCollectWritesDailyTrend(t *testing.T) {
	path := statePath(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	if _, err := Collect(context.Background(), "0.9.0", now); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(context.Background(), "0.9.0", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	points, err := Read(path, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("trend = %+v", points)
	}
}
