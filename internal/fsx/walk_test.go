package fsx

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func duBytes(t *testing.T, path string) int64 {
	t.Helper()
	out, err := exec.Command("du", "-s", "--block-size=1", path).Output()
	if err != nil {
		t.Fatalf("du: %v", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		t.Fatal("du: output kosong")
	}
	n, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWalkMatchesDu(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.bin"), 1<<20)
	writeFile(t, filepath.Join(dir, "sub", "b.bin"), 2<<20)

	// hard link: harus dihitung sekali, sama seperti du.
	if err := os.Link(filepath.Join(dir, "a.bin"), filepath.Join(dir, "a-link.bin")); err != nil {
		t.Fatal(err)
	}
	// symlink: tidak diikuti.
	if err := os.Symlink(filepath.Join(dir, "sub"), filepath.Join(dir, "sub-link")); err != nil {
		t.Fatal(err)
	}

	got, err := Walk(context.Background(), dir, Options{Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	want := duBytes(t, dir)
	if diff := got.Size - want; diff > 8192 || diff < -8192 {
		t.Fatalf("size = %d, du = %d (selisih %d)", got.Size, want, diff)
	}
	// a.bin, b.bin, symlink (hard link tidak dihitung).
	if got.Files != 3 {
		t.Fatalf("files = %d, mau 3", got.Files)
	}
	if got.Dirs < 2 {
		t.Fatalf("dirs = %d, mau minimal 2", got.Dirs)
	}
}

func TestWalkCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Walk(ctx, t.TempDir(), Options{Workers: 2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, mau context.Canceled", err)
	}
}

func TestUsageFileAndDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "one.bin")
	writeFile(t, file, 4096)

	size, files, err := Usage(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}
	if size < 4096 || files != 1 {
		t.Fatalf("file: size=%d files=%d", size, files)
	}

	sizeDir, filesDir, err := Usage(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if sizeDir < size || filesDir != 1 {
		t.Fatalf("dir: size=%d files=%d", sizeDir, filesDir)
	}
}

func TestWalkTopN(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "small"), 1024)
	writeFile(t, filepath.Join(dir, "big"), 3<<20)
	writeFile(t, filepath.Join(dir, "mid"), 1<<20)

	got, err := Walk(context.Background(), dir, Options{Workers: 2, TopN: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Top) != 2 {
		t.Fatalf("top = %d, mau 2", len(got.Top))
	}
	if filepath.Base(got.Top[0].Path) != "big" || filepath.Base(got.Top[1].Path) != "mid" {
		t.Fatalf("top tidak terurut: %+v", got.Top)
	}
}

func TestHuman(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1 << 20, "1.0 MiB"},
		{3 << 30, "3.0 GiB"},
	}
	for _, c := range cases {
		if got := Human(c.in); got != c.want {
			t.Errorf("Human(%d) = %q, mau %q", c.in, got, c.want)
		}
	}
}

func TestStatfs(t *testing.T) {
	v, err := Statfs("/")
	if err != nil {
		t.Fatal(err)
	}
	if v.Total <= 0 || v.Free < 0 || v.Used <= 0 || v.Used > v.Total {
		t.Fatalf("volume tidak masuk akal: %+v", v)
	}
}
