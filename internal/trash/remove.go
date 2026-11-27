package trash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

var errForeignMount = errors.New("ada mount point di dalam entry; dilewati")

// Delete menghapus permanen entry yang diberikan. Entry harus berasal dari
// List/Resolve; path divalidasi ulang terhadap root.
func Delete(ctx context.Context, entries []Entry) ActionResult {
	if ctx == nil {
		ctx = context.Background()
	}
	res := ActionResult{}
	for _, e := range entries {
		if ctx.Err() != nil {
			res.Skipped++
			res.Items = append(res.Items, ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "skipped", Reason: "dibatalkan"})
			continue
		}
		bytes, status, reason := deleteOne(ctx, e)
		ir := ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: status, Bytes: bytes, Reason: reason}
		res.Items = append(res.Items, ir)
		res.Bytes += bytes
		switch status {
		case "deleted":
			res.Deleted++
		case "skipped":
			res.Skipped++
		default:
			res.Failed++
		}
	}
	return res
}

func deleteOne(ctx context.Context, e Entry) (int64, string, string) {
	if !validName(e.Name) {
		return 0, "skipped", "nama entry tidak valid"
	}
	rootPath := filepath.Clean(e.Root)
	if e.RootKind == RootKindHome {
		if p, err := filepath.EvalSymlinks(rootPath); err == nil {
			rootPath = p
		}
	} else if !isRealDir(rootPath) {
		return 0, "skipped", "root volume bukan direktori nyata"
	}
	infoDir := filepath.Dir(e.InfoPath)
	filesDir := filepath.Dir(e.FilesPath)
	if !inside(rootPath, e.FilesPath) || filepath.Dir(filesDir) != rootPath {
		return 0, "skipped", "path di luar root trash"
	}
	if !isRealDir(filesDir) {
		return 0, "skipped", "files/ bukan direktori nyata"
	}
	if filepath.Base(e.FilesPath) != e.Name {
		return 0, "skipped", "nama entry tidak cocok"
	}
	if e.InfoPath != "" {
		if !inside(rootPath, e.InfoPath) || filepath.Dir(infoDir) != rootPath || !isRealDir(infoDir) {
			return 0, "skipped", "path info di luar root trash"
		}
		if filepath.Base(e.InfoPath) != e.Name+".trashinfo" {
			return 0, "skipped", "nama entry tidak cocok"
		}
	}

	var rootDev uint64
	if st, err := statDev(filesDir); err == nil {
		rootDev = st
	}

	fileExists := false
	if _, err := os.Lstat(e.FilesPath); err == nil {
		fileExists = true
	} else if !os.IsNotExist(err) {
		return 0, "failed", err.Error()
	}
	infoExists := false
	if e.InfoPath != "" {
		if _, err := os.Lstat(e.InfoPath); err == nil {
			infoExists = true
		} else if !os.IsNotExist(err) {
			return 0, "failed", err.Error()
		}
	}
	if !fileExists && !infoExists {
		return 0, "skipped", "entry sudah tidak ada"
	}

	var bytes int64
	if fileExists {
		if b, _, err := fsx.Usage(ctx, e.FilesPath); err == nil {
			bytes = b
		}
		if rootDev != 0 {
			if err := checkNoForeignMount(ctx, e.FilesPath, rootDev); err != nil {
				return 0, "skipped", err.Error()
			}
		}
		if err := removeAllOneFS(ctx, e.FilesPath); err != nil {
			return 0, "failed", err.Error()
		}
	}
	if infoExists {
		if err := removeAllOneFS(ctx, e.InfoPath); err != nil {
			return bytes, "failed", err.Error()
		}
	}
	return bytes, "deleted", ""
}

// Empty mengosongkan isi files/ dan info/ semua root. Direktori root,
// files/, dan info/ tidak dihapus.
func Empty(ctx context.Context, roots []Root, confirm string) (ActionResult, error) {
	confirm = strings.ToUpper(strings.TrimSpace(confirm))
	if confirm != "KOSONGKAN" && confirm != "EMPTY" {
		return ActionResult{}, errors.New("konfirmasi kosongkan harus KOSONGKAN atau EMPTY")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	res := ActionResult{}
	for _, root := range roots {
		if ctx.Err() != nil {
			res.Skipped++
			break
		}
		rootPath := filepath.Clean(root.Path)
		if root.Kind == RootKindHome {
			if p, err := filepath.EvalSymlinks(rootPath); err == nil {
				rootPath = p
			}
		} else if !isRealDir(rootPath) {
			continue
		}
		filesDir := filepath.Join(rootPath, "files")
		infoDir := filepath.Join(rootPath, "info")
		if !isRealDir(filesDir) || !isRealDir(infoDir) {
			continue
		}
		rootDev, _ := statDev(filesDir)
		skipInfo := map[string]bool{}

		names, err := os.ReadDir(filesDir)
		if err != nil {
			res.Failed++
			res.Items = append(res.Items, ItemResult{Status: "failed", Reason: err.Error()})
			continue
		}
		for _, n := range names {
			if !validName(n.Name()) {
				res.Skipped++
				continue
			}
			p := filepath.Join(filesDir, n.Name())
			var b int64
			if rootDev != 0 {
				if err := checkNoForeignMount(ctx, p, rootDev); err != nil {
					res.Skipped++
					skipInfo[n.Name()] = true
					res.Items = append(res.Items, ItemResult{
						ID: EntryID(rootPath, n.Name()), Name: n.Name(), Status: "skipped", Reason: err.Error(),
					})
					continue
				}
			}
			if v, _, err := fsx.Usage(ctx, p); err == nil {
				b = v
			}
			if err := removeAllOneFS(ctx, p); err != nil {
				res.Failed++
				skipInfo[n.Name()] = true
				res.Items = append(res.Items, ItemResult{
					ID: EntryID(rootPath, n.Name()), Name: n.Name(), Status: "failed", Bytes: b, Reason: err.Error(),
				})
				continue
			}
			res.Deleted++
			res.Bytes += b
			res.Items = append(res.Items, ItemResult{
				ID: EntryID(rootPath, n.Name()), Name: n.Name(), Status: "deleted", Bytes: b,
			})
		}

		infos, err := os.ReadDir(infoDir)
		if err != nil {
			res.Failed++
			res.Items = append(res.Items, ItemResult{Status: "failed", Reason: err.Error()})
			continue
		}
		for _, n := range infos {
			p := filepath.Join(infoDir, n.Name())
			if skipInfo[strings.TrimSuffix(n.Name(), ".trashinfo")] {
				continue
			}
			if err := removeAllOneFS(ctx, p); err != nil {
				res.Failed++
				res.Items = append(res.Items, ItemResult{Name: n.Name(), Status: "failed", Reason: err.Error()})
			}
		}
	}
	return res, nil
}

func removeAllOneFS(ctx context.Context, path string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return os.RemoveAll(path)
}

// checkNoForeignMount memastikan tidak ada filesystem lain di bawah entry.
// Ini mencegah Empty/Delete menghapus isi mount yang kebetulan berada di trash.
func checkNoForeignMount(ctx context.Context, path string, rootDev uint64) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return nil
	}
	st, err := statDev(path)
	if err != nil {
		return nil
	}
	if st != rootDev {
		return fmt.Errorf("%w: %s", errForeignMount, path)
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	names, _ := f.Readdirnames(-1)
	f.Close()
	for _, name := range names {
		if err := checkNoForeignMount(ctx, filepath.Join(path, name), rootDev); err != nil {
			return err
		}
	}
	return nil
}

func statDev(path string) (uint64, error) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}
