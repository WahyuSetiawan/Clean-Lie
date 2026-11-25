package analysis

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

// Tree memindai disk usage sampai kedalaman tertentu. Direktori pada depth
// terakhir diukur dengan fsx.Usage (paralel) sehingga totalnya tetap allocated
// block dan hardlink dihitung sekali.
func ScanTree(ctx context.Context, opts Options) (Tree, error) {
	started := opts.now()
	root := opts.root()
	fi, err := os.Lstat(root)
	if err != nil {
		return Tree{}, err
	}
	if !fi.IsDir() {
		f, err := fileInfo(root, started)
		if err != nil {
			return Tree{}, err
		}
		return Tree{
			Root:       Node{Name: filepath.Base(root), Path: root, Bytes: f.Bytes, Files: 1},
			Depth:      opts.depth(),
			Bytes:      f.Bytes,
			Files:      1,
			ScannedAt:  started.Format(time.RFC3339),
			DurationMs: time.Since(started).Milliseconds(),
		}, nil
	}

	t := Tree{
		Depth:     opts.depth(),
		ScannedAt: started.Format(time.RFC3339),
	}
	seen := &sync.Map{}
	node, files, dirs, errs := treeDir(ctx, root, 0, opts, seen)
	if err != nil {
		return t, err
	}
	t.Root = node
	t.Bytes = node.Bytes
	t.Files = files
	t.Dirs = dirs + 1
	t.Errors = errs
	t.DurationMs = time.Since(started).Milliseconds()
	return t, nil
}

func treeDir(ctx context.Context, dir string, level int, opts Options, seen *sync.Map) (Node, int64, int64, int64) {
	n := Node{Name: filepath.Base(dir), Path: dir, Dir: true}
	var files, dirs, errs int64
	if fi, err := os.Lstat(dir); err == nil {
		n.Bytes += allocated(fi)
	} else {
		errs++
	}
	if level >= opts.depth() {
		bytes, f, err := fsx.UsageSeenSkip(ctx, dir, seen, opts.skipSet())
		if err != nil {
			errs++
		}
		return Node{Name: n.Name, Path: dir, Dir: true, Bytes: bytes, Files: f}, f, 1, errs
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		errs++
		return n, 0, 1, errs
	}
	for _, e := range entries {
		if ctx.Err() != nil {
			break
		}
		if e.IsDir() && opts.skipDir(e.Name()) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			child, cf, cd, ce := treeDir(ctx, p, level+1, opts, seen)
			n.Children = append(n.Children, child)
			n.Bytes += child.Bytes
			files += cf
			dirs += cd
			errs += ce
			continue
		}
		info, err := e.Info()
		if err != nil {
			errs++
			continue
		}
		if info.Mode().IsRegular() {
			if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Nlink > 1 {
				key := [2]uint64{uint64(st.Dev), st.Ino}
				if _, dup := seen.LoadOrStore(key, struct{}{}); dup {
					continue
				}
			}
		}
		sz := allocated(info)
		n.Bytes += sz
		files++
		n.Children = append(n.Children, Node{
			Name:  e.Name(),
			Path:  p,
			Bytes: sz,
			Files: 1,
		})
	}
	sort.Slice(n.Children, func(i, j int) bool {
		if n.Children[i].Bytes != n.Children[j].Bytes {
			return n.Children[i].Bytes > n.Children[j].Bytes
		}
		return n.Children[i].Name < n.Children[j].Name
	})
	n.Files = files
	return n, files, dirs + 1, errs
}
