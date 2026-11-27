package trash

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

var (
	ErrDestinationExists = errors.New("tujuan restore sudah ada")
	ErrParentMissing     = errors.New("folder tujuan tidak ada")
	ErrProtected         = errors.New("tujuan restore protected")
	ErrOutsideRoot       = errors.New("path di luar root trash")
	ErrBadName           = errors.New("nama entry tidak valid")
	ErrMissing           = errors.New("file trash atau metadata hilang")
)

// Restore mengembalikan satu entry ke path asal yang tertulis di
// .trashinfo. Path dari audit tidak dipercaya; metadata dibaca ulang.
func Restore(ctx context.Context, e Entry) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !validName(e.Name) {
		return ErrBadName
	}
	rootPath := filepath.Clean(e.Root)
	if rootPath == "" || rootPath == "." {
		return ErrOutsideRoot
	}
	if e.RootKind == RootKindHome {
		if p, err := filepath.EvalSymlinks(rootPath); err == nil {
			rootPath = p
		}
	} else if !isRealDir(rootPath) {
		return fmt.Errorf("%w: root volume bukan direktori nyata", ErrOutsideRoot)
	}

	infoDir := filepath.Dir(e.InfoPath)
	filesDir := filepath.Dir(e.FilesPath)
	if !inside(rootPath, e.InfoPath) || !inside(rootPath, e.FilesPath) {
		return ErrOutsideRoot
	}
	if filepath.Dir(infoDir) != rootPath || filepath.Dir(filesDir) != rootPath {
		return ErrOutsideRoot
	}
	if !isRealDir(infoDir) || !isRealDir(filesDir) {
		return fmt.Errorf("%w: info/ atau files/ bukan direktori nyata", ErrOutsideRoot)
	}
	if filepath.Base(e.InfoPath) != e.Name+".trashinfo" || filepath.Base(e.FilesPath) != e.Name {
		return ErrBadName
	}
	if !isRealFile(e.InfoPath) {
		return fmt.Errorf("%w: .trashinfo bukan file biasa", ErrMissing)
	}
	if _, err := os.Lstat(e.FilesPath); err != nil {
		return fmt.Errorf("%w: %v", ErrMissing, err)
	}

	root := Root{Path: rootPath, MountPoint: mountPointOf(e), Kind: e.RootKind}
	original, _, err := parseInfo(e.InfoPath, root)
	if err != nil {
		return fmt.Errorf("baca .trashinfo: %w", err)
	}
	if !filepath.IsAbs(original) {
		return fmt.Errorf("tujuan restore bukan path absolut: %q", original)
	}
	if rules.IsProtected(original) {
		return ErrProtected
	}
	parent := filepath.Dir(original)
	fi, err := os.Stat(parent)
	if err != nil || !fi.IsDir() {
		return fmt.Errorf("%w: %s", ErrParentMissing, parent)
	}
	if resolved, err := filepath.EvalSymlinks(parent); err == nil && rules.IsProtected(filepath.Join(resolved, filepath.Base(original))) {
		return ErrProtected
	}
	if _, err := os.Lstat(original); err == nil {
		return ErrDestinationExists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("periksa tujuan: %w", err)
	}

	// gio hanya untuk home trash. URI `trash:///` untuk topdir volume tidak
	// portabel; volume trash cukup dipindah dengan rename karena satu filesystem.
	if e.RootKind == RootKindHome && os.Getenv("CLEANLIE_NO_GIO") != "1" && tools.Available("gio") {
		uri := "trash:///" + url.PathEscape(e.Name)
		out := tools.Run(ctx, []string{"gio", "trash", "--restore", uri}, 2*time.Minute)
		if out.Err == nil {
			if _, err := os.Lstat(original); err != nil {
				return fmt.Errorf("gio restore selesai tetapi tujuan tidak ada: %w", err)
			}
			// Sebagian gio/gvfs meninggalkan info; bersihkan agar tidak jadi
			// entry hantu di daftar trash.
			if _, err := os.Lstat(e.InfoPath); err == nil {
				if err := os.Remove(e.InfoPath); err != nil {
					return fmt.Errorf("hapus .trashinfo sisa: %w", err)
				}
			}
			return nil
		}
	}

	if err := os.Rename(e.FilesPath, original); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return ErrCrossDevice
		}
		return fmt.Errorf("pindahkan dari trash: %w", err)
	}
	if err := os.Remove(e.InfoPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("hapus .trashinfo: %w", err)
	}
	if _, err := os.Lstat(original); err != nil {
		return fmt.Errorf("verifikasi restore: %w", err)
	}
	return nil
}

func mountPointOf(e Entry) string {
	if e.MountPoint != "" {
		return e.MountPoint
	}
	if e.RootKind != RootKindVolume {
		return ""
	}
	base := filepath.Base(e.Root)
	if strings.HasPrefix(base, ".Trash-") {
		return filepath.Dir(e.Root)
	}
	if base != "" && filepath.Base(filepath.Dir(e.Root)) == ".Trash" {
		return filepath.Dir(filepath.Dir(e.Root))
	}
	return ""
}
