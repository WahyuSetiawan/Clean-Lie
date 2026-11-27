package main

import (
	"fmt"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/notify"
)

// notifyDesktop mengirim notifikasi bila diaktifkan di Pengaturan. Semua
// kegagalan diabaikan: notifikasi bukan bagian dari transaksi pembersihan.
func (a *App) notifyDesktop(title, body string) {
	if !a.settings().Notify {
		return
	}
	_ = notify.Send(a.baseCtx(), title, body)
}

func (a *App) notifyCleanResult(kind string, res clean.Result) {
	removed := 0
	for _, it := range res.Items {
		if it.Status == clean.StatusRemoved {
			removed++
		}
	}
	title := a.msg("Cleanlie — pembersihan selesai", "Cleanlie — cleanup finished")
	switch kind {
	case "analysis":
		title = a.msg("Cleanlie — pembersihan analisis selesai", "Cleanlie — analysis cleanup finished")
	case "direnv":
		title = a.msg("Cleanlie — pembersihan direnv selesai", "Cleanlie — direnv cleanup finished")
	}
	var body string
	if res.Mode == clean.ModeTrash {
		body = a.msg(
			fmt.Sprintf("%d item dipindah ke trash", removed),
			fmt.Sprintf("%d items moved to trash", removed),
		)
	} else {
		body = a.msg(
			fmt.Sprintf("%d item · %s dibebaskan", removed, fsx.Human(res.Freed)),
			fmt.Sprintf("%d items · %s freed", removed, fsx.Human(res.Freed)),
		)
	}
	a.notifyDesktop(title, body)
}

func (a *App) notifyNixResult(kind string, res nix.ActionResult) {
	title := a.msg("Cleanlie — aksi Nix selesai", "Cleanlie — Nix action finished")
	switch kind {
	case "delete":
		title = a.msg("Cleanlie — hapus generasi selesai", "Cleanlie — generation delete finished")
	case "gc":
		title = a.msg("Cleanlie — GC selesai", "Cleanlie — GC finished")
	case "vacuum":
		title = a.msg("Cleanlie — vacuum journal selesai", "Cleanlie — journal vacuum finished")
	}
	body := res.Preview
	if res.Deleted > 0 {
		body = a.msg(
			fmt.Sprintf("%d item dihapus", res.Deleted),
			fmt.Sprintf("%d items deleted", res.Deleted),
		)
	} else if res.Error != "" {
		body = a.msg("gagal: ", "failed: ") + res.Error
	}
	a.notifyDesktop(title, body)
}
