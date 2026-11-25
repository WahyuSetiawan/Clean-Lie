package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const duplicatePrefix = 64 << 10

type dupCandidate struct {
	path string
	size int64
	mod  time.Time
}

// Duplicates mencari file identik: group by size, hash 64 KB pertama, lalu
// SHA-256 penuh. File yang ukurannya unik tidak pernah dibaca isinya.
func FindDuplicates(ctx context.Context, opts Options) (Duplicates, error) {
	started := opts.now()
	root := opts.root()
	if _, err := os.Lstat(root); err != nil {
		return Duplicates{}, err
	}
	out := Duplicates{Root: root, ScannedAt: started.Format(time.RFC3339)}
	minSize := opts.minDupBytes()
	bySize := map[int64][]dupCandidate{}
	seen := map[[2]uint64]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if d.IsDir() {
			if path != root && opts.skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			out.Errors++
			return nil
		}
		if info.Size() < minSize {
			return nil
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			key := [2]uint64{uint64(st.Dev), st.Ino}
			if seen[key] {
				return nil
			}
			seen[key] = true
		}
		bySize[info.Size()] = append(bySize[info.Size()], dupCandidate{
			path: path,
			size: info.Size(),
			mod:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return out, err
	}

	for size, list := range bySize {
		if len(list) < 2 {
			continue
		}
		out.Candidates += len(list)
		byPrefix := map[string][]dupCandidate{}
		for _, c := range list {
			h, err := hashPrefix(ctx, c.path)
			if err != nil {
				out.Errors++
				continue
			}
			out.Hashed++
			byPrefix[h] = append(byPrefix[h], c)
		}
		for _, samePrefix := range byPrefix {
			if len(samePrefix) < 2 {
				continue
			}
			byFull := map[string][]dupCandidate{}
			for _, c := range samePrefix {
				h, err := hashFull(ctx, c.path)
				if err != nil {
					out.Errors++
					continue
				}
				out.Hashed++
				byFull[h] = append(byFull[h], c)
			}
			for h, files := range byFull {
				if len(files) < 2 {
					continue
				}
				sort.Slice(files, func(i, j int) bool {
					if files[i].mod.Equal(files[j].mod) {
						return files[i].path < files[j].path
					}
					return files[i].mod.Before(files[j].mod)
				})
				g := DuplicateGroup{Size: size, Hash: h}
				for i, c := range files {
					g.Files = append(g.Files, DuplicateFile{
						Path:    c.path,
						Bytes:   c.size,
						ModTime: c.mod.Format("2006-01-02 15:04"),
						Keep:    i == 0,
					})
				}
				g.Wasted = size * int64(len(files)-1)
				out.Wasted += g.Wasted
				out.Groups = append(out.Groups, g)
			}
		}
	}
	out.GroupCount = len(out.Groups)
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].Wasted != out.Groups[j].Wasted {
			return out.Groups[i].Wasted > out.Groups[j].Wasted
		}
		return out.Groups[i].Hash < out.Groups[j].Hash
	})
	if len(out.Groups) > opts.topN() {
		out.Groups = out.Groups[:opts.topN()]
	}
	out.DurationMs = time.Since(started).Milliseconds()
	return out, nil
}

func hashPrefix(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.CopyN(h, f, duplicatePrefix); err != nil && err != io.EOF {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFull(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 256<<10)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		if n > 0 {
			if _, werr := h.Write(buf[:n]); werr != nil {
				return "", werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("baca %s: %w", path, err)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
