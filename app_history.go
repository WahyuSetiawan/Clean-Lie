package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

type trashSizeUpdate struct {
	ID    string `json:"id"`
	Bytes int64  `json:"bytes"`
	Err   string `json:"err,omitempty"`
}

type trashSizeEvent struct {
	Updates []trashSizeUpdate `json:"updates"`
	Done    int               `json:"done"`
	Total   int               `json:"total"`
}

// TrashList membaca metadata trash lalu mengukur ukuran di background.
func (a *App) TrashList() (trash.Report, error) {
	a.cleanMu.Lock()
	rep, gen, entries, err := a.loadTrashReport()
	a.cleanMu.Unlock()
	if err != nil {
		return trash.Report{}, err
	}
	a.startTrashSizes(gen, entries)
	return rep, nil
}

// TrashRefresh membuang cache dan membatalkan pengukuran yang berjalan.
func (a *App) TrashRefresh() bool {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()
	a.trashMu.Lock()
	a.trashGen++
	if a.trashSizeCancel != nil {
		a.trashSizeCancel()
		a.trashSizeCancel = nil
	}
	a.trashReport = nil
	a.trashEntries = nil
	a.trashAt = time.Time{}
	a.trashMu.Unlock()
	return true
}

func (a *App) loadTrashReport() (trash.Report, int, []trash.Entry, error) {
	rep, err := trash.List(a.baseCtx(), trash.ListOptions{})
	if err != nil {
		return trash.Report{}, 0, nil, err
	}
	entries := make(map[string]trash.Entry, len(rep.Entries))
	for _, e := range rep.Entries {
		entries[e.ID] = e
	}
	a.trashMu.Lock()
	a.trashGen++
	gen := a.trashGen
	if a.trashSizeCancel != nil {
		a.trashSizeCancel()
		a.trashSizeCancel = nil
	}
	a.trashReport = &rep
	a.trashEntries = entries
	a.trashAt = time.Now()
	a.trashMu.Unlock()
	return rep, gen, rep.Entries, nil
}

func (a *App) startTrashSizes(gen int, entries []trash.Entry) {
	if len(entries) == 0 {
		return
	}
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.trashMu.Lock()
	if a.trashSizeCancel != nil {
		a.trashSizeCancel()
	}
	a.trashSizeCancel = cancel
	a.trashMu.Unlock()

	go func() {
		defer cancel()
		var mu sync.Mutex
		pending := make([]trashSizeUpdate, 0, 32)
		done, total := 0, len(entries)
		last := time.Now()

		take := func(force bool) (trashSizeEvent, bool) {
			mu.Lock()
			defer mu.Unlock()
			if len(pending) == 0 {
				return trashSizeEvent{}, false
			}
			if !force && time.Since(last) < 100*time.Millisecond {
				return trashSizeEvent{}, false
			}
			batch := append([]trashSizeUpdate(nil), pending...)
			pending = pending[:0]
			last = time.Now()
			return trashSizeEvent{Updates: batch, Done: done, Total: total}, true
		}

		sized := trash.SizeEntries(ctx, entries, trash.SizeOptions{
			Workers: 4,
			Progress: func(p trash.SizeProgress) {
				mu.Lock()
				pending = append(pending, trashSizeUpdate{ID: p.ID, Bytes: p.Bytes, Err: p.Err})
				done, total = p.Done, p.Total
				mu.Unlock()
				if ev, ok := take(false); ok {
					a.emit("trash:size", ev)
				}
			},
		})
		if ev, ok := take(true); ok {
			a.emit("trash:size", ev)
		}
		if ctx.Err() != nil {
			return
		}

		a.trashMu.Lock()
		if gen != a.trashGen {
			a.trashMu.Unlock()
			return
		}
		a.trashReport = &trash.Report{
			Roots:      nil,
			Entries:    sized,
			Bytes:      sumTrashBytes(sized),
			ScannedAt:  time.Now().Format(time.RFC3339),
			DurationMs: 0,
		}
		a.trashEntries = make(map[string]trash.Entry, len(sized))
		for _, e := range sized {
			a.trashEntries[e.ID] = e
		}
		a.trashAt = time.Now()
		a.trashSizeCancel = nil
		a.trashMu.Unlock()
		a.emit("trash:done", map[string]any{"count": len(sized), "bytes": sumTrashBytes(sized)})
	}()
}

func sumTrashBytes(entries []trash.Entry) int64 {
	var n int64
	for _, e := range entries {
		if e.Bytes > 0 {
			n += e.Bytes
		}
	}
	return n
}

func (a *App) resolveTrashIDs(ids []string) ([]trash.Entry, error) {
	if len(ids) == 0 {
		return nil, errors.New("tidak ada item trash yang dipilih")
	}
	unique := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, errors.New("tidak ada ID trash yang valid")
	}

	a.trashMu.Lock()
	cache := a.trashEntries
	a.trashMu.Unlock()

	missing := false
	for _, id := range unique {
		if _, ok := cache[id]; !ok {
			missing = true
			break
		}
	}
	if missing {
		if _, _, _, err := a.loadTrashReport(); err != nil {
			return nil, err
		}
		a.trashMu.Lock()
		cache = a.trashEntries
		a.trashMu.Unlock()
	}

	out := make([]trash.Entry, 0, len(unique))
	for _, id := range unique {
		e, ok := cache[id]
		if !ok {
			return nil, errors.New("item trash tidak dikenal: " + id)
		}
		out = append(out, e)
	}
	return out, nil
}

// TrashRestore mengembalikan item ke path asal.
func (a *App) TrashRestore(ids []string, dryRun bool) (trash.ActionResult, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()

	entries, err := a.resolveTrashIDs(ids)
	if err != nil {
		return trash.ActionResult{}, err
	}
	started := time.Now().Format(time.RFC3339)
	res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(entries))}

	if dryRun {
		for _, e := range entries {
			ir := trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: maxBytes(e.Bytes)}
			if !e.Restorable {
				ir.Status, ir.Reason = "skipped", e.Reason
				res.Skipped++
			} else {
				res.Restored++
				res.Bytes += maxBytes(e.Bytes)
			}
			res.Items = append(res.Items, ir)
		}
		return res, nil
	}

	if err := clean.RotateAudit(""); err != nil {
		return trash.ActionResult{}, err
	}
	batch := clean.NewBatch("restore-")
	var audits []clean.Entry
	for i, e := range entries {
		a.emit("trash:progress", map[string]any{"done": i, "total": len(entries), "name": e.Name, "status": "restoring"})
		ir := trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath}
		if !e.Restorable {
			ir.Status, ir.Reason = "skipped", e.Reason
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		err := trash.Restore(a.baseCtx(), e)
		if err != nil {
			ir.Status, ir.Reason = "failed", err.Error()
			res.Failed++
		} else {
			ir.Status, ir.Bytes = "restored", maxBytes(e.Bytes)
			res.Restored++
			res.Bytes += ir.Bytes
		}
		res.Items = append(res.Items, ir)
		status := clean.StatusRemoved
		if err != nil {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{
			Time: started, Mode: clean.ModeTrash, ID: e.ID, Path: e.OriginalPath,
			BytesBefore: ir.Bytes, BytesAfter: 0, Freed: 0, Status: status,
			Reason: ir.Reason, TrashPath: e.FilesPath, Batch: batch, Action: "restore",
		})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	a.invalidateAfterMutation()
	a.emit("trash:changed", map[string]any{"reason": "restore", "count": res.Restored})
	return res, nil
}

// TrashDelete menghapus permanen item trash yang dipilih.
func (a *App) TrashDelete(ids []string, confirm, dryRun bool) (trash.ActionResult, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()

	if !dryRun && !confirm {
		return trash.ActionResult{}, errors.New("hapus permanen dari trash butuh konfirmasi")
	}
	entries, err := a.resolveTrashIDs(ids)
	if err != nil {
		return trash.ActionResult{}, err
	}
	if dryRun {
		res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(entries))}
		for _, e := range entries {
			b := maxBytes(e.Bytes)
			res.Deleted++
			res.Bytes += b
			res.Items = append(res.Items, trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: b})
		}
		return res, nil
	}

	if err := clean.RotateAudit(""); err != nil {
		return trash.ActionResult{}, err
	}
	batch := clean.NewBatch("trash-")
	res := trash.Delete(a.baseCtx(), entries)
	started := time.Now().Format(time.RFC3339)
	audits := make([]clean.Entry, 0, len(res.Items))
	for _, ir := range res.Items {
		if ir.Status == "skipped" {
			continue
		}
		status := clean.StatusRemoved
		if ir.Status != "deleted" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{
			Time: started, Mode: clean.ModePermanent, ID: ir.ID, Path: ir.OriginalPath,
			BytesBefore: ir.Bytes, Freed: ir.Bytes, Status: status,
			Reason: ir.Reason, Batch: batch, Action: "trash-delete",
		})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	a.invalidateAfterMutation()
	a.emit("trash:changed", map[string]any{"reason": "delete", "count": res.Deleted})
	return res, nil
}

// TrashEmpty mengosongkan seluruh isi trash.
func (a *App) TrashEmpty(confirm string, dryRun bool) (trash.ActionResult, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()

	confirm = strings.ToUpper(strings.TrimSpace(confirm))
	if !dryRun && confirm != "KOSONGKAN" && confirm != "EMPTY" {
		return trash.ActionResult{}, errors.New("ketik KOSONGKAN atau EMPTY untuk mengosongkan trash")
	}
	roots := trash.DiscoverRoots()
	if dryRun {
		rep, err := trash.List(a.baseCtx(), trash.ListOptions{Roots: roots})
		if err != nil {
			return trash.ActionResult{}, err
		}
		res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(rep.Entries))}
		for _, e := range rep.Entries {
			b := maxBytes(e.Bytes)
			res.Deleted++
			res.Bytes += b
			res.Items = append(res.Items, trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: b})
		}
		return res, nil
	}

	if err := clean.RotateAudit(""); err != nil {
		return trash.ActionResult{}, err
	}
	res, err := trash.Empty(a.baseCtx(), roots, confirm)
	if err != nil {
		return trash.ActionResult{}, err
	}
	started := time.Now().Format(time.RFC3339)
	batch := clean.NewBatch("trash-")
	audits := make([]clean.Entry, 0, len(res.Items))
	for _, ir := range res.Items {
		if ir.Status == "skipped" {
			continue
		}
		status := clean.StatusRemoved
		if ir.Status != "deleted" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{
			Time: started, Mode: clean.ModePermanent, ID: ir.ID, Path: ir.OriginalPath,
			BytesBefore: ir.Bytes, Freed: ir.Bytes, Status: status,
			Reason: ir.Reason, Batch: batch, Action: "trash-empty",
		})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	a.invalidateAfterMutation()
	a.emit("trash:changed", map[string]any{"reason": "empty", "count": res.Deleted})
	return res, nil
}

// HistorySessions mengembalikan sesi audit untuk layar Riwayat.
func (a *App) HistorySessions(limit int) (clean.SessionPage, error) {
	return clean.ReadSessions("", limit)
}

// UndoSession mengembalikan semua item yang masih bisa di-restore dari satu sesi.
func (a *App) UndoSession(batch string, dryRun bool) (trash.ActionResult, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()

	if strings.TrimSpace(batch) == "" {
		return trash.ActionResult{}, errors.New("batch sesi kosong")
	}
	items, ok, err := clean.ReadSessionEntries("", batch)
	if err != nil {
		return trash.ActionResult{}, err
	}
	if !ok {
		return trash.ActionResult{}, errors.New("sesi tidak ditemukan")
	}

	res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(items))}
	if dryRun {
		for _, e := range items {
			ir := trash.ItemResult{ID: e.ID, Name: e.ID, OriginalPath: e.Path}
			if !clean.RestorableEntry(e) {
				ir.Status, ir.Reason = "skipped", "item tidak bisa di-undo (permanen/perintah)"
				res.Skipped++
			} else {
				ir.Status = "planned"
				res.Restored++
			}
			res.Items = append(res.Items, ir)
		}
		return res, nil
	}

	if err := clean.RotateAudit(""); err != nil {
		return trash.ActionResult{}, err
	}
	undoBatch := clean.NewBatch("undo-")
	started := time.Now().Format(time.RFC3339)
	var audits []clean.Entry
	for i, e := range items {
		ir := trash.ItemResult{ID: e.ID, Name: e.ID, OriginalPath: e.Path}
		if !clean.RestorableEntry(e) {
			ir.Status, ir.Reason = "skipped", "item tidak bisa di-undo (permanen/perintah)"
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		if e.Path == "" {
			ir.Status, ir.Reason = "skipped", "path audit kosong"
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		tr, found := trash.Resolve(a.baseCtx(), e.Path)
		if !found {
			ir.Status, ir.Reason = "skipped", "jejak trash tidak ditemukan"
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		if !tr.Restorable {
			ir.Status, ir.Reason = "skipped", tr.Reason
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		a.emit("trash:progress", map[string]any{"done": i, "total": len(items), "name": e.Path, "status": "undo"})
		err := trash.Restore(a.baseCtx(), tr)
		if err != nil {
			ir.Status, ir.Reason = "failed", err.Error()
			res.Failed++
		} else {
			ir.Status, ir.Bytes = "restored", maxBytes(e.BytesBefore)
			res.Restored++
			res.Bytes += ir.Bytes
		}
		res.Items = append(res.Items, ir)
		status := clean.StatusRemoved
		if err != nil {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{
			Time: started, Mode: clean.ModeTrash, ID: e.ID, Path: e.Path,
			BytesBefore: ir.Bytes, BytesAfter: 0, Freed: 0, Status: status,
			Reason: ir.Reason, TrashPath: tr.FilesPath, Batch: undoBatch, Action: "restore",
		})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	a.invalidateAfterMutation()
	a.emit("trash:changed", map[string]any{"reason": "undo", "count": res.Restored})
	return res, nil
}

func maxBytes(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}

// invalidateAfterMutation menyegarkan semua cache yang mungkin basi setelah
// clean/restore/delete. Tidak memegang cleanMu.
func (a *App) invalidateAfterMutation() {
	a.scanMu.Lock()
	a.scanItems = nil
	a.scanMu.Unlock()

	a.AnalysisRefresh()
	a.DirenvRefresh()

	a.trashMu.Lock()
	a.trashGen++
	if a.trashSizeCancel != nil {
		a.trashSizeCancel()
		a.trashSizeCancel = nil
	}
	a.trashReport = nil
	a.trashEntries = nil
	a.trashAt = time.Time{}
	a.trashMu.Unlock()
}
