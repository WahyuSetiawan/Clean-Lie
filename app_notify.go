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

func (a *App) notifyCleanResult(title string, res clean.Result) {
	removed := 0
	for _, it := range res.Items {
		if it.Status == clean.StatusRemoved {
			removed++
		}
	}
	body := fmt.Sprintf("%d item · %s dibebaskan", removed, fsx.Human(res.Freed))
	if res.Mode == clean.ModeTrash {
		body = fmt.Sprintf("%d item dipindah ke trash", removed)
	}
	a.notifyDesktop(title, body)
}

func (a *App) notifyNixResult(title string, res nix.ActionResult) {
	body := res.Preview
	if res.Deleted > 0 {
		body = fmt.Sprintf("%d item dihapus", res.Deleted)
	} else if res.Error != "" {
		body = "gagal: " + res.Error
	}
	a.notifyDesktop(title, body)
}
