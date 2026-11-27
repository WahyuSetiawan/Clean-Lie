package rules

import (
	"path/filepath"
	"runtime"
)

// platformRules memilih katalog tambahan sesuai OS. Linux memakai rule log
// sistem (journald, /var/log, /var/tmp); macOS memakai ~/Library.
func platformRules() []Rule {
	if runtime.GOOS == "darwin" {
		return macRules()
	}
	return logRules()
}

// macRules adalah sampah khas macOS: cache aplikasi di ~/Library/Caches,
// log, state jendela, plus artefak Xcode/CoreSimulator dan tool developer
// yang menyimpan cache di ~/Library (bukan ~/.cache seperti di Linux).
func macRules() []Rule {
	h := Home()
	p := func(elem ...string) string { return filepath.Join(append([]string{h}, elem...)...) }

	const prio = 18

	return []Rule{
		// ---- Cache & log aplikasi -----------------------------------------
		{
			ID: "mac-caches", Group: "app", Label: "~/Library/Caches", Path: p("Library", "Caches"),
			Depth: 1, Risk: RiskSafe, Priority: prio,
			Note: "cache aplikasi macOS, dibangun ulang saat dipakai",
		},
		{
			ID: "mac-logs", Group: "sys.log", Label: "~/Library/Logs", Path: p("Library", "Logs"),
			Depth: 1, Risk: RiskSafe, Priority: prio,
			Note: "log aplikasi user (bukan log sistem)",
		},
		{
			ID: "mac-saved-state", Group: "app", Label: "Saved Application State", Path: p("Library", "Saved Application State"),
			Depth: 1, Risk: RiskSafe, Priority: 12,
			Note: "state jendela aplikasi; dibuat ulang saat app dibuka",
		},

		// ---- Tool developer yang pakai ~/Library ---------------------------
		{
			ID: "mac-go-build", Group: "dev.rustgo", Label: "go build cache (macOS)", Path: p("Library", "Caches", "go-build"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note:    "cache kompilasi Go di macOS",
			Command: []string{"go", "clean", "-cache"},
		},
		{
			ID: "mac-gopls", Group: "dev.rustgo", Label: "gopls (macOS)", Path: p("Library", "Caches", "gopls"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note: "cache language server gopls",
		},
		{
			ID: "mac-uv", Group: "dev.python", Label: "uv cache (macOS)", Path: p("Library", "Caches", "uv"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note:    "cache wheel/sdist uv di macOS",
			Command: []string{"uv", "cache", "prune"},
		},
		{
			ID: "mac-pip", Group: "dev.python", Label: "pip cache (macOS)", Path: p("Library", "Caches", "pip"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note:    "cache wheel/sdist pip di macOS",
			Command: []string{"pip", "cache", "purge"},
		},
		{
			ID: "mac-pnpm-store", Group: "dev.node", Label: "pnpm store (macOS)", Path: p("Library", "pnpm", "store"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note:    "store konten pnpm di macOS; pakai store prune, jangan rm",
			Command: []string{"pnpm", "store", "prune"},
		},
		{
			ID: "mac-playwright", Group: "dev.node", Label: "Playwright (macOS)", Path: p("Library", "Caches", "ms-playwright"),
			Depth: 1, Risk: RiskWarn, Priority: 22,
			Note: "browser test Playwright; harus diunduh ulang bila dihapus",
		},
		{
			ID: "mac-cypress", Group: "dev.node", Label: "Cypress (macOS)", Path: p("Library", "Caches", "Cypress"),
			Depth: 1, Risk: RiskWarn, Priority: 22,
			Note: "binary Cypress per versi; cek package.json sebelum hapus",
		},
		{
			ID: "mac-homebrew", Group: "app", Label: "Homebrew cache", Path: p("Library", "Caches", "Homebrew"),
			Whole: true, Risk: RiskSafe, Priority: 22,
			Note:    "unduhan botol Homebrew; dihapus lewat brew cleanup",
			Command: []string{"brew", "cleanup", "-s"},
		},

		// ---- Xcode & simulator ---------------------------------------------
		{
			ID: "xcode-derived", Group: "dev.macos", Label: "Xcode DerivedData", Path: p("Library", "Developer", "Xcode", "DerivedData"),
			Whole: true, Risk: RiskSafe, Priority: 24,
			Note: "index & build product Xcode; dibuat ulang saat build",
		},
		{
			ID: "xcode-devicesupport", Group: "dev.macos", Label: "iOS DeviceSupport", Path: p("Library", "Developer", "Xcode", "iOS DeviceSupport"),
			Depth: 1, Risk: RiskSafe, Priority: 24,
			Note: "symbol device per versi iOS; diunduh ulang saat device dicolok",
		},
		{
			ID: "xcode-archives", Group: "dev.macos", Label: "Xcode Archives", Path: p("Library", "Developer", "Xcode", "Archives"),
			Depth: 1, Risk: RiskWarn, Priority: 24,
			Note: "arsip build untuk submit App Store; simpan yang masih dipakai",
		},
		{
			ID: "coresim-caches", Group: "dev.macos", Label: "CoreSimulator caches", Path: p("Library", "Developer", "CoreSimulator", "Caches"),
			Whole: true, Risk: RiskSafe, Priority: 24,
			Note: "cache runtime simulator",
		},
		{
			ID: "coresim-devices", Group: "dev.macos", Label: "Simulator", Path: p("Library", "Developer", "CoreSimulator", "Devices"),
			Depth: 1, Risk: RiskWarn, Priority: 24,
			Note: "isi simulator hilang; bisa dibuat ulang lewat Xcode",
		},
	}
}
