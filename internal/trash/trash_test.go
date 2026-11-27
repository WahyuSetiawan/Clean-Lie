package trash

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	return home
}

func makeRoot(t *testing.T, path, mountPoint string, kind RootKind) Root {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Root{Path: path, MountPoint: mountPoint, Kind: kind}
}

func homeRoot(t *testing.T, home string) Root {
	t.Helper()
	return makeRoot(t, filepath.Join(home, ".local", "share", "Trash"), "", RootKindHome)
}

func escapeTrashPath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func writeInfo(t *testing.T, root Root, name, original string, deleted time.Time) {
	t.Helper()
	if deleted.IsZero() {
		deleted = time.Now()
	}
	body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		escapeTrashPath(original), deleted.Format("2006-01-02T15:04:05"))
	if err := os.WriteFile(filepath.Join(root.Path, "info", name+".trashinfo"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func addFile(t *testing.T, root Root, name, original string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root.Path, "files", filepath.Dir(name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "files", name), content, 0o644); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, root, name, original, time.Now())
}

func addDir(t *testing.T, root Root, name, original string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root.Path, "files", name, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "files", name, "sub", "file"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, root, name, original, time.Now())
}

func findEntry(t *testing.T, rep Report, name string) Entry {
	t.Helper()
	for _, e := range rep.Entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q tidak ditemukan di %+v", name, rep.Entries)
	return Entry{}
}

func TestListHome(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	original := filepath.Join(home, "junk.txt")
	addFile(t, root, "junk.txt", original, []byte("hello"))

	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 1 {
		t.Fatalf("entries = %d, want 1: %+v", len(rep.Entries), rep.Entries)
	}
	e := rep.Entries[0]
	if e.OriginalPath != original || e.Name != "junk.txt" || e.RootKind != RootKindHome {
		t.Fatalf("entry salah: %+v", e)
	}
	if e.Bytes != -1 {
		t.Fatalf("Bytes = %d, want -1 sebelum SizeEntries", e.Bytes)
	}
	if !e.Restorable {
		t.Fatalf("entry harus restorable: %+v", e)
	}
	if e.ID != EntryID(root.Path, "junk.txt") {
		t.Fatalf("ID tidak stabil: %s", e.ID)
	}
	if len(rep.Roots) != 1 || rep.Truncated {
		t.Fatalf("report salah: %+v", rep)
	}
}

func TestListEncodedAndOrphan(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	original := filepath.Join(home, "a b", "file name.txt")
	if err := os.MkdirAll(filepath.Dir(original), 0o755); err != nil {
		t.Fatal(err)
	}
	addFile(t, root, "file name.txt", original, []byte("x"))

	// File tanpa .trashinfo harus muncul sebagai orphan.
	if err := os.WriteFile(filepath.Join(root.Path, "files", "orphan"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Info tanpa file.
	writeInfo(t, root, "ghost", filepath.Join(home, "ghost"), time.Now())

	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 3 {
		t.Fatalf("entries = %d, want 3: %+v", len(rep.Entries), rep.Entries)
	}
	if e := findEntry(t, rep, "file name.txt"); e.OriginalPath != original {
		t.Fatalf("path encoded salah: %+v", e)
	}
	orphan := findEntry(t, rep, "orphan")
	if orphan.Restorable || orphan.InfoPath != "" || orphan.Reason == "" {
		t.Fatalf("orphan salah: %+v", orphan)
	}
	ghost := findEntry(t, rep, "ghost")
	if ghost.Restorable || !strings.Contains(ghost.Reason, "hilang") {
		t.Fatalf("ghost salah: %+v", ghost)
	}
}

func TestListRejectsUnsafeMetadata(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	// Relative path di home tidak sah.
	if err := os.WriteFile(filepath.Join(root.Path, "info", "rel.trashinfo"), []byte("[Trash Info]\nPath=relative\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Path dengan .. setelah unescape harus ditolak.
	if err := os.WriteFile(filepath.Join(root.Path, "info", "dots.trashinfo"), []byte("[Trash Info]\nPath=%2E%2E/evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Symlink .trashinfo harus dilewati.
	realInfo := filepath.Join(home, "real.trashinfo")
	if err := os.WriteFile(realInfo, []byte("[Trash Info]\nPath="+escapeTrashPath(filepath.Join(home, "x"))+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realInfo, filepath.Join(root.Path, "info", "link.trashinfo")); err != nil {
		t.Fatal(err)
	}

	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rep.Entries {
		if e.Name == "link" {
			t.Fatalf("symlink .trashinfo tidak boleh jadi entry: %+v", e)
		}
		if e.Name == "rel" || e.Name == "dots" {
			if e.Restorable || e.Reason == "" {
				t.Fatalf("metadata tidak aman harus tidak restorable: %+v", e)
			}
		}
	}
	if len(rep.Errors) == 0 {
		t.Fatal("harus ada error untuk symlink .trashinfo")
	}
}

func TestListTruncatedAndRootErrors(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	for i := 0; i < 3; i++ {
		addFile(t, root, fmt.Sprintf("f%d", i), filepath.Join(home, fmt.Sprintf("f%d", i)), []byte("x"))
	}
	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}, MaxEntries: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Truncated || len(rep.Entries) != 2 {
		t.Fatalf("truncate salah: truncated=%v entries=%d", rep.Truncated, len(rep.Entries))
	}

	bad := Root{Path: filepath.Join(home, "no-trash"), Kind: RootKindHome}
	rep, err = List(context.Background(), ListOptions{Roots: []Root{bad}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) == 0 {
		t.Fatal("root tanpa info/ harus menghasilkan error")
	}
}

func TestRestoreHome(t *testing.T) {
	home := setupHome(t)
	t.Setenv("CLEANLIE_NO_GIO", "1")
	root := homeRoot(t, home)
	original := filepath.Join(home, "restored.txt")
	addFile(t, root, "restored.txt", original, []byte("hello"))

	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(t, rep, "restored.txt")
	if err := Restore(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(original); err != nil || string(b) != "hello" {
		t.Fatalf("isi restore salah: %q %v", b, err)
	}
	if _, err := os.Lstat(e.FilesPath); !os.IsNotExist(err) {
		t.Fatalf("file trash masih ada: %v", err)
	}
	if _, err := os.Lstat(e.InfoPath); !os.IsNotExist(err) {
		t.Fatalf(".trashinfo masih ada: %v", err)
	}
}

func TestRestoreConflictAndMissingParent(t *testing.T) {
	home := setupHome(t)
	t.Setenv("CLEANLIE_NO_GIO", "1")
	root := homeRoot(t, home)

	conflict := filepath.Join(home, "conflict.txt")
	if err := os.WriteFile(conflict, []byte("asli"), 0o644); err != nil {
		t.Fatal(err)
	}
	addFile(t, root, "conflict.txt", conflict, []byte("trash"))
	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	e := findEntry(t, rep, "conflict.txt")
	if err := Restore(context.Background(), e); !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("err = %v, want ErrDestinationExists", err)
	}
	if _, err := os.Lstat(e.FilesPath); err != nil {
		t.Fatalf("file trash harus tetap ada: %v", err)
	}

	missing := filepath.Join(home, "newdir", "missing.txt")
	addFile(t, root, "missing.txt", missing, []byte("trash"))
	rep, _ = List(context.Background(), ListOptions{Roots: []Root{root}})
	e = findEntry(t, rep, "missing.txt")
	if e.Restorable {
		t.Fatalf("entry dengan parent hilang tidak boleh restorable: %+v", e)
	}
	if err := Restore(context.Background(), e); !errors.Is(err, ErrParentMissing) {
		t.Fatalf("err = %v, want ErrParentMissing", err)
	}
}

func TestRestoreSymlink(t *testing.T) {
	home := setupHome(t)
	t.Setenv("CLEANLIE_NO_GIO", "1")
	root := homeRoot(t, home)
	target := filepath.Join(home, "target")
	if err := os.WriteFile(target, []byte("target"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "link")
	if err := os.Symlink(target, filepath.Join(root.Path, "files", "link")); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, root, "link", link, time.Now())

	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	e := findEntry(t, rep, "link")
	if err := Restore(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(link)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("harus kembali sebagai symlink: %v %v", fi, err)
	}
	if got, _ := os.Readlink(link); got != target {
		t.Fatalf("target symlink = %q, want %q", got, target)
	}
}

func TestRestoreRejectsOutsideRoot(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	addFile(t, root, "x", filepath.Join(home, "x"), []byte("x"))
	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	e := findEntry(t, rep, "x")
	e.FilesPath = filepath.Join(home, "outside")
	if err := Restore(context.Background(), e); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err = %v, want ErrOutsideRoot", err)
	}
}

func TestVolumeListAndRestore(t *testing.T) {
	vol := t.TempDir()
	root := makeRoot(t, filepath.Join(vol, ".Trash-1000"), vol, RootKindVolume)
	music := filepath.Join(vol, "Music")
	if err := os.MkdirAll(music, 0o755); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(music, "song.mp3")
	addFile(t, root, "song.mp3", "Music/song.mp3", []byte("audio"))

	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(t, rep, "song.mp3")
	if e.OriginalPath != original {
		t.Fatalf("path volume = %q, want %q", e.OriginalPath, original)
	}
	if err := Restore(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(original); string(b) != "audio" {
		t.Fatalf("isi restore volume salah: %q", b)
	}
}

func TestVolumeRejectsEscape(t *testing.T) {
	vol := t.TempDir()
	root := makeRoot(t, filepath.Join(vol, ".Trash-1000"), vol, RootKindVolume)
	if err := os.WriteFile(filepath.Join(root.Path, "info", "evil.trashinfo"), []byte("[Trash Info]\nPath=../../etc/passwd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "files", "evil"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(t, rep, "evil")
	if e.Restorable || !strings.Contains(e.Reason, "..") {
		t.Fatalf("escape volume harus ditolak: %+v", e)
	}
}

func TestDeleteAndOrphan(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	addFile(t, root, "delete.txt", filepath.Join(home, "delete.txt"), []byte("data"))
	if err := os.WriteFile(filepath.Join(root.Path, "files", "orphan"), []byte("orphan"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})

	res := Delete(context.Background(), []Entry{findEntry(t, rep, "delete.txt"), findEntry(t, rep, "orphan")})
	if res.Deleted != 2 || res.Failed != 0 || res.Bytes <= 0 {
		t.Fatalf("delete salah: %+v", res)
	}
	for _, name := range []string{"delete.txt", "orphan"} {
		if _, err := os.Lstat(filepath.Join(root.Path, "files", name)); !os.IsNotExist(err) {
			t.Fatalf("%s masih ada: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root.Path, "info", "delete.txt.trashinfo")); !os.IsNotExist(err) {
		t.Fatalf("info delete.txt masih ada: %v", err)
	}
}

func TestEmpty(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	addFile(t, root, "a", filepath.Join(home, "a"), []byte("a"))
	addFile(t, root, "b", filepath.Join(home, "b"), []byte("b"))
	if err := os.WriteFile(filepath.Join(root.Path, "files", "orphan"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Empty(context.Background(), []Root{root}, "salah"); err == nil {
		t.Fatal("konfirmasi salah harus error")
	}
	res, err := Empty(context.Background(), []Root{root}, "KOSONGKAN")
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 3 || res.Bytes <= 0 {
		t.Fatalf("empty salah: %+v", res)
	}
	for _, dir := range []string{"files", "info"} {
		names, err := os.ReadDir(filepath.Join(root.Path, dir))
		if err != nil || len(names) != 0 {
			t.Fatalf("%s harus kosong: %v %v", dir, names, err)
		}
	}
}

func TestResolveAndSize(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	original := filepath.Join(home, "dir")
	addDir(t, root, "dir", original)

	got, ok := Resolve(context.Background(), original)
	if !ok || got.Name != "dir" || got.OriginalPath != original {
		t.Fatalf("resolve salah: %+v ok=%v", got, ok)
	}
	if _, ok := Resolve(context.Background(), filepath.Join(home, "tidak-ada")); ok {
		t.Fatal("resolve path tidak ada harus false")
	}

	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	entries := SizeEntries(context.Background(), rep.Entries, SizeOptions{Workers: 2})
	if len(entries) != 1 || entries[0].Bytes < 0 {
		t.Fatalf("size salah: %+v", entries)
	}
	if entries[0].Bytes == 0 {
		t.Fatalf("size harus > 0: %+v", entries[0])
	}

	missing := SizeEntries(context.Background(), []Entry{{FilesPath: filepath.Join(home, "hilang"), Bytes: -1}}, SizeOptions{})
	if missing[0].Bytes != -1 || missing[0].Reason == "" {
		t.Fatalf("missing size salah: %+v", missing[0])
	}
}

func TestSizeDedupesHardlinks(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	payload := make([]byte, 1<<20)
	addDir(t, root, "one", filepath.Join(home, "one"))
	addDir(t, root, "two", filepath.Join(home, "two"))
	src := filepath.Join(root.Path, "files", "one", "sub", "file")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root.Path, "files", "two", "sub", "file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(src, filepath.Join(root.Path, "files", "two", "sub", "file")); err != nil {
		t.Fatal(err)
	}
	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	entries := SizeEntries(context.Background(), rep.Entries, SizeOptions{Workers: 2})
	var total int64
	for _, e := range entries {
		total += e.Bytes
	}
	if total >= int64(len(payload))*3/2 {
		t.Fatalf("hardlink terhitung dua kali: total=%d", total)
	}
	if total < int64(len(payload)) {
		t.Fatalf("total terlalu kecil: %d", total)
	}
}

func TestSizeCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	entries := []Entry{{ID: "a", FilesPath: "/tidak-ada", Bytes: -1}, {ID: "b", FilesPath: "/tidak-ada", Bytes: -1}}
	done := make(chan []Entry, 1)
	go func() { done <- SizeEntries(ctx, entries, SizeOptions{Workers: 2}) }()
	select {
	case got := <-done:
		if len(got) != 2 {
			t.Fatalf("len = %d", len(got))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SizeEntries tidak berhenti setelah cancel")
	}
}

func BenchmarkList1000(b *testing.B) {
	home := b.TempDir()
	b.Setenv("HOME", home)
	b.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	root := makeRootB(b, filepath.Join(home, ".local", "share", "Trash"), "", RootKindHome)
	for i := 0; i < 1000; i++ {
		name := fmt.Sprintf("f%04d", i)
		if err := os.WriteFile(filepath.Join(root.Path, "files", name), []byte("x"), 0o644); err != nil {
			b.Fatal(err)
		}
		body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=2026-01-01T00:00:%02d\n", escapeTrashPath(filepath.Join(home, name)), i%60)
		if err := os.WriteFile(filepath.Join(root.Path, "info", name+".trashinfo"), []byte(body), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rep, err := List(context.Background(), ListOptions{Roots: []Root{root}})
		if err != nil || len(rep.Entries) != 1000 {
			b.Fatalf("list: %v %d", err, len(rep.Entries))
		}
	}
}

func makeRootB(b *testing.B, path, mountPoint string, kind RootKind) Root {
	b.Helper()
	if err := os.MkdirAll(filepath.Join(path, "files"), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "info"), 0o755); err != nil {
		b.Fatal(err)
	}
	return Root{Path: path, MountPoint: mountPoint, Kind: kind}
}

func TestUsageMatchesFSX(t *testing.T) {
	home := setupHome(t)
	root := homeRoot(t, home)
	addDir(t, root, "dir", filepath.Join(home, "dir"))
	rep, _ := List(context.Background(), ListOptions{Roots: []Root{root}})
	e := rep.Entries[0]
	got, _, err := fsx.Usage(context.Background(), e.FilesPath)
	if err != nil {
		t.Fatal(err)
	}
	sized := SizeEntries(context.Background(), []Entry{e}, SizeOptions{})
	if sized[0].Bytes != got {
		t.Fatalf("size = %d, fsx = %d", sized[0].Bytes, got)
	}
}
