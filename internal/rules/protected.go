package rules

import (
	"path/filepath"
	"runtime"
	"strings"
)

// ProtectedPaths adalah daftar path yang tidak boleh masuk katalog clean.
// Dipakai juga oleh modul clean sebagai pertahanan terakhir.
func ProtectedPaths() []string {
	h := Home()
	exact := []string{
		"/", "/nix", NixStore, "/nix/var",
		"/etc", "/usr", "/bin", "/sbin", "/lib", "/lib64", "/boot",
		"/dev", "/proc", "/sys", "/run",
		"/var", "/var/lib", "/var/db", "/var/log",
		"/home", h,
		filepath.Join(h, "Documents"),
		filepath.Join(h, "Downloads"),
		filepath.Join(h, "Pictures"),
		filepath.Join(h, "Videos"),
		filepath.Join(h, "Music"),
		filepath.Join(h, "Desktop"),
		filepath.Join(h, "Projects"),
		filepath.Join(h, ".config"),
		filepath.Join(h, ".local", "share"),
		filepath.Join(h, ".local", "state"),
	}
	if runtime.GOOS == "darwin" {
		exact = append(exact,
			"/System", "/Library", "/Applications", "/private", "/Volumes", "/cores",
			"/opt", "/usr/local",
			filepath.Join(h, "Library"),
			filepath.Join(h, "Movies"),
			filepath.Join(h, "Public"),
		)
	}
	return exact
}

// ProtectedPrefixes adalah awalan path yang tidak boleh disentuh.
func ProtectedPrefixes() []string {
	h := Home()
	prefixes := []string{
		"/nix/store/", "/nix/var/",
		"/etc/", "/usr/", "/bin/", "/sbin/", "/lib/", "/lib64/", "/boot/",
		"/dev/", "/proc/", "/sys/", "/run/",
		"/var/lib/",
		filepath.Join(h, ".ssh") + "/",
		filepath.Join(h, ".gnupg") + "/",
		filepath.Join(h, ".pki") + "/",
		filepath.Join(h, ".password-store") + "/",
		filepath.Join(h, ".local", "share", "keyrings") + "/",
	}
	if runtime.GOOS == "darwin" {
		// ~/Library/Caches dan ~/Library/Logs sengaja tidak diprefix, karena
		// justru itu yang dibersihkan; data aplikasi tetap dilindungi.
		prefixes = append(prefixes,
			"/System/", "/Library/", "/Applications/", "/private/", "/Volumes/",
			"/opt/homebrew/", "/usr/local/",
			filepath.Join(h, "Library", "Application Support")+"/",
			filepath.Join(h, "Library", "Containers")+"/",
			filepath.Join(h, "Library", "Group Containers")+"/",
			filepath.Join(h, "Library", "Keychains")+"/",
			filepath.Join(h, "Library", "Mobile Documents")+"/",
			filepath.Join(h, "Library", "Preferences")+"/",
			filepath.Join(h, "Library", "Mail")+"/",
			filepath.Join(h, "Library", "Messages")+"/",
		)
	}
	return prefixes
}

// IsProtected melaporkan apakah path tidak boleh dihapus.
func IsProtected(p string) bool {
	if p == "" {
		return true
	}
	clean := filepath.Clean(ExpandHome(p))
	for _, e := range ProtectedPaths() {
		if clean == cleanPath(e) {
			return true
		}
	}
	for _, pre := range ProtectedPrefixes() {
		if strings.HasPrefix(clean, cleanPath(pre)) {
			return true
		}
	}
	// Jangan izinkan root atau satu level di bawah root (mis. /homebar).
	if clean == "/" || strings.Count(clean, string(filepath.Separator)) < 2 {
		return true
	}
	return false
}

func cleanPath(p string) string {
	return filepath.Clean(ExpandHome(p))
}
