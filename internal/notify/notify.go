// Package notify mengirim notifikasi desktop lewat notify-send. Notifikasi
// bersifat best-effort: kegagalan tidak boleh menggagalkan operasi utama.
package notify

import (
	"context"
	"errors"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// ErrUnavailable berarti notify-send tidak ada di PATH.
var ErrUnavailable = errors.New("notify-send tidak tersedia")

// Available melaporkan apakah notify-send bisa dipakai.
func Available() bool { return tools.Available("notify-send") }

// Send mengirim satu notifikasi. Timeout internal 2 detik.
func Send(ctx context.Context, title, body string) error {
	if !Available() {
		return ErrUnavailable
	}
	res := tools.Run(ctx, []string{
		"notify-send", "--app-name=Cleanlie", "--urgency=low", title, body,
	}, 2*time.Second)
	if res.Err != nil {
		return res.Err
	}
	if res.ExitCode != 0 {
		return errors.New(res.Stderr)
	}
	return nil
}
