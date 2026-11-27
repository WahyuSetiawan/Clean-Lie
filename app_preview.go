package main

import (
	"errors"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/preview"
)

// PreviewCleanup menampilkan isi item hasil scan katalog. UI hanya mengirim ID.
func (a *App) PreviewCleanup(id string) (preview.Result, error) {
	if id == "" {
		return preview.Result{}, errors.New("ID item kosong")
	}
	a.scanMu.Lock()
	it, ok := a.scanItems[id]
	a.scanMu.Unlock()
	if !ok {
		return preview.Result{}, errors.New("item tidak dikenal; jalankan scan ulang")
	}
	res := preview.Preview(a.baseCtx(), it.Path, 20)
	res.Blocked = clean.BlockedReason(it.Path)
	return res, nil
}

// PreviewTrash menampilkan isi file di trash (bukan path asal yang mungkin
// sudah tidak ada).
func (a *App) PreviewTrash(id string) (preview.Result, error) {
	a.cleanMu.Lock()
	entries, err := a.resolveTrashIDs([]string{id})
	a.cleanMu.Unlock()
	if err != nil {
		return preview.Result{}, err
	}
	e := entries[0]
	res := preview.Preview(a.baseCtx(), e.FilesPath, 20)
	if e.OriginalPath != "" {
		res.Blocked = clean.BlockedReason(e.OriginalPath)
	}
	return res, nil
}
