// Package analysis membaca disk untuk kebutuhan layar Analisis dan Dashboard.
// Semua fungsi di sini read-only: tidak ada penghapusan, pemindahan, atau
// perubahan metadata.
package analysis

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

const (
	DefaultDepth        = 2
	DefaultLargeBytes   = 100 << 20 // 100 MiB
	DefaultStaleDays    = 365
	DefaultTopN         = 100
	DefaultMinDupBytes  = 1 << 20 // 1 MiB
	DefaultArtifactDays = 7
)

// Options mengatur satu pemindaian analisis.
type Options struct {
	Root              string
	Depth             int
	LargeBytes        int64
	StaleDays         int
	TopN              int
	MinDuplicateBytes int64
	Now               func() time.Time
	// SkipDirs berisi nama direktori yang tidak dimasuki (default: .git).
	SkipDirs map[string]bool
	// Excluded adalah path absolut yang tidak dipindai (dari Pengaturan).
	Excluded []string
	// ExtraRoots adalah root tambahan untuk scan artefak di luar Root.
	ExtraRoots []string
}

func (o Options) root() string {
	root := o.Root
	if root == "" {
		home, _ := os.UserHomeDir()
		root = home
	}
	if root == "~" || len(root) > 1 && root[0] == '~' && root[1] == '/' {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, root[1:])
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return filepath.Clean(root)
}

func (o Options) depth() int {
	if o.Depth <= 0 {
		return DefaultDepth
	}
	if o.Depth > 12 {
		return 12
	}
	return o.Depth
}

func (o Options) largeBytes() int64 {
	if o.LargeBytes <= 0 {
		return DefaultLargeBytes
	}
	return o.LargeBytes
}

func (o Options) staleDays() int {
	if o.StaleDays <= 0 {
		return DefaultStaleDays
	}
	return o.StaleDays
}

func (o Options) excluded() []string {
	out := make([]string, 0, len(o.Excluded))
	for _, p := range o.Excluded {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "~" || strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				if p == "~" {
					p = home
				} else {
					p = filepath.Join(home, p[2:])
				}
			}
		}
		if abs, err := filepath.Abs(p); err == nil {
			out = append(out, filepath.Clean(abs))
		}
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

func (o Options) extraRoots() []string {
	out := make([]string, 0, len(o.ExtraRoots))
	for _, p := range o.ExtraRoots {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "~" || strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				if p == "~" {
					p = home
				} else {
					p = filepath.Join(home, p[2:])
				}
			}
		}
		if abs, err := filepath.Abs(p); err == nil {
			out = append(out, filepath.Clean(abs))
		}
	}
	return out
}

func (o Options) topN() int {
	if o.TopN <= 0 {
		return DefaultTopN
	}
	if o.TopN > 1000 {
		return 1000
	}
	return o.TopN
}

func (o Options) minDupBytes() int64 {
	if o.MinDuplicateBytes <= 0 {
		return DefaultMinDupBytes
	}
	return o.MinDuplicateBytes
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) skipDir(name string) bool {
	if o.SkipDirs != nil {
		return o.SkipDirs[name]
	}
	return name == ".git"
}

func (o Options) skipSet() map[string]bool {
	if o.SkipDirs != nil {
		return o.SkipDirs
	}
	return map[string]bool{".git": true}
}

// File adalah satu file pada hasil analisis.
type File struct {
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	ModTime string `json:"modTime"`
	AgeDays int    `json:"ageDays"`
}

// Node adalah satu folder pada tree disk usage.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Files    int64  `json:"files"`
	Dir      bool   `json:"dir"`
	Children []Node `json:"children,omitempty"`
}

// Tree adalah hasil pemindaian disk usage.
type Tree struct {
	Root       Node   `json:"root"`
	Depth      int    `json:"depth"`
	Bytes      int64  `json:"bytes"`
	Files      int64  `json:"files"`
	Dirs       int64  `json:"dirs"`
	Errors     int64  `json:"errors"`
	ScannedAt  string `json:"scannedAt"`
	DurationMs int64  `json:"durationMs"`
}

// Overview adalah hasil satu walk untuk file besar, file lama, dan artefak.
type Overview struct {
	Root       string     `json:"root"`
	Large      []File     `json:"large"`
	Stale      []File     `json:"stale"`
	Artifacts  []Artifact `json:"artifacts"`
	Files      int64      `json:"files"`
	Dirs       int64      `json:"dirs"`
	Errors     int64      `json:"errors"`
	ScannedAt  string     `json:"scannedAt"`
	DurationMs int64      `json:"durationMs"`
}

// Artifact adalah folder hasil build yang bisa dibuat ulang.
type Artifact struct {
	Project   string `json:"project"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	ModTime   string `json:"modTime"`
	AgeDays   int    `json:"ageDays"`
	Active    bool   `json:"active"`
	Protected bool   `json:"protected"`
}

// DuplicateFile adalah satu anggota grup duplikat.
type DuplicateFile struct {
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	ModTime string `json:"modTime"`
	Keep    bool   `json:"keep"`
}

// DuplicateGroup adalah sekelompok file dengan isi identik.
type DuplicateGroup struct {
	Size   int64           `json:"size"`
	Hash   string          `json:"hash"`
	Files  []DuplicateFile `json:"files"`
	Wasted int64           `json:"wasted"`
}

// Duplicates adalah hasil pencarian duplikat.
type Duplicates struct {
	Root          string           `json:"root"`
	Groups        []DuplicateGroup `json:"groups"`
	GroupCount    int              `json:"groupCount"`
	Wasted        int64            `json:"wasted"`
	Hardlinks     int              `json:"hardlinks"`
	HardlinkBytes int64            `json:"hardlinkBytes"`
	Candidates    int              `json:"candidates"`
	Hashed        int              `json:"hashed"`
	Errors        int64            `json:"errors"`
	ScannedAt     string           `json:"scannedAt"`
	DurationMs    int64            `json:"durationMs"`
}

func allocated(fi os.FileInfo) int64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int64(st.Blocks) * 512
	}
	return fi.Size()
}

func fileInfo(path string, now time.Time) (File, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return File{}, err
	}
	if fi.IsDir() {
		return File{}, os.ErrInvalid
	}
	return File{
		Path:    path,
		Bytes:   allocated(fi),
		ModTime: fi.ModTime().Format("2006-01-02 15:04"),
		AgeDays: ageDays(now, fi.ModTime()),
	}, nil
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

func pushTop(list *[]File, f File, n int, older bool) {
	*list = append(*list, f)
	if len(*list) <= n*2 {
		return
	}
	sort.Slice(*list, func(i, j int) bool {
		if older {
			return (*list)[i].ModTime < (*list)[j].ModTime
		}
		return (*list)[i].Bytes > (*list)[j].Bytes
	})
	*list = (*list)[:n]
}

func sortFiles(list []File, older bool) {
	sort.Slice(list, func(i, j int) bool {
		if older {
			return list[i].ModTime < list[j].ModTime
		}
		return list[i].Bytes > list[j].Bytes
	})
}

func usage(ctx context.Context, path string) (int64, int64, error) {
	return fsx.Usage(ctx, path)
}
