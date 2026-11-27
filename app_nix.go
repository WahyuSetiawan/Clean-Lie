package main

import (
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/privileged"
)

// nixCacheTTL membatasi berapa lama status Nix dipakai ulang. Dead paths
// butuh ~30 detik, jadi jangan dihitung ulang setiap layar dibuka.
const nixCacheTTL = 2 * time.Minute

// NixStatus mengembalikan store, generasi, dan journal. Hasil di-cache.
func (a *App) NixStatus() (nix.Status, error) {
	a.nixMu.Lock()
	defer a.nixMu.Unlock()
	if a.nixStatus != nil && time.Since(a.nixStatusAt) < nixCacheTTL {
		return *a.nixStatus, nil
	}
	st := nix.ReadStatus(a.baseCtx(), nix.Options{})
	agent := privileged.DetectAgent()
	st.Agent = agent
	st.GCCommand = nix.GCPreview(agent)
	a.nixStatus = &st
	a.nixStatusAt = time.Now()
	return st, nil
}

// NixDeadPaths menjalankan dry-run GC (bisa puluhan detik) dan di-cache.
func (a *App) NixDeadPaths() (nix.Dead, error) {
	a.nixDeadMu.Lock()
	defer a.nixDeadMu.Unlock()
	if a.nixDead != nil && time.Since(a.nixDeadAt) < nixCacheTTL {
		return *a.nixDead, nil
	}
	d := nix.DeadPaths(a.baseCtx(), nix.Options{})
	a.nixDead = &d
	a.nixDeadAt = time.Now()
	return d, nil
}

// NixRefresh membuang cache status/dead; pemanggilan berikutnya memindai ulang.
func (a *App) NixRefresh() bool {
	a.invalidateNix()
	return true
}

func (a *App) invalidateNix() {
	a.nixMu.Lock()
	a.nixStatus = nil
	a.nixMu.Unlock()
	a.nixDeadMu.Lock()
	a.nixDead = nil
	a.nixDeadMu.Unlock()
}

// NixDeleteGenerations menghapus generasi non-current. Confirm harus "GC".
func (a *App) NixDeleteGenerations(req nix.GenRequest) (nix.ActionResult, error) {
	res, err := nix.DeleteGenerations(a.baseCtx(), req, nix.Options{})
	if err != nil {
		return res, err
	}
	if res.ExitCode == 0 && !res.NeedsSudo {
		a.invalidateNix()
		a.notifyNixResult("Cleanlie — hapus generasi selesai", res)
	}
	return res, nil
}

// NixCollectGarbage menjalankan GC penuh (root). Confirm harus "GC".
func (a *App) NixCollectGarbage(confirm string) (nix.ActionResult, error) {
	res, err := nix.CollectGarbage(a.baseCtx(), confirm, nix.Options{})
	if err != nil {
		return res, err
	}
	if res.ExitCode == 0 && !res.NeedsSudo {
		a.invalidateNix()
		a.notifyNixResult("Cleanlie — GC selesai", res)
	}
	return res, nil
}

// VacuumJournal mengecilkan journal ke target (mis. "500M") lewat root.
func (a *App) VacuumJournal(target string) (nix.ActionResult, error) {
	res, err := nix.VacuumJournal(a.baseCtx(), target, nix.Options{})
	if err != nil {
		return res, err
	}
	if res.ExitCode == 0 && !res.NeedsSudo {
		a.invalidateNix()
		a.notifyNixResult("Cleanlie — vacuum journal selesai", res)
	}
	return res, nil
}

// NixPreview mengembalikan perintah yang akan dijalankan untuk modal
// konfirmasi. Tidak mengeksekusi apa pun.
func (a *App) NixPreview(kind, profile string, numbers []int) string {
	agent := privileged.DetectAgent()
	switch kind {
	case "gc":
		return nix.GCPreview(agent)
	case "delete":
		return nix.DeletePreview(profile, numbers, "", agent)
	case "journal":
		return nix.JournalPreview(profile, agent)
	}
	return ""
}
