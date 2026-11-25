package clean

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupHome(t *testing.T) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	return home
}

func TestManualTrash(t *testing.T) {
	home := setupHome(t)
	src := filepath.Join(home, "junk.bin")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := manualTrash(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("file asal masih ada: %v", err)
	}
	if _, err := os.Stat(info.TrashPath); err != nil {
		t.Fatalf("file trash hilang: %v", err)
	}
	if !HasTrashInfo(info.InfoPath) {
		t.Fatalf("metadata trash tidak valid: %s", info.InfoPath)
	}

	body, err := os.ReadFile(info.InfoPath)
	if err != nil {
		t.Fatal(err)
	}
	var encoded string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "Path=") {
			encoded = strings.TrimPrefix(line, "Path=")
		}
	}
	decoded, err := url.PathUnescape(encoded)
	if err != nil || decoded != src {
		t.Fatalf("Path di trashinfo = %q (%q), mau %q", encoded, decoded, src)
	}

	// restore manual (yang dilakukan file manager): pindahkan kembali.
	if err := os.Rename(info.TrashPath, src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("restore gagal: %v", err)
	}
}

func TestMoveToTrashGio(t *testing.T) {
	home := setupHome(t)
	src := filepath.Join(home, "gio-junk.bin")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := MoveToTrash(context.Background(), src)
	if err != nil {
		t.Skipf("trash tidak tersedia di lingkungan ini: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("file asal masih ada: %v", err)
	}
	if info.TrashPath != "" {
		if _, err := os.Stat(info.TrashPath); err != nil {
			t.Fatalf("trash path hilang: %v", err)
		}
	}
}

// TestTrashRestoreWithGio membuktikan metadata buatan manualTrash bisa
// di-restore oleh gio. Dijalankan hanya dengan env khusus karena menyentuh
// trash asli user (file uji dibuat lalu langsung dikembalikan).
func TestTrashRestoreWithGio(t *testing.T) {
	if os.Getenv("CLEAN_TEST_REAL_TRASH") == "" {
		t.Skip("set CLEAN_TEST_REAL_TRASH=1 untuk menguji restore gio")
	}
	t.Setenv("XDG_DATA_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(home, ".cache", fmt.Sprintf("cleanlie-trash-test-%d", time.Now().UnixNano()))
	if err := os.WriteFile(src, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(src) })

	info, err := manualTrash(src)
	if err != nil {
		t.Fatal(err)
	}
	uri := "trash:///" + url.PathEscape(filepath.Base(info.TrashPath))
	out, err := exec.Command("gio", "trash", "--restore", uri).CombinedOutput()
	if err != nil {
		t.Fatalf("gio restore: %v: %s", err, out)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("file tidak kembali: %v", err)
	}
}
