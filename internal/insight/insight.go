// Package insight menyimpan snapshot disk harian, menghitung ringkasan
// mingguan dari audit, dan merender laporan yang bisa diekspor.
package insight

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

const (
	maxPoints      = 90
	defaultDays    = 30
	weekDays       = 7
	recentSessions = 10
)

// Point adalah satu snapshot pemakaian disk per hari.
type Point struct {
	Date  string `json:"date"`
	At    string `json:"at,omitempty"`
	Total int64  `json:"total"`
	Free  int64  `json:"free"`
	Used  int64  `json:"used"`
}

// Week adalah ringkasan audit tujuh hari terakhir.
type Week struct {
	Days            int    `json:"days"`
	Since           string `json:"since"`
	Sessions        int    `json:"sessions"`
	Removed         int    `json:"removed"`
	Failed          int    `json:"failed"`
	MovedBytes      int64  `json:"movedBytes"`
	FreedBytes      int64  `json:"freedBytes"`
	Restorable      int    `json:"restorable"`
	RestorableBytes int64  `json:"restorableBytes"`
	TrashEntries    int    `json:"trashEntries"`
	TrashBytes      int64  `json:"trashBytes"`
}

// Report adalah data yang ditampilkan dashboard dan diekspor.
type Report struct {
	Version     string          `json:"version"`
	GeneratedAt string          `json:"generatedAt"`
	Disk        fsx.Volume      `json:"disk"`
	Week        Week            `json:"week"`
	Trend       []Point         `json:"trend"`
	Sessions    []clean.Session `json:"sessions"`
	Trash       trash.Report    `json:"trash"`
}

// DefaultPath mengembalikan lokasi trend.jsonl di state dir Cleanlie.
func DefaultPath() string {
	return filepath.Join(filepath.Dir(clean.DefaultAuditPath()), "trend.jsonl")
}

// DefaultReportsDir mengembalikan folder laporan.
func DefaultReportsDir() string {
	return filepath.Join(filepath.Dir(clean.DefaultAuditPath()), "reports")
}

// Record mencatat snapshot hari ini. Satu tanggal hanya punya satu titik.
func Record(path string, vol fsx.Volume, now time.Time) error {
	if path == "" {
		path = DefaultPath()
	}
	points, err := readAll(path)
	if err != nil {
		return err
	}
	today := now.Format("2006-01-02")
	out := points[:0]
	for _, p := range points {
		if p.Date != today {
			out = append(out, p)
		}
	}
	out = append(out, Point{
		Date: today, At: now.Format(time.RFC3339),
		Total: vol.Total, Free: vol.Free, Used: vol.Used,
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	if len(out) > maxPoints {
		out = out[len(out)-maxPoints:]
	}
	return writeAll(path, out)
}

// Read mengembalikan titik unik terbaru, urut tanggal naik.
func Read(path string, days int) ([]Point, error) {
	points, err := readAll(path)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]Point, len(points))
	for _, p := range points {
		if p.Date != "" {
			byDate[p.Date] = p
		}
	}
	out := make([]Point, 0, len(byDate))
	for _, p := range byDate {
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	if days > 0 && len(out) > days {
		out = out[len(out)-days:]
	}
	return out, nil
}

func readAll(path string) ([]Point, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Point
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 16*1024), 1<<20)
	for s.Scan() {
		var p Point
		if err := json.Unmarshal(s.Bytes(), &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func writeAll(path string, points []Point) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".trend-*.jsonl")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	enc := json.NewEncoder(tmp)
	for _, p := range points {
		if err := enc.Encode(p); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Collect membaca disk, audit, trash, dan tren; snapshot hari ini dicatat
// bila belum ada. Kegagalan mencatat tren tidak menggagalkan laporan.
func Collect(ctx context.Context, version string, now time.Time) (Report, error) {
	vol, err := fsx.Statfs("/")
	if err != nil {
		return Report{}, err
	}
	page, err := clean.ReadSessions("", 50)
	if err != nil {
		return Report{}, err
	}
	tr, err := trash.List(ctx, trash.ListOptions{MaxEntries: 500})
	if err != nil {
		tr = trash.Report{}
	}
	trend, err := Read(DefaultPath(), defaultDays)
	if err != nil {
		trend = nil
	}
	_ = Record(DefaultPath(), vol, now)
	if len(trend) == 0 || trend[len(trend)-1].Date != now.Format("2006-01-02") {
		if p, err := Read(DefaultPath(), defaultDays); err == nil {
			trend = p
		}
	}
	return Build(version, now, vol, page.Sessions, tr, trend), nil
}

// Build menyusun Report dari data yang sudah dibaca.
func Build(version string, now time.Time, disk fsx.Volume, sessions []clean.Session, tr trash.Report, trend []Point) Report {
	since := now.AddDate(0, 0, -weekDays)
	week := Week{Days: weekDays, Since: since.Format(time.RFC3339), TrashEntries: len(tr.Entries), TrashBytes: tr.Bytes}
	for _, s := range sessions {
		t, err := time.Parse(time.RFC3339, s.StartedAt)
		if err != nil || t.Before(since) || t.After(now.Add(time.Hour)) {
			continue
		}
		week.Sessions++
		week.Removed += s.Removed
		week.Failed += s.Failed
		week.MovedBytes += s.RemovedBytes
		week.FreedBytes += s.Freed
		week.Restorable += s.Restorable
		week.RestorableBytes += s.RestorableBytes
	}
	recent := sessions
	if len(recent) > recentSessions {
		recent = recent[:recentSessions]
	}
	return Report{
		Version: version, GeneratedAt: now.Format(time.RFC3339),
		Disk: disk, Week: week, Trend: trend, Sessions: recent, Trash: tr,
	}
}

// Render menghasilkan laporan md, json, atau csv.
func Render(r Report, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "md", "markdown":
		return renderMarkdown(r), nil
	case "json":
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	case "csv":
		return renderCSV(r)
	default:
		return "", fmt.Errorf("format laporan tidak dikenal: %s (pakai md, json, atau csv)", format)
	}
}

func renderMarkdown(r Report) string {
	var b strings.Builder
	b.WriteString("# Cleanlie — laporan disk\n\n")
	fmt.Fprintf(&b, "Dibuat: %s · versi %s\n\n", r.GeneratedAt, r.Version)
	fmt.Fprintf(&b, "**Disk /**: %s terpakai dari %s · sisa %s\n\n", fsx.Human(r.Disk.Used), fsx.Human(r.Disk.Total), fsx.Human(r.Disk.Free))
	b.WriteString("## Minggu ini\n\n")
	fmt.Fprintf(&b, "- Dibebaskan permanen: **%s**\n", fsx.Human(r.Week.FreedBytes))
	fmt.Fprintf(&b, "- Dipindah ke trash: **%s** (%d item)\n", fsx.Human(r.Week.MovedBytes), r.Week.Removed)
	fmt.Fprintf(&b, "- Sesi: %d · gagal: %d · masih bisa di-undo: %d (%s)\n", r.Week.Sessions, r.Week.Failed, r.Week.Restorable, fsx.Human(r.Week.RestorableBytes))
	fmt.Fprintf(&b, "- Trash saat ini: %d entry · %s\n\n", r.Week.TrashEntries, fsx.Human(r.Week.TrashBytes))

	b.WriteString("## Tren disk\n\n| Tanggal | Sisa | Terpakai |\n|---|---:|---:|\n")
	for _, p := range r.Trend {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", p.Date, fsx.Human(p.Free), fsx.Human(p.Used))
	}
	if len(r.Trend) == 0 {
		b.WriteString("| — | belum ada data | — |\n")
	}
	b.WriteString("\n## Sesi terakhir\n\n| Waktu | Aksi | Item | Dipindah | Dibebaskan |\n|---|---|---:|---:|---:|\n")
	for _, s := range r.Sessions {
		action := s.Action
		if action == "" {
			action = string(s.Mode)
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %s | %s |\n", s.StartedAt, mdEscape(action), s.Removed, fsx.Human(s.RemovedBytes), fsx.Human(s.Freed))
	}
	if len(r.Sessions) == 0 {
		b.WriteString("| — | belum ada sesi | — | — | — |\n")
	}
	b.WriteString("\n## Trash\n\n")
	if len(r.Trash.Entries) == 0 {
		b.WriteString("Trash kosong.\n")
	} else {
		b.WriteString("| Nama | Asal | Dihapus | Status |\n|---|---|---|---|\n")
		for _, e := range r.Trash.Entries {
			state := "bisa dipulihkan"
			if !e.Restorable {
				state = "tidak: " + e.Reason
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", mdEscape(e.Name), mdEscape(e.OriginalPath), e.DeletedAt, mdEscape(state))
		}
	}
	return b.String()
}

func renderCSV(r Report) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"section", "date", "label", "bytes", "count"})
	date := r.GeneratedAt
	_ = w.Write([]string{"summary", date, "disk_used", strconv.FormatInt(r.Disk.Used, 10), ""})
	_ = w.Write([]string{"summary", date, "disk_free", strconv.FormatInt(r.Disk.Free, 10), ""})
	_ = w.Write([]string{"week", date, "freed", strconv.FormatInt(r.Week.FreedBytes, 10), ""})
	_ = w.Write([]string{"week", date, "moved", strconv.FormatInt(r.Week.MovedBytes, 10), strconv.Itoa(r.Week.Removed)})
	_ = w.Write([]string{"week", date, "sessions", "", strconv.Itoa(r.Week.Sessions)})
	_ = w.Write([]string{"week", date, "trash_entries", strconv.FormatInt(r.Week.TrashBytes, 10), strconv.Itoa(r.Week.TrashEntries)})
	for _, p := range r.Trend {
		_ = w.Write([]string{"trend", p.Date, "free", strconv.FormatInt(p.Free, 10), ""})
		_ = w.Write([]string{"trend", p.Date, "used", strconv.FormatInt(p.Used, 10), ""})
	}
	for _, s := range r.Sessions {
		action := s.Action
		if action == "" {
			action = string(s.Mode)
		}
		_ = w.Write([]string{"session", s.StartedAt, action, strconv.FormatInt(s.Freed, 10), strconv.Itoa(s.Removed)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func mdEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
}
