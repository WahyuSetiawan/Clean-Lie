package fsx

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"syscall"
)

// UsageSeen seperti Usage, tetapi memakai satu set inode bersama agar hard
// link yang tersebar di beberapa subtree tidak dihitung dua kali.
func UsageSeen(ctx context.Context, path string, seen *sync.Map) (bytes int64, files int64, err error) {
	return UsageSeenSkip(ctx, path, seen, nil)
}

// UsageSeenSkip seperti UsageSeen, dengan nama entri yang dilewati.
func UsageSeenSkip(ctx context.Context, path string, seen *sync.Map, skipNames map[string]bool) (bytes int64, files int64, err error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, 0, err
	}
	if !fi.IsDir() {
		return allocated(fi), 1, nil
	}
	res, err := Walk(ctx, path, Options{OneFS: true, Inodes: seen, SkipNames: skipNames})
	return res.Size, res.Files, err
}

// Usage menghitung ukuran satu path (file atau direktori).
// Untuk direktori, hard link dihitung sekali dan filesystem lain tidak diikuti.
func Usage(ctx context.Context, path string) (bytes int64, files int64, err error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, 0, err
	}
	if !fi.IsDir() {
		return allocated(fi), 1, nil
	}
	res, err := Walk(ctx, path, Options{OneFS: true})
	return res.Size, res.Files, err
}

func allocated(fi os.FileInfo) int64 {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fi.Size()
	}
	return int64(st.Blocks) * blockSize
}

// Exists melaporkan apakah path ada (tanpa mengikuti symlink).
func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// IsDir melaporkan apakah path adalah direktori (tanpa mengikuti symlink).
func IsDir(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.IsDir()
}

// List mengembalikan nama anak direktori, terurut.
func List(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// Human memformat byte menjadi KiB/MiB/GiB dengan satu desimal.
func Human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Volume adalah ringkasan pemakaian filesystem.
type Volume struct {
	Total int64 `json:"total"`
	Free  int64 `json:"free"`
	Used  int64 `json:"used"`
}

// Statfs membaca pemakaian filesystem tempat path berada.
func Statfs(path string) (Volume, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Volume{}, err
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	return Volume{Total: total, Free: free, Used: total - free}, nil
}
