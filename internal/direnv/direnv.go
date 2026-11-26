// Package direnv menemukan folder .direnv milik proyek dan cache global
// direnv/nix-direnv. Semua pemindaian read-only: paket ini tidak pernah
// menghapus atau memindahkan file.
package direnv

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

const (
	// DirName adalah nama folder yang dibuat direnv di tiap proyek.
	DirName = ".direnv"
	// DefaultActiveDays menandai proyek yang baru saja memakai direnv.
	DefaultActiveDays = 7
	// MaxRoots membatasi jumlah folder yang dipindai agar walk tidak liar.
	MaxRoots = 32
)

// Entry adalah satu folder .direnv atau satu cache global.
type Entry struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "project" atau "cache"
	Project string `json:"project"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	Files   int64  `json:"files"`
	ModTime string `json:"modTime"`
	AgeDays int    `json:"ageDays"`
	Active  bool   `json:"active"`
	Note    string `json:"note,omitempty"`
}

// Report adalah hasil satu pemindaian.
type Report struct {
	Roots      []string `json:"roots"`
	Missing    []string `json:"missing,omitempty"`
	Entries    []Entry  `json:"entries"`
	Bytes      int64    `json:"bytes"`
	Files      int64    `json:"files"`
	Errors     int64    `json:"errors"`
	ScannedAt  string   `json:"scannedAt"`
	DurationMs int64    `json:"durationMs"`
}

// Options mengatur pemindaian.
type Options struct {
	Roots      []string
	Excluded   []string
	SkipDirs   map[string]bool
	Now        func() time.Time
	ActiveDays int
	// SkipCache menonaktifkan pengukuran cache global (~/.cache/direnv).
	SkipCache bool
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) activeDays() int {
	if o.ActiveDays <= 0 {
		return DefaultActiveDays
	}
	return o.ActiveDays
}

func (o Options) roots() []string {
	out := make([]string, 0, len(o.Roots))
	seen := map[string]bool{}
	for _, p := range o.Roots {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = expandHome(p)
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		p = filepath.Clean(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) >= MaxRoots {
			break
		}
	}
	return out
}

func (o Options) excluded() []string {
	out := make([]string, 0, len(o.Excluded))
	for _, p := range o.Excluded {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = expandHome(p)
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		out = append(out, filepath.Clean(p))
	}
	return out
}

func (o Options) isExcluded(path string) bool {
	path = filepath.Clean(path)
	for _, root := range o.excluded() {
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// skipDir melewati folder artefak besar supaya walk tetap cepat. .direnv
// ditangani terpisah (dicatat lalu tidak dimasuki).
func (o Options) skipDir(name string) bool {
	if o.SkipDirs != nil {
		return o.SkipDirs[name]
	}
	return defaultSkipDirs[name]
}

var defaultSkipDirs = map[string]bool{
	".git":          true,
	".hg":           true,
	".svn":          true,
	"node_modules":  true,
	"target":        true,
	"vendor":        true,
	".venv":         true,
	"__pycache__":   true,
	".gradle":       true,
	".dart_tool":    true,
	".next":         true,
	".terraform":    true,
	".tox":          true,
	".mypy_cache":   true,
	".pytest_cache": true,
	".cache":        true,
	".local":        true,
	".npm":          true,
	".cargo":        true,
	".rustup":       true,
	".pub-cache":    true,
	".stack-work":   true,
}

// CacheRoots mengembalikan cache global yang dibuat direnv/nix-direnv.
func CacheRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".cache", "direnv"),
		filepath.Join(home, ".cache", "nix-direnv"),
	}
}

// Scan memindai setiap root dan mengukur folder .direnv yang ditemukan.
// Root yang tidak ada dicatat di Report.Missing, bukan dianggap error.
func Scan(ctx context.Context, opts Options) (Report, error) {
	started := opts.now()
	out := Report{ScannedAt: started.Format(time.RFC3339)}
	seenEntries := map[string]bool{}

	add := func(e Entry) {
		if seenEntries[e.Path] {
			return
		}
		seenEntries[e.Path] = true
		out.Entries = append(out.Entries, e)
	}

	for _, root := range opts.roots() {
		out.Roots = append(out.Roots, root)
		fi, err := os.Lstat(root)
		if err != nil || !fi.IsDir() {
			out.Missing = append(out.Missing, root)
			continue
		}
		if opts.isExcluded(root) {
			continue
		}
		// Root yang ditambahkan tepat pada folder .direnv.
		if filepath.Base(root) == DirName {
			if e, err := measure(ctx, root, "project", started, opts); err != nil {
				out.Errors++
			} else {
				add(e)
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				out.Errors++
				if d != nil && d.IsDir() && path != root {
					return fs.SkipDir
				}
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !d.IsDir() || path == root {
				return nil
			}
			if opts.isExcluded(path) {
				return fs.SkipDir
			}
			if d.Name() == DirName {
				if e, err := measure(ctx, path, "project", started, opts); err != nil {
					out.Errors++
				} else {
					add(e)
				}
				return fs.SkipDir
			}
			if opts.skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			return out, err
		}
	}

	if !opts.SkipCache {
		for _, root := range CacheRoots() {
			if !fsx.IsDir(root) || opts.isExcluded(root) {
				continue
			}
			if e, err := measure(ctx, root, "cache", started, opts); err != nil {
				out.Errors++
			} else {
				add(e)
			}
		}
	}

	sort.Slice(out.Entries, func(i, j int) bool {
		if out.Entries[i].Bytes != out.Entries[j].Bytes {
			return out.Entries[i].Bytes > out.Entries[j].Bytes
		}
		return out.Entries[i].Path < out.Entries[j].Path
	})
	for _, e := range out.Entries {
		out.Bytes += e.Bytes
		out.Files += e.Files
	}
	out.DurationMs = time.Since(started).Milliseconds()
	return out, nil
}

func measure(ctx context.Context, path, kind string, now time.Time, opts Options) (Entry, error) {
	bytes, files, err := fsx.Usage(ctx, path)
	if err != nil {
		return Entry{}, err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return Entry{}, err
	}
	age := ageDays(now, fi.ModTime())
	return Entry{
		ID:      "direnv:" + path,
		Kind:    kind,
		Project: filepath.Dir(path),
		Path:    path,
		Bytes:   bytes,
		Files:   files,
		ModTime: fi.ModTime().Format("2006-01-02 15:04"),
		AgeDays: age,
		Active:  age <= opts.activeDays(),
		Note:    noteFor(kind, path),
	}, nil
}

func noteFor(kind, path string) string {
	if kind == "project" {
		return "dibuat ulang oleh direnv/nix-direnv setelah direnv allow"
	}
	switch filepath.Base(path) {
	case "nix-direnv":
		return "profil nix-direnv; isi utamanya symlink ke /nix/store"
	default:
		return "layout cache direnv; dibuat ulang saat reload"
	}
}

func ageDays(now, t time.Time) int {
	if t.IsZero() {
		return 0
	}
	d := int(now.Sub(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
