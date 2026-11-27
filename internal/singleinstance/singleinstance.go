// Package singleinstance mencegah dua GUI Cleanlie berjalan bersamaan.
// Hanya GUI yang memakai lock; mode CLI tetap boleh berjalan saat GUI terbuka.
package singleinstance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ErrRunning dikembalikan bila GUI lain sudah memegang lock.
var ErrRunning = errors.New("Cleanlie sudah berjalan")

// Lock adalah file lock yang dipegang selama proses hidup.
type Lock struct {
	f    *os.File
	path string
}

// Dir mengembalikan folder state Cleanlie.
func Dir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "cleanlie")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "state", "cleanlie")
	}
	return filepath.Join("cleanlie")
}

// Path mengembalikan lokasi file lock.
func Path() string {
	return filepath.Join(Dir(), "cleanlie.lock")
}

// Acquire mencoba mengambil lock non-blocking. File lock tidak dihapus saat
// Release supaya inode tidak berubah di tengah perlombaan.
func Acquire() (*Lock, error) {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrRunning
		}
		return nil, err
	}
	if err := f.Truncate(0); err == nil {
		_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
		_ = f.Sync()
	}
	return &Lock{f: f, path: path}, nil
}

// Release melepas lock dan menutup file.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	cerr := l.f.Close()
	l.f = nil
	if err != nil {
		return err
	}
	return cerr
}

// Path mengembalikan path lock yang dipegang.
func (l *Lock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}
