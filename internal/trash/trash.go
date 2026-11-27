// Package trash mengelola isi trash freedesktop (home dan volume Linux
// native). Paket ini sengaja tidak mengimpor internal/clean: clean justru
// memakai paket ini untuk menemukan TrashPath setelah `gio trash`.
package trash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// RootKind membedakan trash milik home dan trash per-volume.
type RootKind string

const (
	RootKindHome   RootKind = "home"
	RootKindVolume RootKind = "volume"
)

// Root adalah satu direktori trash yang dikenal.
type Root struct {
	Path       string   `json:"path"`
	MountPoint string   `json:"mountPoint,omitempty"`
	Kind       RootKind `json:"kind"`
}

// Entry adalah satu item di trash.
type Entry struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalPath string    `json:"originalPath"`
	DeletedAt    time.Time `json:"deletedAt"`
	Bytes        int64     `json:"bytes"` // -1 = belum diukur
	FilesPath    string    `json:"-"`
	InfoPath     string    `json:"-"`
	Root         string    `json:"root"`
	MountPoint   string    `json:"mountPoint,omitempty"`
	RootKind     RootKind  `json:"rootKind"`
	Restorable   bool      `json:"restorable"`
	Reason       string    `json:"reason,omitempty"`
}

// Report adalah hasil pemindaian metadata trash.
type Report struct {
	Roots      []Root   `json:"roots"`
	Entries    []Entry  `json:"entries"`
	Bytes      int64    `json:"bytes"`
	Errors     []string `json:"errors,omitempty"`
	Truncated  bool     `json:"truncated"`
	ScannedAt  string   `json:"scannedAt"`
	DurationMs int64    `json:"durationMs"`
}

// ItemResult adalah hasil satu item restore/delete/empty.
type ItemResult struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalPath string `json:"originalPath"`
	Status       string `json:"status"` // restored | deleted | skipped | failed | planned
	Bytes        int64  `json:"bytes"`
	Reason       string `json:"reason,omitempty"`
}

// ActionResult adalah ringkasan operasi trash.
type ActionResult struct {
	Items      []ItemResult `json:"items"`
	Restored   int          `json:"restored"`
	Deleted    int          `json:"deleted"`
	Skipped    int          `json:"skipped"`
	Failed     int          `json:"failed"`
	Bytes      int64        `json:"bytes"`
	AuditError string       `json:"auditError,omitempty"`
}

// ErrCrossDevice dikembalikan bila restore harus menyeberang filesystem.
var ErrCrossDevice = fmt.Errorf("beda filesystem; restore tidak menyalin data")

// EntryID adalah ID stabil untuk UI/CLI: hash(root + NUL + name).
func EntryID(root, name string) string {
	h := sha256.Sum256([]byte(root + "\x00" + name))
	return hex.EncodeToString(h[:])[:12]
}

// HomeTrash mengembalikan direktori trash milik user saat ini.
func HomeTrash() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "Trash")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "Trash")
}

// DiscoverRoots menemukan home trash dan trash volume Linux native yang
// sudah ter-mount. Root yang tidak bisa dibaca tidak menggagalkan discovery;
// List mencatatnya di Report.Errors.
func DiscoverRoots() []Root {
	roots := make([]Root, 0, 4)
	seen := map[string]bool{}

	add := func(r Root) {
		if r.Path == "" {
			return
		}
		r.Path = filepath.Clean(r.Path)
		if seen[r.Path] {
			return
		}
		var st syscall.Stat_t
		if err := syscall.Stat(r.Path, &st); err != nil {
			return
		}
		seen[r.Path] = true
		roots = append(roots, r)
	}

	if home := HomeTrash(); home != "" {
		if p, err := filepath.EvalSymlinks(home); err == nil {
			home = p
		}
		if isRealDir(home) {
			add(Root{Path: home, Kind: RootKindHome})
		}
	}

	uid := strconv.Itoa(os.Getuid())
	for _, m := range mounts() {
		if !nativeFS[m.fstype] || m.readOnly {
			continue
		}
		mp := m.mountPoint
		if mp == "" {
			continue
		}

		// $mnt/.Trash/$uid hanya sah bila .Trash memiliki sticky bit.
		shared := filepath.Join(mp, ".Trash")
		if fi, err := os.Lstat(shared); err == nil && fi.IsDir() && fi.Mode()&os.ModeSticky != 0 {
			p := filepath.Join(shared, uid)
			if isRealDir(p) && isRealDir(filepath.Join(p, "files")) {
				add(Root{Path: p, MountPoint: mp, Kind: RootKindVolume})
			}
		}

		// $mnt/.Trash-$uid
		p := filepath.Join(mp, ".Trash-"+uid)
		if isRealDir(p) && isRealDir(filepath.Join(p, "files")) {
			add(Root{Path: p, MountPoint: mp, Kind: RootKindVolume})
		}
	}

	sort.SliceStable(roots, func(i, j int) bool {
		if roots[i].Kind != roots[j].Kind {
			return roots[i].Kind == RootKindHome
		}
		return roots[i].MountPoint < roots[j].MountPoint
	})
	return roots
}

// validName menolak nama yang bisa keluar dari files/ atau info/.
func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsRune(name, '/') && !strings.ContainsRune(name, 0)
}

// inside melaporkan apakah path berada di dalam root (bukan root itu sendiri).
func inside(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isRealDir(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.IsDir() && fi.Mode()&os.ModeSymlink == 0
}

func isRealFile(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode().IsRegular()
}

func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

type mountEntry struct {
	device     string
	mountPoint string
	fstype     string
	readOnly   bool
}

var nativeFS = map[string]bool{
	"ext2": true, "ext3": true, "ext4": true,
	"btrfs": true, "xfs": true, "f2fs": true,
	"zfs": true, "bcachefs": true,
}

func mounts() []mountEntry {
	b, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return nil
	}
	var out []mountEntry
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		opts := strings.Split(fields[3], ",")
		ro := false
		for _, o := range opts {
			if o == "ro" {
				ro = true
				break
			}
		}
		out = append(out, mountEntry{
			device:     unescapeMount(fields[0]),
			mountPoint: unescapeMount(fields[1]),
			fstype:     fields[2],
			readOnly:   ro,
		})
	}
	return out
}

func unescapeMount(s string) string {
	// /proc/self/mounts memakai escape oktal untuk spasi/tab/newline/backslash.
	r := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return r.Replace(s)
}
