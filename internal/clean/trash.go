package clean

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// ErrCrossDevice dikembalikan bila path berada di filesystem lain dan
// tidak bisa dipindah ke trash milik home.
var ErrCrossDevice = errors.New("beda filesystem")

// TrashInfo adalah lokasi hasil pemindahan ke trash.
type TrashInfo struct {
	Path      string `json:"path"`
	TrashPath string `json:"trashPath"`
	InfoPath  string `json:"infoPath"`
}

// TrashDir mengembalikan direktori trash milik user: ~/.Trash di macOS,
// freedesktop (~/.local/share/Trash) di Linux.
func TrashDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir(), ".Trash")
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "Trash")
	}
	return filepath.Join(homeDir(), ".local", "share", "Trash")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

// MoveToTrash memakai `gio trash` bila tersedia (Linux), lalu fallback
// manual. Di macOS langsung dipakai ~/.Trash milik Finder.
func MoveToTrash(ctx context.Context, path string) (TrashInfo, error) {
	if runtime.GOOS != "darwin" && tools.Available("gio") {
		res := tools.Run(ctx, []string{"gio", "trash", "--", path}, 2*time.Minute)
		if res.Err == nil {
			return TrashInfo{Path: path}, nil
		}
	}
	return manualTrash(path)
}

func manualTrash(path string) (TrashInfo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return TrashInfo{}, err
	}
	if _, err := os.Lstat(abs); err != nil {
		return TrashInfo{}, err
	}

	trash := TrashDir()
	filesDir := filepath.Join(trash, "files")
	infoDir := filepath.Join(trash, "info")
	if runtime.GOOS == "darwin" {
		// macOS tidak punya metadata .trashinfo; Finder cukup melihat isi ~/.Trash.
		filesDir = trash
		infoDir = ""
	}

	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return TrashInfo{}, err
	}
	if infoDir != "" {
		if err := os.MkdirAll(infoDir, 0o700); err != nil {
			return TrashInfo{}, err
		}
	}

	var srcDev, trashDev syscall.Stat_t
	if err := syscall.Stat(filepath.Dir(abs), &srcDev); err != nil {
		return TrashInfo{}, err
	}
	if err := syscall.Stat(filesDir, &trashDev); err != nil {
		return TrashInfo{}, err
	}
	if srcDev.Dev != trashDev.Dev {
		return TrashInfo{}, ErrCrossDevice
	}

	name := filepath.Base(abs)
	dest := filepath.Join(filesDir, name)
	for i := 1; ; i++ {
		if _, err := os.Lstat(dest); errors.Is(err, os.ErrNotExist) {
			break
		}
		dest = filepath.Join(filesDir, name+"."+strconv.Itoa(i))
	}

	if err := os.Rename(abs, dest); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return TrashInfo{}, ErrCrossDevice
		}
		return TrashInfo{}, err
	}

	infoPath := ""
	if infoDir != "" {
		infoPath = filepath.Join(infoDir, filepath.Base(dest)+".trashinfo")
		body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
			escapeTrashPath(abs), time.Now().Format("2006-01-02T15:04:05"))
		if err := os.WriteFile(infoPath, []byte(body), 0o600); err != nil {
			// kembalikan file supaya tidak ada sampah tanpa metadata restore.
			_ = os.Rename(dest, abs)
			return TrashInfo{}, err
		}
	}

	return TrashInfo{Path: abs, TrashPath: dest, InfoPath: infoPath}, nil
}

func escapeTrashPath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

// HasTrashInfo memeriksa apakah path punya metadata trash (untuk test/UI).
func HasTrashInfo(infoPath string) bool {
	b, err := os.ReadFile(infoPath)
	return err == nil && strings.Contains(string(b), "[Trash Info]")
}
