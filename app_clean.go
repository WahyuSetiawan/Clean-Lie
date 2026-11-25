package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

// CleanRequest dikirim UI. Path tidak dikirim dari JS: hanya ID dari hasil
// scan terakhir, supaya UI tidak bisa meminta hapus path sembarangan.
type CleanRequest struct {
	IDs              []string `json:"ids"`
	Mode             string   `json:"mode"`
	ConfirmPermanent bool     `json:"confirmPermanent"`
	DryRun           bool     `json:"dryRun"`
}

func (a *App) emit(name string, data any) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, name, data)
	}
}

func (a *App) baseCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// ScanJunk memindai katalog dan menyimpan hasilnya untuk Clean.
func (a *App) ScanJunk() (rules.Report, error) {
	a.scanMu.Lock()
	if a.scanCancel != nil {
		a.scanMu.Unlock()
		return rules.Report{}, errors.New("scan lain sedang berjalan")
	}
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.scanCancel = cancel
	a.scanMu.Unlock()

	defer func() {
		cancel()
		a.scanMu.Lock()
		a.scanCancel = nil
		a.scanMu.Unlock()
	}()

	var lastEmit time.Time
	report, err := rules.Plan(ctx, rules.DefaultRules(), rules.PlanOptions{
		Progress: func(p rules.Progress) {
			if time.Since(lastEmit) < 120*time.Millisecond {
				return
			}
			lastEmit = time.Now()
			a.emit("scan:progress", map[string]any{
				"done":  p.Done,
				"total": p.Total,
				"path":  p.Path,
			})
		},
	})
	if err != nil {
		return rules.Report{}, err
	}

	items := make(map[string]rules.Item, report.Files)
	for _, g := range report.Groups {
		for _, it := range g.Items {
			items[it.ID] = it
		}
	}
	a.scanMu.Lock()
	a.scanItems = items
	a.scanMu.Unlock()

	a.emit("scan:done", map[string]any{
		"bytes":  report.Bytes,
		"files":  report.Files,
		"groups": len(report.Groups),
	})
	return report, nil
}

// CancelScan membatalkan scan yang sedang berjalan.
func (a *App) CancelScan() bool {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	if a.scanCancel == nil {
		return false
	}
	a.scanCancel()
	return true
}

// Clean mengeksekusi item terpilih dari hasil ScanJunk terakhir.
func (a *App) Clean(req CleanRequest) (clean.Result, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()

	mode := clean.Mode(req.Mode)
	if mode != clean.ModeTrash && mode != clean.ModePermanent {
		return clean.Result{}, errors.New("mode harus trash atau permanent")
	}
	if mode == clean.ModePermanent && !req.ConfirmPermanent && !req.DryRun {
		return clean.Result{}, errors.New("hapus permanen butuh konfirmasi")
	}
	if len(req.IDs) == 0 {
		return clean.Result{}, errors.New("tidak ada item yang dipilih")
	}

	a.scanMu.Lock()
	items := a.scanItems
	a.scanMu.Unlock()
	if len(items) == 0 {
		return clean.Result{}, errors.New("belum ada hasil scan; jalankan scan dulu")
	}

	targets := make([]clean.Target, 0, len(req.IDs))
	seen := map[string]bool{}
	for _, id := range req.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		it, ok := items[id]
		if !ok {
			return clean.Result{}, errors.New("item tidak dikenal: " + id)
		}
		targets = append(targets, clean.Target{
			ID:           it.ID,
			Path:         it.Path,
			Root:         it.Root,
			Label:        it.Label,
			Whole:        it.Whole,
			Depth:        it.Depth,
			Bytes:        it.Bytes,
			Command:      it.Command,
			FallbackOK:   it.FallbackOK,
			RequiresRoot: it.RequiresRoot,
		})
	}

	var lastEmit time.Time
	res := clean.Execute(a.baseCtx(), clean.Request{
		Targets: targets,
		Mode:    mode,
		DryRun:  req.DryRun,
	}, clean.Options{
		Progress: func(p clean.Progress) {
			if time.Since(lastEmit) < 80*time.Millisecond && p.Done != p.Total {
				return
			}
			lastEmit = time.Now()
			a.emit("clean:progress", map[string]any{
				"done":  p.Done,
				"total": p.Total,
				"path":  p.Path,
				"state": p.State,
			})
		},
	})

	// Item yang sudah hilang tidak boleh bisa dieksekusi dua kali.
	if !req.DryRun {
		a.scanMu.Lock()
		for _, it := range res.Items {
			if it.Status == clean.StatusRemoved {
				delete(a.scanItems, it.ID)
			}
		}
		a.scanMu.Unlock()
	}
	return res, nil
}

// History mengembalikan audit log terakhir.
func (a *App) History(limit int) ([]clean.Entry, error) {
	return clean.ReadAudit("", limit)
}

// OpenInFileManager membuka folder (atau induk file) di file manager.
func (a *App) OpenInFileManager(path string) error {
	p := filepath.Clean(rules.ExpandHome(path))
	fi, err := os.Stat(p)
	if err != nil {
		p = filepath.Dir(p)
		if _, err := os.Stat(p); err != nil {
			return err
		}
	} else if !fi.IsDir() {
		p = filepath.Dir(p)
	}
	if !filepath.IsAbs(p) {
		return errors.New("path harus absolut")
	}
	return exec.Command("xdg-open", p).Start()
}

// OpenTrash membuka folder trash freedesktop.
func (a *App) OpenTrash() error {
	dir := clean.TrashDir()
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o700); err != nil {
		return err
	}
	return exec.Command("xdg-open", filepath.Join(dir, "files")).Start()
}

// CopyText menyalin teks (mis. laporan hasil) ke clipboard.
func (a *App) CopyText(text string) error {
	if len(text) > 1<<20 {
		text = text[:1<<20]
	}
	if a.ctx != nil {
		wruntime.ClipboardSetText(a.ctx, text)
	}
	return nil
}
