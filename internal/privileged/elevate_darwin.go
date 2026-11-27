//go:build darwin

package privileged

import (
	"context"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// elevate menjalankan perintah sebagai root lewat dialog admin native macOS
// (osascript). Ini pengganti pkexec di Linux: user diminta Touch ID/password,
// lalu perintah dijalankan sebagai root.
func elevate(ctx context.Context, opts Options, absCmd []string, res Result, started time.Time) Result {
	if !opts.HasAgent {
		return sudoFallback(absCmd, res, "prompt admin tidak tersedia; jalankan perintah sudo di bawah")
	}

	osascript, err := opts.LookPath("osascript")
	if err != nil {
		return sudoFallback(absCmd, res, "osascript tidak tersedia; jalankan perintah sudo di bawah")
	}

	script := `do shell script "` + escapeAppleScript(tools.Preview(absCmd)) + `" with administrator privileges`
	runCmd := []string{osascript, "-e", script}
	res.Used = "osascript"
	res.Preview = tools.Preview(runCmd)
	res = run(ctx, opts, runCmd, res, started)
	if res.ExitCode != 0 && isUserCanceled(res.Stderr) {
		res = sudoFallback(absCmd, res, "dibatalkan oleh pengguna; jalankan perintah sudo di bawah")
	}
	return res
}

// escapeAppleScript menyiapkan string perintah agar aman di dalam literal
// AppleScript ("..."): backslash dulu, baru tanda kutip.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// isUserCanceled mendeteksi pembatalan dialog admin. osascript keluar dengan
// kode 1 untuk cancel maupun kegagalan perintah, jadi pesannya yang diperiksa.
func isUserCanceled(stderr string) bool {
	low := strings.ToLower(stderr)
	return strings.Contains(low, "user canceled") ||
		strings.Contains(low, "(-128)") ||
		strings.Contains(low, "user cancelled")
}

// adminPreview menampilkan perintah sudo; eksekusinya lewat prompt osascript.
func adminPreview(absCmd []string, _ bool) string {
	return SudoCommand(absCmd)
}

// detectAgent di macOS selalu tersedia selama osascript ada, karena dialog
// admin tidak butuh daemon polkit.
func detectAgent() bool {
	return true
}
