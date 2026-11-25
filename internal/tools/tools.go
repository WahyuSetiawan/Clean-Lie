// Package tools menjalankan perintah pembersih resmi milik tool
// (npm cache clean, pnpm store prune, uv cache prune, ...).
package tools

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Result adalah hasil satu eksekusi perintah.
type Result struct {
	Command  []string      `json:"command"`
	Preview  string        `json:"preview"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
	ExitCode int           `json:"exitCode"`
	Duration time.Duration `json:"duration"`
	Err      error         `json:"-"`
}

// Available melaporkan apakah binary ada di PATH.
func Available(name string) bool {
	if name == "" {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// Preview mengembalikan perintah yang aman ditampilkan ke user.
func Preview(cmd []string) string {
	parts := make([]string, len(cmd))
	for i, c := range cmd {
		if c == "" || strings.ContainsAny(c, " \t'\"$`\\") {
			parts[i] = "'" + strings.ReplaceAll(c, "'", `'\''`) + "'"
		} else {
			parts[i] = c
		}
	}
	return strings.Join(parts, " ")
}

// Run menjalankan perintah dengan timeout. Timeout <= 0 berarti hanya
// mengikuti context pemanggil (tetap disarankan memberi batas).
func Run(ctx context.Context, cmd []string, timeout time.Duration) Result {
	return RunEnv(ctx, cmd, timeout, nil)
}

// RunEnv seperti Run, tetapi menambahkan env ke environment proses.
func RunEnv(ctx context.Context, cmd []string, timeout time.Duration, env []string) Result {
	res := Result{Command: cmd, Preview: Preview(cmd)}
	if len(cmd) == 0 {
		res.ExitCode = -1
		res.Err = errors.New("perintah kosong")
		return res
	}
	if !Available(cmd[0]) {
		res.ExitCode = -1
		res.Err = exec.ErrNotFound
		return res
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	start := time.Now()
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	if len(env) > 0 {
		c.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	res.Duration = time.Since(start)
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	if err != nil {
		res.Err = err
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		if ctx.Err() != nil {
			res.Err = ctx.Err()
		}
	}
	return res
}
