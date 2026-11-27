package trash

import (
	"context"
	"sync"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

// SizeProgress adalah progres pengukuran satu entry.
type SizeProgress struct {
	Done  int
	Total int
	ID    string
	Name  string
	Bytes int64
	Err   string
}

// SizeOptions mengatur SizeEntries. Workers default 4.
type SizeOptions struct {
	Workers  int
	Progress func(SizeProgress)
}

// SizeEntries menghitung ukuran entry secara streaming. Satu set inode
// dipakai bersama agar hard link lintas entry tidak dihitung dua kali.
// Entry yang gagal diukur tetap Bytes=-1 dan alasannya ditambahkan ke Reason.
func SizeEntries(ctx context.Context, entries []Entry, opts SizeOptions) []Entry {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	out := append([]Entry(nil), entries...)
	if len(out) == 0 {
		return out
	}

	seen := &sync.Map{}
	jobs := make(chan int)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	done := 0

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if ctx.Err() != nil {
					continue
				}
				e := out[idx]
				b, _, err := fsx.UsageSeen(ctx, e.FilesPath, seen)
				if err != nil {
					b = -1
					if e.Reason == "" {
						e.Reason = "gagal mengukur: " + err.Error()
					} else {
						e.Reason += "; gagal mengukur: " + err.Error()
					}
				}
				out[idx].Bytes = b
				out[idx].Reason = e.Reason

				progressMu.Lock()
				done++
				p := SizeProgress{
					Done: done, Total: len(out),
					ID: e.ID, Name: e.Name, Bytes: b,
				}
				if err != nil {
					p.Err = err.Error()
				}
				progressMu.Unlock()
				if opts.Progress != nil {
					opts.Progress(p)
				}
			}
		}()
	}

	for i := range out {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	return out
}
