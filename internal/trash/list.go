package trash

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

// ListOptions mengatur List. Roots diisi di test; MaxEntries default 5000.
type ListOptions struct {
	Roots      []Root
	MaxEntries int
	Now        func() time.Time
}

// List membaca metadata trash. Ukuran tidak dihitung di sini (Bytes = -1);
// pakai SizeEntries untuk menghitungnya secara streaming.
func List(ctx context.Context, opts ListOptions) (Report, error) {
	started := time.Now()
	if opts.Now != nil {
		started = opts.Now()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 5000
	}
	roots := opts.Roots
	if roots == nil {
		roots = DiscoverRoots()
	}

	rep := Report{Roots: roots, ScannedAt: started.Format(time.RFC3339)}
	remaining := opts.MaxEntries
	for _, root := range roots {
		if ctx.Err() != nil {
			rep.Errors = append(rep.Errors, "pemindaian dibatalkan")
			break
		}
		if remaining <= 0 {
			rep.Truncated = true
			break
		}
		entries, errs, truncated, err := listRoot(ctx, root, remaining)
		rep.Errors = append(rep.Errors, errs...)
		if err != nil {
			rep.Errors = append(rep.Errors, root.Path+": "+err.Error())
			continue
		}
		rep.Entries = append(rep.Entries, entries...)
		remaining -= len(entries)
		if truncated {
			rep.Truncated = true
		}
	}

	sort.SliceStable(rep.Entries, func(i, j int) bool {
		if !rep.Entries[i].DeletedAt.Equal(rep.Entries[j].DeletedAt) {
			return rep.Entries[i].DeletedAt.After(rep.Entries[j].DeletedAt)
		}
		return rep.Entries[i].Name < rep.Entries[j].Name
	})
	for _, e := range rep.Entries {
		if e.Bytes >= 0 {
			rep.Bytes += e.Bytes
		}
	}
	rep.DurationMs = time.Since(started).Milliseconds()
	return rep, nil
}

func listRoot(ctx context.Context, root Root, max int) ([]Entry, []string, bool, error) {
	rootPath := filepath.Clean(root.Path)
	if root.Kind == RootKindHome {
		if p, err := filepath.EvalSymlinks(rootPath); err == nil {
			rootPath = p
		}
	} else if !isRealDir(rootPath) {
		return nil, nil, false, fmt.Errorf("root volume bukan direktori nyata")
	}

	infoDir := filepath.Join(rootPath, "info")
	filesDir := filepath.Join(rootPath, "files")
	if !isRealDir(infoDir) {
		return nil, nil, false, fmt.Errorf("info/ bukan direktori nyata")
	}
	if !isRealDir(filesDir) {
		return nil, nil, false, fmt.Errorf("files/ bukan direktori nyata")
	}

	names, err := os.ReadDir(infoDir)
	if err != nil {
		return nil, nil, false, err
	}
	infos := make([]string, 0, len(names))
	for _, n := range names {
		if !strings.HasSuffix(n.Name(), ".trashinfo") {
			continue
		}
		infos = append(infos, n.Name())
	}
	sort.Strings(infos)

	truncated := false
	if len(infos) > max {
		infos = infos[:max]
		truncated = true
	}

	var out []Entry
	var errs []string
	seenNames := map[string]bool{}
	for _, infoName := range infos {
		if ctx.Err() != nil {
			break
		}
		name := strings.TrimSuffix(infoName, ".trashinfo")
		if !validName(name) {
			errs = append(errs, infoName+": nama tidak valid")
			continue
		}
		seenNames[name] = true
		infoPath := filepath.Join(infoDir, infoName)
		if !isRealFile(infoPath) {
			errs = append(errs, infoName+": .trashinfo bukan file biasa (symlink ditolak)")
			continue
		}
		filesPath := filepath.Join(filesDir, name)
		fi, fileErr := os.Lstat(filesPath)
		fileExists := fileErr == nil

		original, deletedAt, parseErr := parseInfo(infoPath, root)
		if deletedAt.IsZero() {
			if fi != nil {
				deletedAt = fi.ModTime()
			} else if st, err := os.Stat(infoPath); err == nil {
				deletedAt = st.ModTime()
			}
		}

		e := Entry{
			ID:         EntryID(rootPath, name),
			Name:       name,
			DeletedAt:  deletedAt,
			Bytes:      -1,
			FilesPath:  filesPath,
			InfoPath:   infoPath,
			Root:       rootPath,
			MountPoint: root.MountPoint,
			RootKind:   root.Kind,
		}
		if parseErr != nil {
			e.Reason = parseErr.Error()
		} else {
			e.OriginalPath = original
		}

		restorable, reason := restorableState(e.OriginalPath, fileExists, parseErr)
		e.Restorable = restorable
		if !restorable && e.Reason == "" {
			e.Reason = reason
		}
		out = append(out, e)
	}

	// File tanpa .trashinfo tetap tampil sebagai orphan agar bisa dihapus.
	files, err := os.ReadDir(filesDir)
	if err != nil {
		errs = append(errs, "files/: "+err.Error())
	} else {
		for _, n := range files {
			if len(out) >= max {
				truncated = true
				break
			}
			name := n.Name()
			if !validName(name) || seenNames[name] {
				continue
			}
			filesPath := filepath.Join(filesDir, name)
			fi, err := os.Lstat(filesPath)
			if err != nil {
				continue
			}
			out = append(out, Entry{
				ID:         EntryID(rootPath, name),
				Name:       name,
				DeletedAt:  fi.ModTime(),
				Bytes:      -1,
				FilesPath:  filesPath,
				Root:       rootPath,
				MountPoint: root.MountPoint,
				RootKind:   root.Kind,
				Restorable: false,
				Reason:     "metadata .trashinfo hilang",
			})
		}
	}
	return out, errs, truncated, nil
}

func parseInfo(infoPath string, root Root) (string, time.Time, error) {
	b, err := os.ReadFile(infoPath)
	if err != nil {
		return "", time.Time{}, err
	}
	var rawPath, rawDate string
	inSection := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = line == "[Trash Info]"
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "Path=") {
			rawPath = strings.TrimPrefix(line, "Path=")
		} else if strings.HasPrefix(line, "DeletionDate=") {
			rawDate = strings.TrimPrefix(line, "DeletionDate=")
		}
	}
	if rawPath == "" {
		return "", time.Time{}, fmt.Errorf("metadata .trashinfo tidak punya Path=")
	}
	if hasDotDot(rawPath) {
		return "", time.Time{}, fmt.Errorf("Path= mengandung ..")
	}
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		decoded = rawPath
	}
	if hasDotDot(decoded) {
		return "", time.Time{}, fmt.Errorf("Path= mengandung ..")
	}
	original, err := normalizeOriginal(decoded, root)
	if err != nil {
		return "", time.Time{}, err
	}

	var deleted time.Time
	if rawDate != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", rawDate, time.Local); err == nil {
			deleted = t
		} else if t, err := time.Parse(time.RFC3339, rawDate); err == nil {
			deleted = t
		}
	}
	return original, deleted, nil
}

func normalizeOriginal(p string, root Root) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("Path= kosong")
	}
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("Path= mengandung NUL")
	}
	if root.Kind == RootKindHome {
		if !filepath.IsAbs(p) {
			return "", fmt.Errorf("Path= home harus absolut")
		}
		p = filepath.Clean(p)
		if p == "/" {
			return "", fmt.Errorf("Path= tidak boleh root")
		}
		return p, nil
	}

	mp := filepath.Clean(root.MountPoint)
	if mp == "" {
		return "", fmt.Errorf("mount point volume kosong")
	}
	if filepath.IsAbs(p) {
		p = filepath.Clean(p)
		if !inside(mp, p) {
			return "", fmt.Errorf("Path= absolut di luar mount point")
		}
		return p, nil
	}
	p = filepath.Clean(p)
	if p == "." || p == ".." {
		return "", fmt.Errorf("Path= relatif tidak valid")
	}
	abs := filepath.Join(mp, p)
	if !inside(mp, abs) {
		return "", fmt.Errorf("Path= relatif keluar dari mount point")
	}
	return abs, nil
}

func hasDotDot(p string) bool {
	for _, part := range strings.Split(strings.ReplaceAll(p, `\`, "/"), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func restorableState(original string, fileExists bool, parseErr error) (bool, string) {
	if !fileExists {
		return false, "file trash hilang"
	}
	if parseErr != nil || original == "" {
		return false, "metadata .trashinfo rusak"
	}
	if rules.IsProtected(original) {
		return false, "tujuan restore protected"
	}
	if _, err := os.Lstat(original); err == nil {
		return false, "tujuan restore sudah ada"
	} else if !os.IsNotExist(err) {
		return false, "tidak bisa memeriksa tujuan restore: " + err.Error()
	}
	parent := filepath.Dir(original)
	fi, err := os.Stat(parent)
	if err != nil || !fi.IsDir() {
		return false, "folder tujuan tidak ada"
	}
	if resolved, err := filepath.EvalSymlinks(parent); err == nil && rules.IsProtected(filepath.Join(resolved, filepath.Base(original))) {
		return false, "folder tujuan protected"
	}
	return true, ""
}
