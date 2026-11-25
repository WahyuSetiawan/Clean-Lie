package rules

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// DefaultRules mengembalikan seluruh katalog v1.
func DefaultRules() []Rule {
	out := make([]Rule, 0, 64)
	out = append(out, devRules()...)
	out = append(out, androidRules()...)
	out = append(out, appRules()...)
	out = append(out, logRules()...)
	out = append(out, trashRules()...)
	return out
}

func appRules() []Rule {
	h := Home()
	p := func(elem ...string) string { return filepath.Join(append([]string{h}, elem...)...) }

	// Cache aplikasi yang terukur di mesin ini.
	type entry struct {
		id, label, rel, note string
		risk                 Risk
	}
	entries := []entry{
		{"opencode", "opencode", ".cache/opencode", "session & model cache, regenerate", RiskSafe},
		{"qmd", "qmd", ".cache/qmd", "index embedding, regenerate", RiskSafe},
		{"phpactor", "phpactor", ".cache/phpactor", "index PHP, regenerate", RiskSafe},
		{"codebase-memory", "codebase-memory-mcp", ".cache/codebase-memory-mcp", "index kode, regenerate", RiskSafe},
		{"nix-cache", "nix eval cache", ".cache/nix", "eval cache Nix, regenerate", RiskSafe},
		{"appimage", "appimage-run", ".cache/appimage-run", "cache AppImage", RiskSafe},
		{"thumbnails", "thumbnailer", ".cache/Thumbnailer", "thumbnail file manager", RiskSafe},
		{"thumbnails-lower", "thumbnails", ".cache/thumbnails", "thumbnail file manager", RiskSafe},
		{"fontconfig", "fontconfig", ".cache/fontconfig", "cache font, dibangun ulang", RiskSafe},
		{"mesa", "mesa shader cache", ".cache/mesa_shader_cache", "shader GPU, dibangun ulang", RiskSafe},
		{"jetbrains", "JetBrains", ".cache/JetBrains", "index IDE, dibangun ulang", RiskSafe},
		{"brave", "Brave", ".cache/BraveSoftware", "skip kalau Brave sedang berjalan", RiskWarn},
		{"mozilla", "Firefox", ".cache/mozilla", "skip kalau Firefox sedang berjalan", RiskWarn},
		{"chrome", "Chrome", ".cache/google-chrome", "skip kalau Chrome sedang berjalan", RiskWarn},
		{"chromium", "Chromium", ".cache/chromium", "skip kalau Chromium sedang berjalan", RiskWarn},
		{"edge", "Edge", ".cache/microsoft-edge", "skip kalau Edge sedang berjalan", RiskWarn},
	}

	rules := make([]Rule, 0, len(entries)+1)
	for _, e := range entries {
		rules = append(rules, Rule{
			ID: "app-" + e.id, Group: "app", Label: e.label, Path: p(filepath.FromSlash(e.rel)),
			Whole: true, Risk: e.risk, Note: e.note, Priority: 15,
		})
	}

	// Sisa cache aplikasi: anak langsung ~/.cache yang tidak tercakup rule lain.
	rules = append(rules, Rule{
		ID: "app-cache-other", Group: "app", Path: p(".cache"),
		Depth: 1, Risk: RiskSafe, Priority: -10,
		Note: "cache aplikasi lain, regenerate",
	})
	return rules
}

func logRules() []Rule {
	return []Rule{
		{
			ID: "journal", Group: "sys.log", Label: "systemd journal", Path: "/var/log/journal",
			Whole: true, Risk: RiskWarn, Priority: 30, RequiresRoot: true,
			Note:    "vacuum journal ke target (butuh root)",
			Command: []string{"journalctl", "--vacuum-size=500M"},
			Probe:   probeJournal,
		},
		{
			ID: "log-rotated", Group: "sys.log", Label: "log rotasi", Path: "/var/log",
			Depth: 1, IncludeFiles: true, Risk: RiskSafe, Priority: 5,
			Match: func(n string) bool {
				return strings.HasSuffix(n, ".gz") || strings.HasSuffix(n, ".xz") ||
					strings.HasSuffix(n, ".zst") || strings.HasSuffix(n, ".old")
			},
			Note: "file log yang sudah dirotasi",
		},
		{
			ID: "var-tmp", Group: "sys.log", Label: "/var/tmp", Path: "/var/tmp",
			Depth: 1, IncludeFiles: true, Risk: RiskSafe, Priority: 5,
			Note: "file sementara sistem",
		},
	}
}

func trashRules() []Rule {
	return []Rule{{
		ID: "trash", Group: "trash", Label: "isi Trash", Path: filepath.Join(Home(), ".local", "share", "Trash", "files"),
		Depth: 1, IncludeFiles: true, Risk: RiskWarn, Priority: 25,
		Note: "setelah dikosongkan tidak bisa di-restore",
	}}
}

var journalRe = regexp.MustCompile(`(?i)([0-9]+(?:[.,][0-9]+)?)\s*([KMGTP])(?:i?B)?`)

// parseJournalUsage membaca output `journalctl --disk-usage`:
//
//	"Archived and active journals take up 4.0G in the file system."
func parseJournalUsage(out string) int64 {
	m := journalRe.FindStringSubmatch(out)
	if m == nil {
		return 0
	}
	val, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0
	}
	mult := map[string]float64{"K": 1 << 10, "M": 1 << 20, "G": 1 << 30, "T": 1 << 40, "P": 1 << 50}
	return int64(val * mult[strings.ToUpper(m[2])])
}

func probeJournal(ctx context.Context) (int64, int64, error) {
	if tools.Available("journalctl") {
		res := tools.Run(ctx, []string{"journalctl", "--disk-usage"}, 5*time.Second)
		if res.Err == nil {
			if n := parseJournalUsage(res.Stdout); n > 0 {
				return n, 0, nil
			}
		}
	}
	// fallback: hitung isi direktori (mungkin butuh izin baca).
	size, files, err := fsx.Usage(ctx, "/var/log/journal")
	if err != nil {
		return 0, 0, err
	}
	return size, files, nil
}
