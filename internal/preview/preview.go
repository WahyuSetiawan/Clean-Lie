// Package preview membaca isi satu path secara terbatas untuk modal
// "Lihat isi". Tidak pernah mengikuti symlink dan berhenti setelah 1,5 detik.
package preview

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

// Entry adalah satu anak direktori.
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"` // -1 = gagal diukur
	Dir     bool   `json:"dir"`
	Symlink bool   `json:"symlink"`
	ModTime string `json:"modTime,omitempty"`
}

// Result adalah isi ringkas satu path.
type Result struct {
	Path       string  `json:"path"`
	Kind       string  `json:"kind"` // dir | file | symlink | missing
	Bytes      int64   `json:"bytes"`
	Files      int64   `json:"files"`
	ModTime    string  `json:"modTime,omitempty"`
	Entries    []Entry `json:"entries,omitempty"`
	Truncated  bool    `json:"truncated"`
	TimedOut   bool    `json:"timedOut"`
	Blocked    string  `json:"blocked,omitempty"`
	Error      string  `json:"error,omitempty"`
	DurationMs int64   `json:"durationMs"`
}

const (
	defaultLimit = 20
	maxChildren  = 1000
	scanTimeout  = 1500 * time.Millisecond
)

// Preview mengembalikan metadata dan maksimal limit anak direktori.
func Preview(ctx context.Context, path string, limit int) Result {
	started := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}
	res := Result{Path: path, Kind: "missing"}
	fi, err := os.Lstat(path)
	if err != nil {
		res.Error = err.Error()
		res.DurationMs = time.Since(started).Milliseconds()
		return res
	}
	res.ModTime = fi.ModTime().Format(time.RFC3339)
	if fi.Mode()&os.ModeSymlink != 0 {
		res.Kind = "symlink"
		res.Bytes = fi.Size()
		res.Files = 1
		res.DurationMs = time.Since(started).Milliseconds()
		return res
	}
	if !fi.IsDir() {
		res.Kind = "file"
		if b, n, err := fsx.Usage(ctx, path); err == nil {
			res.Bytes, res.Files = b, n
		} else {
			res.Error = err.Error()
		}
		res.DurationMs = time.Since(started).Milliseconds()
		return res
	}

	res.Kind = "dir"
	scanCtx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()
	f, err := os.Open(path)
	if err != nil {
		res.Error = err.Error()
		res.DurationMs = time.Since(started).Milliseconds()
		return res
	}
	names, err := f.Readdirnames(maxChildren + 1)
	f.Close()
	if err != nil && len(names) == 0 {
		res.Error = err.Error()
		res.DurationMs = time.Since(started).Milliseconds()
		return res
	}
	if len(names) > maxChildren {
		names = names[:maxChildren]
		res.Truncated = true
	}

	for _, name := range names {
		if scanCtx.Err() != nil {
			res.TimedOut = true
			break
		}
		child := filepath.Join(path, name)
		cfi, err := os.Lstat(child)
		if err != nil {
			continue
		}
		ce := Entry{
			Name: name, Path: child,
			Dir: cfi.IsDir(), Symlink: cfi.Mode()&os.ModeSymlink != 0,
			ModTime: cfi.ModTime().Format(time.RFC3339),
		}
		b, n, err := fsx.Usage(scanCtx, child)
		if err != nil {
			ce.Bytes = -1
			if scanCtx.Err() != nil {
				res.TimedOut = true
			}
		} else {
			ce.Bytes = b
			res.Bytes += b
			res.Files += n
		}
		res.Entries = append(res.Entries, ce)
	}
	if scanCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
	}
	sort.SliceStable(res.Entries, func(i, j int) bool {
		if res.Entries[i].Bytes != res.Entries[j].Bytes {
			return res.Entries[i].Bytes > res.Entries[j].Bytes
		}
		return res.Entries[i].Name < res.Entries[j].Name
	})
	if len(res.Entries) > limit {
		res.Entries = res.Entries[:limit]
		res.Truncated = true
	}
	res.DurationMs = time.Since(started).Milliseconds()
	return res
}
