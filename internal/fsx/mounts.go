package fsx

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
)

// Mount adalah volume lokal yang bisa ditampilkan di dashboard.
type Mount struct {
	Path   string `json:"path"`
	Fstype string `json:"fstype"`
	Device string `json:"device"`
	Volume
}

type mountEntry struct {
	device string
	path   string
	fstype string
}

// Mounts mengembalikan filesystem lokal yang layak ditampilkan: hanya tipe
// disk nyata, tanpa pseudo-fs/overlay, dan dideduplikasi berdasarkan device.
// Root "/" selalu di urutan pertama bila tersedia.
func Mounts() ([]Mount, error) {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries := parseMounts(f)

	seen := make(map[uint64]bool)
	out := make([]Mount, 0, len(entries))
	for _, e := range entries {
		if !localFS(e.fstype) {
			continue
		}
		vol, err := Statfs(e.path)
		if err != nil || vol.Total <= 0 {
			continue
		}
		var st syscall.Stat_t
		if err := syscall.Stat(e.path, &st); err == nil {
			dev := uint64(st.Dev)
			if seen[dev] {
				continue
			}
			seen[dev] = true
		}
		out = append(out, Mount{Path: e.path, Fstype: e.fstype, Device: e.device, Volume: vol})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path == "/" {
			return true
		}
		if out[j].Path == "/" {
			return false
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func parseMounts(r io.Reader) []mountEntry {
	var out []mountEntry
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		out = append(out, mountEntry{
			device: unescapeMount(fields[0]),
			path:   unescapeMount(fields[1]),
			fstype: fields[2],
		})
	}
	return out
}

// unescapeMount mengubah escape octal dari /proc/mounts (\040 spasi, \011 tab,
// \134 backslash, \012 newline).
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			b.WriteByte((s[i+1]-'0')<<6 | (s[i+2]-'0')<<3 | (s[i+3] - '0'))
			i += 3
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isOctal(b byte) bool { return b >= '0' && b <= '7' }

func localFS(fstype string) bool {
	switch fstype {
	case "ext2", "ext3", "ext4", "btrfs", "xfs", "f2fs", "vfat", "exfat",
		"ntfs", "ntfs3", "fuseblk", "zfs", "bcachefs":
		return true
	default:
		return false
	}
}
