// Package fsx menyediakan scanner ukuran disk paralel.
//
// Ukuran yang dilaporkan adalah blok disk yang benar-benar terpakai
// (st_blocks * 512, seperti `du`), bukan ukuran logis file.
package fsx

import (
	"container/heap"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
)

const blockSize = 512

// File adalah satu file besar yang ditemukan selama walk.
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Progress adalah snapshot penghitung saat walk berjalan.
type Progress struct {
	Files   int64
	Dirs    int64
	Errors  int64
	Current string
}

// Options mengatur perilaku Walk.
type Options struct {
	// Workers adalah jumlah goroutine pemindai. 0 = DefaultWorkers().
	Workers int
	// OneFS mencegah walk menyeberang ke filesystem lain.
	OneFS bool
	// Skip berisi path absolut yang tidak boleh dimasuki.
	Skip map[string]bool
	// TopN > 0 mengumpulkan N file terbesar.
	TopN int
	// Progress dipanggil setiap selesai satu direktori (boleh nil).
	Progress func(Progress)
}

// Result adalah hasil satu walk.
type Result struct {
	Size   int64  `json:"size"`
	Files  int64  `json:"files"`
	Dirs   int64  `json:"dirs"`
	Errors int64  `json:"errors"`
	Top    []File `json:"top,omitempty"`
}

// DefaultWorkers mengembalikan jumlah worker default (CPU x 4, 4..64).
func DefaultWorkers() int {
	n := runtime.NumCPU() * 4
	if n < 4 {
		n = 4
	}
	if n > 64 {
		n = 64
	}
	return n
}

type scanner struct {
	ctx     context.Context
	opts    Options
	rootDev uint64
	sem     chan struct{}
	inodes  sync.Map

	size  atomic.Int64
	files atomic.Int64
	dirs  atomic.Int64
	errs  atomic.Int64

	hmu  sync.Mutex
	heap fileHeap
}

// Walk menghitung ukuran terpakai di bawah root.
func Walk(ctx context.Context, root string, opts Options) (Result, error) {
	if opts.Workers <= 0 {
		opts.Workers = DefaultWorkers()
	}
	root = filepath.Clean(root)

	var st syscall.Stat_t
	if err := syscall.Lstat(root, &st); err != nil {
		return Result{}, err
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		size := int64(st.Blocks) * blockSize
		return Result{Size: size, Files: 1}, nil
	}

	s := &scanner{
		ctx:     ctx,
		opts:    opts,
		rootDev: uint64(st.Dev),
		sem:     make(chan struct{}, opts.Workers),
	}
	s.scan(root)

	res := Result{
		Size:   s.size.Load(),
		Files:  s.files.Load(),
		Dirs:   s.dirs.Load(),
		Errors: s.errs.Load(),
	}
	if len(s.heap) > 0 {
		res.Top = make([]File, len(s.heap))
		copy(res.Top, s.heap)
		sort.Slice(res.Top, func(i, j int) bool { return res.Top[i].Size > res.Top[j].Size })
	}
	return res, ctx.Err()
}

func (s *scanner) scan(dir string) {
	if s.ctx.Err() != nil {
		return
	}
	s.dirs.Add(1)

	f, err := os.Open(dir)
	if err != nil {
		s.errs.Add(1)
		return
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil && len(names) == 0 {
		s.errs.Add(1)
		return
	}

	var wg sync.WaitGroup
	for _, name := range names {
		if s.ctx.Err() != nil {
			break
		}
		p := filepath.Join(dir, name)
		if s.opts.Skip[p] {
			continue
		}

		var st syscall.Stat_t
		if err := syscall.Lstat(p, &st); err != nil {
			s.errs.Add(1)
			continue
		}
		size := int64(st.Blocks) * blockSize

		if st.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			if s.opts.OneFS && uint64(st.Dev) != s.rootDev {
				continue
			}
			// blok direktori ikut dihitung, supaya total = du.
			s.size.Add(size)

			work := func() { s.scan(p) }
			select {
			case s.sem <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					work()
					<-s.sem
				}()
			default:
				work()
			}
			continue
		}

		// hard link hanya dihitung sekali.
		if st.Mode&syscall.S_IFMT == syscall.S_IFREG && st.Nlink > 1 {
			key := [2]uint64{uint64(st.Dev), st.Ino}
			if _, dup := s.inodes.LoadOrStore(key, struct{}{}); dup {
				continue
			}
		}

		s.size.Add(size)
		s.files.Add(1)
		if st.Mode&syscall.S_IFMT == syscall.S_IFREG && s.opts.TopN > 0 {
			s.addFile(p, size)
		}
	}
	wg.Wait()

	if s.opts.Progress != nil {
		s.opts.Progress(Progress{
			Files:   s.files.Load(),
			Dirs:    s.dirs.Load(),
			Errors:  s.errs.Load(),
			Current: dir,
		})
	}
}

func (s *scanner) addFile(path string, size int64) {
	s.hmu.Lock()
	if len(s.heap) < s.opts.TopN {
		heap.Push(&s.heap, File{Path: path, Size: size})
	} else if size > s.heap[0].Size {
		s.heap[0] = File{Path: path, Size: size}
		heap.Fix(&s.heap, 0)
	}
	s.hmu.Unlock()
}

type fileHeap []File

func (h fileHeap) Len() int           { return len(h) }
func (h fileHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h fileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *fileHeap) Push(x any)        { *h = append(*h, x.(File)) }
func (h *fileHeap) Pop() any          { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }
