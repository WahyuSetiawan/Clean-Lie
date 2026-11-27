package trash

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Resolve mencari entry trash dari path asal tanpa memindai semua entry.
// Dipakai setelah `gio trash` (yang tidak memberi TrashPath) dan saat undo.
// Home trash dicari lebih dulu, lalu volume dengan mount point terpanjang.
func Resolve(ctx context.Context, originalPath string) (Entry, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	originalPath = filepath.Clean(originalPath)
	if !filepath.IsAbs(originalPath) {
		return Entry{}, false
	}

	roots := DiscoverRoots()
	for _, r := range roots {
		if r.Kind != RootKindHome {
			continue
		}
		if e, ok := findInRoot(ctx, r, originalPath); ok {
			return e, true
		}
	}

	volumes := make([]Root, 0, len(roots))
	for _, r := range roots {
		if r.Kind == RootKindVolume && r.MountPoint != "" {
			volumes = append(volumes, r)
		}
	}
	sort.SliceStable(volumes, func(i, j int) bool {
		return len(volumes[i].MountPoint) > len(volumes[j].MountPoint)
	})
	for _, r := range volumes {
		if !inside(r.MountPoint, originalPath) {
			continue
		}
		if e, ok := findInRoot(ctx, r, originalPath); ok {
			return e, true
		}
	}
	return Entry{}, false
}

func findInRoot(ctx context.Context, root Root, originalPath string) (Entry, bool) {
	if ctx.Err() != nil {
		return Entry{}, false
	}
	rootPath := filepath.Clean(root.Path)
	if root.Kind == RootKindHome {
		if p, err := filepath.EvalSymlinks(rootPath); err == nil {
			rootPath = p
		}
	} else if !isRealDir(rootPath) {
		return Entry{}, false
	}
	infoDir := filepath.Join(rootPath, "info")
	filesDir := filepath.Join(rootPath, "files")
	if !isRealDir(infoDir) || !isRealDir(filesDir) {
		return Entry{}, false
	}

	base := filepath.Base(originalPath)
	if !validName(base) {
		return Entry{}, false
	}
	names, err := os.ReadDir(infoDir)
	if err != nil {
		return Entry{}, false
	}

	var best Entry
	var bestAt time.Time
	for _, n := range names {
		if ctx.Err() != nil {
			break
		}
		if !strings.HasSuffix(n.Name(), ".trashinfo") || !strings.HasPrefix(n.Name(), base) {
			continue
		}
		name := strings.TrimSuffix(n.Name(), ".trashinfo")
		if !validName(name) {
			continue
		}
		infoPath := filepath.Join(infoDir, n.Name())
		if !isRealFile(infoPath) {
			continue
		}
		got, deletedAt, err := parseInfo(infoPath, root)
		if err != nil || filepath.Clean(got) != originalPath {
			continue
		}
		if deletedAt.IsZero() {
			if st, err := os.Stat(infoPath); err == nil {
				deletedAt = st.ModTime()
			}
		}
		if best.ID != "" && !deletedAt.After(bestAt) {
			continue
		}
		filesPath := filepath.Join(filesDir, name)
		_, fileErr := os.Lstat(filesPath)
		restorable, reason := restorableState(got, fileErr == nil, nil)
		best = Entry{
			ID:           EntryID(rootPath, name),
			Name:         name,
			OriginalPath: got,
			DeletedAt:    deletedAt,
			Bytes:        -1,
			FilesPath:    filesPath,
			InfoPath:     infoPath,
			Root:         rootPath,
			MountPoint:   root.MountPoint,
			RootKind:     root.Kind,
			Restorable:   restorable,
			Reason:       reason,
		}
		bestAt = deletedAt
	}
	return best, best.ID != ""
}
