package analysis

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

var artifactNames = map[string]bool{
	"node_modules": true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".venv":        true,
	"__pycache__":  true,
	".gradle":      true,
	".dart_tool":   true,
	"vendor":       true,
}

// Overview memindai root satu kali untuk file besar, file lama, dan artefak
// build. Folder artefak tidak dimasuki lagi, jadi node_modules di dalam
// node_modules tidak dihitung dua kali.
func ScanOverview(ctx context.Context, opts Options) (Overview, error) {
	started := opts.now()
	root := opts.root()
	if _, err := os.Lstat(root); err != nil {
		return Overview{}, err
	}
	out := Overview{Root: root, ScannedAt: started.Format(time.RFC3339)}
	cutoff := started.AddDate(0, 0, -opts.staleDays())
	topN := opts.topN()
	largeMin := opts.largeBytes()

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
			if path == root {
				return nil
			}
			if opts.skipDir(d.Name()) {
				return fs.SkipDir
			}
			if artifactNames[d.Name()] {
				if a := readArtifact(ctx, path, started, &out); a != nil {
					out.Artifacts = append(out.Artifacts, *a)
				}
				return fs.SkipDir
			}
			out.Dirs++
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
		out.Files++
		sz := allocated(info)
		if sz >= largeMin {
			pushTop(&out.Large, File{
				Path:    path,
				Bytes:   sz,
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
				AgeDays: ageDays(started, info.ModTime()),
			}, topN, false)
		}
		if info.ModTime().Before(cutoff) && !rules.IsProtected(path) {
			pushTop(&out.Stale, File{
				Path:    path,
				Bytes:   sz,
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
				AgeDays: ageDays(started, info.ModTime()),
			}, topN, true)
		}
		return nil
	})
	if err != nil {
		return out, err
	}

	sortFiles(out.Large, false)
	sortFiles(out.Stale, true)
	if len(out.Large) > topN {
		out.Large = out.Large[:topN]
	}
	if len(out.Stale) > topN {
		out.Stale = out.Stale[:topN]
	}
	sort.Slice(out.Artifacts, func(i, j int) bool {
		if out.Artifacts[i].Bytes != out.Artifacts[j].Bytes {
			return out.Artifacts[i].Bytes > out.Artifacts[j].Bytes
		}
		return out.Artifacts[i].Path < out.Artifacts[j].Path
	})
	if len(out.Artifacts) > topN {
		out.Artifacts = out.Artifacts[:topN]
	}
	out.DurationMs = time.Since(started).Milliseconds()
	return out, nil
}

func readArtifact(ctx context.Context, path string, now time.Time, out *Overview) *Artifact {
	bytes, _, err := usage(ctx, path)
	if err != nil {
		out.Errors++
		return nil
	}
	fi, err := os.Lstat(path)
	if err != nil {
		out.Errors++
		return nil
	}
	project := filepath.Dir(path)
	projectMod := fi.ModTime()
	if pfi, err := os.Stat(project); err == nil {
		projectMod = pfi.ModTime()
	}
	return &Artifact{
		Protected: rules.IsProtected(path),
		Project:   project,
		Name:      filepath.Base(path),
		Path:      path,
		Bytes:     bytes,
		ModTime:   fi.ModTime().Format("2006-01-02 15:04"),
		AgeDays:   ageDays(now, fi.ModTime()),
		Active:    now.Sub(projectMod) <= DefaultArtifactDays*24*time.Hour,
	}
}
