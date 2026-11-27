package preview

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewFileAndSymlink(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(file, make([]byte, 1<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Preview(context.Background(), file, 0)
	if res.Kind != "file" || res.Bytes <= 0 || res.Files != 1 {
		t.Fatalf("file preview salah: %+v", res)
	}

	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	res = Preview(context.Background(), link, 0)
	if res.Kind != "symlink" || res.Bytes >= 1<<20 {
		t.Fatalf("symlink harus tidak diikuti: %+v", res)
	}
}

func TestPreviewDirLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 4; i++ {
		if err := os.WriteFile(filepath.Join(dir, string(rune('a'+i))), make([]byte, 4096), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res := Preview(context.Background(), dir, 2)
	if res.Kind != "dir" || !res.Truncated || len(res.Entries) != 2 {
		t.Fatalf("dir preview salah: %+v", res)
	}
	if res.Bytes <= 0 || res.Files != 4 {
		t.Fatalf("total dir salah: %+v", res)
	}
	if res.Entries[0].Bytes < res.Entries[1].Bytes {
		t.Fatalf("entry harus urut ukuran: %+v", res.Entries)
	}
}

func TestPreviewMissing(t *testing.T) {
	res := Preview(context.Background(), filepath.Join(t.TempDir(), "missing"), 0)
	if res.Kind != "missing" || res.Error == "" {
		t.Fatalf("missing preview salah: %+v", res)
	}
}
