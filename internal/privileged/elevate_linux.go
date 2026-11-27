//go:build linux

package privileged

import (
	"context"
	"os/exec"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// elevate menjalankan perintah sebagai root lewat pkexec. Bila agent polkit
// tidak ada atau pkexec menolak (126/127), hasilnya dialihkan ke sudo siap-copy.
func elevate(ctx context.Context, opts Options, absCmd []string, res Result, started time.Time) Result {
	if !opts.HasAgent {
		return sudoFallback(absCmd, res, "polkit agent tidak terdeteksi; jalankan perintah sudo di bawah")
	}

	pkexec, err := opts.LookPath("pkexec")
	if err != nil {
		return sudoFallback(absCmd, res, "pkexec tidak tersedia; jalankan perintah sudo di bawah")
	}

	runCmd := append([]string{pkexec}, absCmd...)
	res.Used = "pkexec"
	res.Preview = tools.Preview(runCmd)
	res = run(ctx, opts, runCmd, res, started)
	if res.ExitCode == 126 || res.ExitCode == 127 {
		res = sudoFallback(absCmd, res, "pkexec gagal (agent tidak merespons); jalankan perintah sudo di bawah")
	}
	return res
}

// adminPreview menampilkan pkexec bila agent tersedia, selain itu sudo.
func adminPreview(absCmd []string, hasAgent bool) string {
	if hasAgent {
		if p, err := exec.LookPath("pkexec"); err == nil {
			return tools.Preview(append([]string{p}, absCmd...))
		}
	}
	return SudoCommand(absCmd)
}

// detectAgent melaporkan apakah ada proses polkit agent yang hidup.
func detectAgent() bool {
	return hasAgentFrom("/proc")
}
