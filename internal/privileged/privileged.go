// Package privileged menjalankan perintah yang butuh root lewat prompt admin
// platform: pkexec (polkit) di Linux, osascript di macOS. Bila prompt tidak
// tersedia atau dibatalkan, hasilnya berisi perintah sudo siap-copy.
//
// Tidak pernah gagal senyap: hasil selalu membawa exit code, stderr, preview
// perintah, dan (bila perlu) perintah sudo untuk disalin user.
package privileged

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// Runner adalah fungsi eksekusi yang bisa diganti di test.
type Runner func(ctx context.Context, cmd []string, timeout time.Duration) tools.Result

// Options mengatur eksekusi.
type Options struct {
	Runner   Runner
	HasAgent bool
	IsRoot   bool
	LookPath func(string) (string, error)
	Timeout  time.Duration
}

// Result adalah hasil satu eksekusi privileged.
type Result struct {
	Command     []string `json:"command"`
	Preview     string   `json:"preview"`
	SudoCommand string   `json:"sudoCommand,omitempty"`
	NeedsSudo   bool     `json:"needsSudo"`
	Used        string   `json:"used"` // root | pkexec | sudo
	ExitCode    int      `json:"exitCode"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	Error       string   `json:"error,omitempty"`
	DurationMs  int64    `json:"durationMs"`
}

// Execute menjalankan cmd sebagai root. Bila agent polkit tidak ada atau
// pkexec gagal dengan 126/127, hasil berisi NeedsSudo + SudoCommand dan
// perintah tidak dijalankan.
func Execute(ctx context.Context, cmd []string, opts Options) Result {
	started := time.Now()
	res := Result{Command: cmd, ExitCode: -1}
	if len(cmd) == 0 {
		res.Error = "perintah kosong"
		return res
	}
	if opts.Runner == nil {
		opts.Runner = tools.Run
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	if os.Geteuid() == 0 {
		opts.IsRoot = true
	}

	abs, err := opts.LookPath(cmd[0])
	if err != nil {
		res.Error = "binary tidak ditemukan: " + cmd[0]
		res.SudoCommand = SudoCommand(cmd)
		if !opts.IsRoot {
			res.NeedsSudo = true
			res.Used = "sudo"
			res.Preview = res.SudoCommand
		}
		return res
	}
	absCmd := append([]string{abs}, cmd[1:]...)

	if opts.IsRoot {
		res.Used = "root"
		res.Preview = tools.Preview(absCmd)
		return run(ctx, opts, absCmd, res, started)
	}

	return elevate(ctx, opts, absCmd, res, started)
}

// sudoFallback menandai hasil supaya UI menampilkan perintah sudo siap-copy.
func sudoFallback(absCmd []string, res Result, msg string) Result {
	res.NeedsSudo = true
	res.Used = "sudo"
	res.SudoCommand = SudoCommand(absCmd)
	res.Preview = res.SudoCommand
	res.Error = msg
	return res
}

func run(ctx context.Context, opts Options, cmd []string, res Result, started time.Time) Result {
	out := opts.Runner(ctx, cmd, opts.Timeout)
	res.ExitCode = out.ExitCode
	res.Stdout = out.Stdout
	res.Stderr = out.Stderr
	res.DurationMs = time.Since(started).Milliseconds()
	if out.Err != nil && res.Error == "" {
		res.Error = out.Err.Error()
	}
	if res.ExitCode != 0 && res.Error == "" {
		res.Error = "perintah gagal (exit " + strconv.Itoa(res.ExitCode) + ")"
	}
	return res
}

// SudoCommand mengembalikan perintah sudo siap-copy.
func SudoCommand(cmd []string) string {
	return "sudo " + tools.Preview(cmd)
}

// DetectAgent melaporkan apakah aksi root bisa memunculkan prompt admin:
// agent polkit di Linux, osascript di macOS.
func DetectAgent() bool {
	return detectAgent()
}

// Preview mengembalikan preview perintah admin sesuai platform.
func Preview(cmd []string, hasAgent bool) string {
	if len(cmd) == 0 {
		return ""
	}
	abs := cmd
	if p, err := exec.LookPath(cmd[0]); err == nil {
		abs = append([]string{p}, cmd[1:]...)
	}
	return adminPreview(abs, hasAgent)
}

func hasAgentFrom(procRoot string) bool {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(procRoot + "/" + e.Name() + "/cmdline")
		if err != nil || len(b) == 0 {
			continue
		}
		line := strings.ToLower(strings.ReplaceAll(string(b), "\x00", " "))
		if strings.Contains(line, "hyprpolkitagent") {
			return true
		}
		if strings.Contains(line, "polkit") && strings.Contains(line, "agent") {
			return true
		}
	}
	return false
}

// ErrNoAgent bisa dipakai pemanggil yang butuh error bertipe.
var ErrNoAgent = errors.New("prompt admin tidak tersedia")
