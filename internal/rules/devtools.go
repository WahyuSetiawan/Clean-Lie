package rules

import (
	"path/filepath"
)

// devRules adalah cache tool pengembangan yang benar-benar dipakai mesin ini.
// Semua item hanya muncul kalau path-nya ada; perintah resmi dipakai bila
// binary-nya tersedia.
func devRules() []Rule {
	h := Home()
	p := func(elem ...string) string { return filepath.Join(append([]string{h}, elem...)...) }

	const prio = 20

	return []Rule{
		// ---- Node / npm / pnpm -------------------------------------------------
		{
			ID: "npm-cacache", Group: "dev.node", Label: "_cacache", Path: p(".npm", "_cacache"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache tarball npm, regenerate",
			Command: []string{"npm", "cache", "clean", "--force"},
		},
		{
			ID: "npm-npx", Group: "dev.node", Label: "_npx", Path: p(".npm", "_npx"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache paket npx, regenerate",
		},
		{
			ID: "pnpm-cache", Group: "dev.node", Label: "pnpm cache", Path: p(".cache", "pnpm"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "metadata & tarball pnpm",
			Command: []string{"pnpm", "store", "prune"},
		},
		{
			ID: "pnpm-store", Group: "dev.node", Label: "pnpm store", Path: p(".local", "share", "pnpm", "store"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "store konten pnpm; pakai store prune, jangan rm",
			Command: []string{"pnpm", "store", "prune"},
		},
		{
			ID: "playwright", Group: "dev.node", Path: p(".cache", "ms-playwright"),
			Depth: 1, Risk: RiskWarn, Priority: prio,
			Note: "browser test Playwright; harus diunduh ulang bila dihapus",
		},
		{
			ID: "cypress", Group: "dev.node", Path: p(".cache", "Cypress"),
			Depth: 1, Risk: RiskWarn, Priority: prio,
			Note: "binary Cypress per versi; cek package.json sebelum hapus",
		},

		// ---- JVM / Gradle ------------------------------------------------------
		{
			ID: "gradle-caches", Group: "dev.jvm", Path: p(".gradle", "caches"),
			Depth: 1, Risk: RiskSafe, Priority: prio,
			Note: "cache dependency Gradle per versi; build berikutnya re-download",
		},
		{
			ID: "gradle-wrapper", Group: "dev.jvm", Label: "wrapper dists", Path: p(".gradle", "wrapper", "dists"),
			Depth: 1, Risk: RiskSafe, Priority: prio,
			Note: "distribusi Gradle lama yang diunduh wrapper",
		},
		{
			ID: "maven", Group: "dev.jvm", Label: "m2 repository", Path: p(".m2", "repository"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache Maven lokal, re-download saat build",
		},

		// ---- Python ------------------------------------------------------------
		{
			ID: "uv-cache", Group: "dev.python", Label: "uv cache", Path: p(".cache", "uv"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache wheel/sdist uv",
			Command: []string{"uv", "cache", "prune"},
		},
		{
			ID: "uv-share", Group: "dev.python", Label: "uv data", Path: p(".local", "share", "uv"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache python terkelola uv",
			Command: []string{"uv", "cache", "prune"},
		},
		{
			ID: "pip-cache", Group: "dev.python", Label: "pip cache", Path: p(".cache", "pip"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache wheel/sdist pip",
			Command: []string{"pip", "cache", "purge"},
		},

		// ---- Dart / Flutter ----------------------------------------------------
		{
			ID: "pub-cache", Group: "dev.dart", Label: "pub-cache", Path: p(".pub-cache", "hosted"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "paket pub; diunduh ulang saat pub get",
			Command: []string{"dart", "pub", "cache", "clean"},
		},

		// ---- Rust / Go ---------------------------------------------------------
		{
			ID: "cargo-registry", Group: "dev.rustgo", Label: "cargo registry", Path: p(".cargo", "registry"),
			Depth: 1, Risk: RiskSafe, Priority: prio,
			Match: func(n string) bool { return n == "cache" || n == "src" },
			Note:  "cache crate; bin/ dan config tidak disentuh",
		},
		{
			ID: "go-modcache", Group: "dev.rustgo", Label: "go modcache", Path: p("go", "pkg", "mod"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "modul Go; diunduh ulang saat build",
			Command: []string{"go", "clean", "-modcache"},
		},
		{
			ID: "go-build", Group: "dev.rustgo", Label: "go build cache", Path: p(".cache", "go-build"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache kompilasi Go",
			Command: []string{"go", "clean", "-cache"},
		},
		{
			ID: "gopls", Group: "dev.rustgo", Label: "gopls", Path: p(".cache", "gopls"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache language server gopls",
		},

		// ---- Lain-lain (muncul hanya bila path ada) ----------------------------
		{
			ID: "composer", Group: "dev.node", Label: "composer", Path: p(".cache", "composer"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note:    "cache paket Composer",
			Command: []string{"composer", "clear-cache"},
		},
		{
			ID: "yarn", Group: "dev.node", Label: "yarn", Path: p(".cache", "yarn"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache Yarn",
		},
		{
			ID: "deno", Group: "dev.node", Label: "deno", Path: p(".cache", "deno"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache modul Deno",
		},
		{
			ID: "bun", Group: "dev.node", Label: "bun cache", Path: p(".cache", "bun"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache Bun",
		},
		{
			ID: "prisma", Group: "dev.node", Label: "prisma", Path: p(".cache", "prisma"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "engine Prisma terunduh",
		},
		{
			ID: "typescript", Group: "dev.node", Label: "typescript", Path: p(".cache", "typescript"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "cache TypeScript",
		},
		{
			ID: "esbuild", Group: "dev.node", Label: "esbuild", Path: p(".cache", "esbuild"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "binary esbuild",
		},
		{
			ID: "node-gyp", Group: "dev.node", Label: "node-gyp", Path: p(".cache", "node-gyp"),
			Whole: true, Risk: RiskSafe, Priority: prio,
			Note: "header Node untuk node-gyp",
		},
	}
}
