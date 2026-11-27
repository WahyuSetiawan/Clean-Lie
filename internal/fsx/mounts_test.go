package fsx

import (
	"strings"
	"testing"
)

func TestParseMounts(t *testing.T) {
	in := `/dev/nvme0n1p2 / ext4 rw,relatime 0 0
/dev/nvme0n1p1 /boot vfat rw 0 0
tmpfs /run tmpfs rw 0 0
/dev/sda1 /mnt/My\040Disk ext4 rw 0 0
`
	got := parseMounts(strings.NewReader(in))
	if len(got) != 4 {
		t.Fatalf("parse = %d entri, mau 4", len(got))
	}
	if got[3].path != "/mnt/My Disk" {
		t.Fatalf("unescape path = %q", got[3].path)
	}
}

func TestLocalFS(t *testing.T) {
	for _, fs := range []string{"ext4", "btrfs", "xfs", "f2fs", "vfat", "zfs", "bcachefs"} {
		if !localFS(fs) {
			t.Errorf("%s harus lokal", fs)
		}
	}
	for _, fs := range []string{"tmpfs", "proc", "sysfs", "overlay", "squashfs", "devtmpfs"} {
		if localFS(fs) {
			t.Errorf("%s bukan volume lokal", fs)
		}
	}
}

func TestMountsIncludesRoot(t *testing.T) {
	ms, err := Mounts()
	if err != nil {
		t.Skipf("mounts tidak tersedia: %v", err)
	}
	if len(ms) == 0 {
		t.Skip("tidak ada volume lokal")
	}
	if ms[0].Path != "/" {
		t.Fatalf("volume pertama = %q, mau /", ms[0].Path)
	}
	if ms[0].Health == "" {
		t.Fatal("health root kosong")
	}
}
