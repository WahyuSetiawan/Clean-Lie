// Package config menyimpan preferensi Cleanlie di config.json. Semua nilai
// divalidasi ulang saat dibaca supaya file yang rusak tidak membuat app gagal.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"

	LanguageID = "id"
	LanguageEN = "en"

	DeleteTrash     = "trash"
	DeletePermanent = "permanent"
)

// Config adalah isi ~/.config/cleanlie/config.json.
type Config struct {
	Theme          string       `json:"theme"`
	Language       string       `json:"language"`
	ShowRisky      bool         `json:"showRisky"`
	DeleteMode     string       `json:"deleteMode"`
	AutoSelectSafe bool         `json:"autoSelectSafe"`
	Notify         bool         `json:"notify"`
	JournalTarget  string       `json:"journalTarget"`
	LargeBytes     int64        `json:"largeBytes"`
	StaleDays      int          `json:"staleDays"`
	ArtifactRoots  []string     `json:"artifactRoots"`
	DirenvRoots    []string     `json:"direnvRoots"`
	Excluded       []string     `json:"excluded"`
	CustomRules    []CustomRule `json:"customRules,omitempty"`
}

// CustomRule adalah aturan pemindaian tambahan dari pengguna.
type CustomRule struct {
	Path  string `json:"path"`
	Label string `json:"label,omitempty"`
	Risk  string `json:"risk,omitempty"`
	Whole bool   `json:"whole,omitempty"`
	Depth int    `json:"depth,omitempty"`
}

// Default mengembalikan preferensi awal yang aman.
func Default() Config {
	return Config{
		Theme:          ThemeSystem,
		Language:       LanguageID,
		ShowRisky:      true,
		DeleteMode:     DeleteTrash,
		AutoSelectSafe: true,
		Notify:         true,
		JournalTarget:  "500M",
		LargeBytes:     100 << 20,
		StaleDays:      365,
	}
}

// Path mengembalikan lokasi config sesuai XDG.
func Path() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "cleanlie", "config.json")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "cleanlie", "config.json")
	}
	return "cleanlie/config.json"
}

// Normalize memperbaiki nilai yang tidak dikenal dan mengembalikan peringatan
// yang bisa ditampilkan di UI. Fungsi ini tidak menulis file.
func Normalize(c Config) (Config, []string) {
	d := Default()
	var warnings []string

	switch c.Theme {
	case ThemeSystem, ThemeLight, ThemeDark:
	case "":
		c.Theme = d.Theme
	default:
		warnings = append(warnings, "tema tidak dikenal: "+c.Theme)
		c.Theme = d.Theme
	}
	switch c.Language {
	case LanguageID, LanguageEN:
	case "":
		c.Language = d.Language
	default:
		warnings = append(warnings, "bahasa tidak dikenal: "+c.Language)
		c.Language = d.Language
	}
	switch c.DeleteMode {
	case DeleteTrash, DeletePermanent:
	case "":
		c.DeleteMode = d.DeleteMode
	default:
		warnings = append(warnings, "mode hapus tidak dikenal: "+c.DeleteMode)
		c.DeleteMode = d.DeleteMode
	}
	if c.JournalTarget == "" {
		c.JournalTarget = d.JournalTarget
	}
	if c.LargeBytes <= 0 {
		c.LargeBytes = d.LargeBytes
	}
	if c.LargeBytes < 1<<20 {
		warnings = append(warnings, "ambang file besar terlalu kecil; memakai 100 MB")
		c.LargeBytes = d.LargeBytes
	}
	if c.LargeBytes > 100<<30 {
		warnings = append(warnings, "ambang file besar terlalu besar; memakai 100 MB")
		c.LargeBytes = d.LargeBytes
	}
	if c.StaleDays <= 0 {
		c.StaleDays = d.StaleDays
	}
	if c.StaleDays > 3650 {
		warnings = append(warnings, "ambang file lama dibatasi 3650 hari")
		c.StaleDays = 3650
	}
	c.ArtifactRoots = cleanPaths(c.ArtifactRoots)
	c.DirenvRoots = cleanPaths(c.DirenvRoots)
	c.Excluded = cleanPaths(c.Excluded)
	c.CustomRules, warnings = normalizeCustomRules(c.CustomRules, warnings)
	return c, warnings
}

// Load membaca config. Bila file tidak ada atau rusak, default dipakai dan
// warning dikembalikan agar UI bisa memberi tahu pengguna.
func Load() (Config, string) {
	path := Path()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		c, _ := Normalize(Default())
		return c, ""
	}
	if err != nil {
		c, _ := Normalize(Default())
		return c, "config tidak bisa dibaca: " + err.Error()
	}
	c := Default()
	if err := json.Unmarshal(data, &c); err != nil {
		c, _ = Normalize(Default())
		return c, "config tidak valid, memakai default: " + err.Error()
	}
	c, warnings := Normalize(c)
	if len(warnings) > 0 {
		return c, strings.Join(warnings, "; ")
	}
	return c, ""
}

// Save menulis config secara atomik. Nilai dinormalisasi sebelum disimpan.
func Save(c Config) (Config, []string, error) {
	c, warnings := Normalize(c)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return c, warnings, err
	}
	data = append(data, '\n')
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return c, warnings, fmt.Errorf("buat folder config: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return c, warnings, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return c, warnings, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return c, warnings, err
	}
	if err := tmp.Close(); err != nil {
		return c, warnings, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return c, warnings, err
	}
	return c, warnings, nil
}

func cleanPaths(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = expandHome(p)
		if !filepath.IsAbs(p) {
			continue
		}
		p = filepath.Clean(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func normalizeCustomRules(in []CustomRule, warnings []string) ([]CustomRule, []string) {
	const maxCustom = 32
	seen := map[string]bool{}
	out := make([]CustomRule, 0, len(in))
	for i, r := range in {
		if len(out) >= maxCustom {
			warnings = append(warnings, fmt.Sprintf("aturan kustom dibatasi %d; sisanya diabaikan", maxCustom))
			break
		}
		p := strings.TrimSpace(r.Path)
		if p == "" {
			continue
		}
		p = expandHome(p)
		if !filepath.IsAbs(p) {
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: path harus absolut: %s", i+1, r.Path))
			continue
		}
		p = filepath.Clean(p)
		if rules.IsProtected(p) {
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: path dilindungi: %s", i+1, p))
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		risk := strings.ToLower(strings.TrimSpace(r.Risk))
		switch risk {
		case "", "warn":
			risk = "warn"
		case "safe", "danger":
		default:
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: risk tidak dikenal: %s", i+1, r.Risk))
			risk = "warn"
		}
		depth := r.Depth
		if depth <= 0 {
			depth = 1
		}
		if depth > 6 {
			depth = 6
		}
		label := strings.TrimSpace(r.Label)
		if label == "" {
			label = filepath.Base(p)
		}
		whole := r.Whole
		if whole {
			depth = 0
		}
		out = append(out, CustomRule{Path: p, Label: label, Risk: risk, Whole: whole, Depth: depth})
	}
	return out, warnings
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
