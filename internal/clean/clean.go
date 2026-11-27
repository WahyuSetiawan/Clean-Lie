// Package clean mengeksekusi rencana pembersihan: guard path, trash
// freedesktop, hapus permanen, perintah resmi tool, ukur ulang, audit.
package clean

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// Mode menentukan cara item dihapus.
type Mode string

const (
	ModeTrash     Mode = "trash"
	ModePermanent Mode = "permanent"
)

// Status hasil per item.
type Status string

const (
	StatusRemoved Status = "removed"
	StatusFailed  Status = "failed"
	StatusKept    Status = "kept"
	StatusSkipped Status = "skipped"
	StatusPlanned Status = "planned"
)

// Target adalah satu item yang sudah dipilih di UI.
type Target struct {
	ID           string
	Path         string
	Root         string
	Label        string
	Whole        bool
	Depth        int
	Bytes        int64
	Command      []string
	FallbackOK   bool
	RequiresRoot bool
}

// Request adalah satu permintaan eksekusi.
type Request struct {
	Targets []Target
	Mode    Mode
	DryRun  bool
}

// Options mengatur eksekusi.
type Options struct {
	// Progress dipanggil setelah tiap item selesai.
	Progress func(Progress)
	// Timeout per perintah resmi. 0 = 5 menit.
	Timeout time.Duration
	// AuditPath 0 = ~/.local/state/cleanlie/history.jsonl.
	AuditPath string
	// Batch bisa dipakai test/undo; kosong = batch baru.
	Batch string
	// Now bisa diganti di test.
	Now func() time.Time
}

// Progress adalah progres eksekusi.
type Progress struct {
	Done  int
	Total int
	ID    string
	Path  string
	State Status
}

// ItemResult adalah hasil satu item.
type ItemResult struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Label       string `json:"label"`
	Status      Status `json:"status"`
	BytesBefore int64  `json:"bytesBefore"`
	BytesAfter  int64  `json:"bytesAfter"`
	Freed       int64  `json:"freed"`
	Mode        Mode   `json:"mode"`
	Reason      string `json:"reason,omitempty"`
	CommandLine string `json:"commandLine,omitempty"`
	TrashPath   string `json:"trashPath,omitempty"`
}

// Result adalah ringkasan satu eksekusi.
type Result struct {
	Mode       Mode         `json:"mode"`
	StartedAt  string       `json:"startedAt"`
	DurationMs int64        `json:"durationMs"`
	Before     int64        `json:"before"`
	After      int64        `json:"after"`
	Freed      int64        `json:"freed"`
	Items      []ItemResult `json:"items"`
	AuditPath  string       `json:"auditPath"`
	AuditError string       `json:"auditError,omitempty"`
	Batch      string       `json:"batch,omitempty"`
}

// Execute menjalankan request secara berurutan; item gagal tidak
// menghentikan item lain.
func Execute(ctx context.Context, req Request, opts Options) Result {
	started := time.Now()
	if opts.Now != nil {
		started = opts.Now()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.AuditPath == "" {
		opts.AuditPath = DefaultAuditPath()
	}
	mode := req.Mode
	if mode != ModePermanent {
		mode = ModeTrash
	}

	res := Result{
		Mode:      mode,
		StartedAt: started.Format(time.RFC3339),
		AuditPath: opts.AuditPath,
	}
	res.Batch = opts.Batch
	if res.Batch == "" {
		res.Batch = NewBatch("")
	}
	if !req.DryRun {
		if err := RotateAudit(opts.AuditPath); err != nil {
			res.AuditError = err.Error()
		}
	}
	total := len(req.Targets)

	for i, t := range req.Targets {
		ir := executeOne(ctx, t, mode, req.DryRun, opts)

		res.Before += ir.BytesBefore
		res.After += ir.BytesAfter
		res.Freed += ir.Freed
		res.Items = append(res.Items, ir)

		if !req.DryRun {
			entry := Entry{
				Time:        started.Format(time.RFC3339),
				Mode:        mode,
				ID:          ir.ID,
				Path:        ir.Path,
				BytesBefore: ir.BytesBefore,
				BytesAfter:  ir.BytesAfter,
				Freed:       ir.Freed,
				Status:      ir.Status,
				Reason:      ir.Reason,
				TrashPath:   ir.TrashPath,
				CommandLine: ir.CommandLine,
				Batch:       res.Batch,
			}
			if err := AppendAudit(opts.AuditPath, entry); err != nil && res.AuditError == "" {
				res.AuditError = err.Error()
			}
		}

		if opts.Progress != nil {
			opts.Progress(Progress{Done: i + 1, Total: total, ID: t.ID, Path: t.Path, State: ir.Status})
		}
	}

	res.DurationMs = time.Since(started).Milliseconds()
	return res
}

func executeOne(ctx context.Context, t Target, mode Mode, dryRun bool, opts Options) ItemResult {
	ir := ItemResult{ID: t.ID, Path: t.Path, Label: t.Label, Mode: mode}

	if t.RequiresRoot {
		ir.Status = StatusSkipped
		ir.Reason = "butuh aksi root (vacuum journal ada di M3)"
		return ir
	}
	if err := Validate(t); err != nil {
		ir.Status = StatusFailed
		ir.Reason = err.Error()
		return ir
	}
	if reason := BlockedReason(t.Path); reason != "" {
		ir.Status = StatusSkipped
		ir.Reason = reason
		return ir
	}

	before, _, err := fsx.Usage(ctx, t.Path)
	if err != nil {
		ir.Status = StatusFailed
		ir.Reason = "tidak bisa diukur: " + err.Error()
		return ir
	}
	ir.BytesBefore = before

	if dryRun {
		ir.Status = StatusPlanned
		ir.BytesAfter = before
		return ir
	}

	if err := runAction(ctx, t, mode, &ir, opts); err != nil {
		ir.Status = StatusFailed
		ir.Reason = err.Error()
	} else {
		ir.Status = StatusRemoved
	}

	after := int64(0)
	if fsx.Exists(t.Path) {
		if a, _, err := fsx.Usage(ctx, t.Path); err == nil {
			after = a
		}
	}
	ir.BytesAfter = after
	if before > after {
		ir.Freed = before - after
	}
	if ir.Status == StatusRemoved && ir.Freed == 0 && fsx.Exists(t.Path) {
		ir.Status = StatusKept
		ir.Reason = "perintah selesai tetapi ukuran tidak berubah"
	}
	return ir
}

func runAction(ctx context.Context, t Target, mode Mode, ir *ItemResult, opts Options) error {
	if len(t.Command) > 0 {
		ir.CommandLine = tools.Preview(t.Command)
		out := tools.Run(ctx, t.Command, opts.Timeout)
		if out.Err == nil {
			return nil
		}
		if !t.FallbackOK {
			return fmt.Errorf("perintah gagal (exit %d): %s", out.ExitCode, firstLine(out.Stderr))
		}
		// lanjut ke hapus path sebagai fallback
	}

	switch mode {
	case ModePermanent:
		if err := os.RemoveAll(t.Path); err != nil {
			return fmt.Errorf("hapus permanen: %w", err)
		}
		return nil
	default:
		info, err := MoveToTrash(ctx, t.Path)
		if err != nil {
			if errors.Is(err, ErrCrossDevice) {
				return errors.New("tidak bisa ke trash: beda filesystem (pilih Hapus permanen)")
			}
			return fmt.Errorf("trash: %w", err)
		}
		ir.TrashPath = info.TrashPath
		return nil
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 240 {
		s = s[:240] + "…"
	}
	return s
}

// Validate memastikan target memang anak dari root katalog dan tidak protected.
func Validate(t Target) error {
	if t.Path == "" {
		return errors.New("path kosong")
	}
	if !filepath.IsAbs(t.Path) {
		return errors.New("path harus absolut")
	}
	path := filepath.Clean(t.Path)
	root := filepath.Clean(t.Root)
	if root == "." || root == "" || !filepath.IsAbs(root) {
		return errors.New("root katalog tidak valid")
	}

	if rules.IsProtected(path) {
		return errors.New("path ada di daftar protected")
	}
	if path == root {
		if !t.Whole {
			return errors.New("root katalog bukan item yang bisa dihapus")
		}
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "" || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("item di luar root katalog")
	}
	depth := len(strings.Split(rel, string(filepath.Separator)))
	maxDepth := t.Depth
	if maxDepth <= 0 {
		maxDepth = 1
	}
	if depth > maxDepth {
		return errors.New("item lebih dalam dari aturan katalog")
	}
	return nil
}
