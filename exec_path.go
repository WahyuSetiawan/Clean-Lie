package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// selfExecutable mengembalikan path yang dipakai pengguna (wrapper Nix), bukan
// target wrapper di dalam store. os.Executable() bisa menunjuk
// .cleanlie-wrapped sehingga PATH dari makeWrapper hilang saat timer berjalan.
func selfExecutable() (string, error) {
	if len(os.Args) > 0 && os.Args[0] != "" {
		p := os.Args[0]
		if filepath.IsAbs(p) {
			if _, err := os.Stat(p); err == nil {
				return resolveSelf(p), nil
			}
		}
		if found, err := exec.LookPath(p); err == nil {
			if abs, err := filepath.Abs(found); err == nil {
				return resolveSelf(abs), nil
			}
			return resolveSelf(found), nil
		}
	}
	return os.Executable()
}

// resolveSelf mengikuti symlink satu level (result/bin/cleanlie) ke wrapper di
// store, tetapi tidak menunjuk target .cleanlie-wrapped.
func resolveSelf(p string) string {
	p = filepath.Clean(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil && resolved != "" {
		return resolved
	}
	return p
}
