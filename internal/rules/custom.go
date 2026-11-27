package rules

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CustomRule adalah aturan yang ditambahkan pengguna lewat config. Path
// divalidasi terhadap daftar protected sebelum dipakai.
type CustomRule struct {
	Path  string `json:"path"`
	Label string `json:"label,omitempty"`
	Risk  Risk   `json:"risk,omitempty"`
	Whole bool   `json:"whole,omitempty"`
	Depth int    `json:"depth,omitempty"`
}

// CustomRules mengubah aturan config menjadi Rule katalog. Aturan tidak valid
// dilewati dan dikembalikan sebagai warning.
func CustomRules(list []CustomRule) ([]Rule, []string) {
	out := make([]Rule, 0, len(list))
	var warnings []string
	seen := map[string]bool{}
	for i, c := range list {
		path := filepath.Clean(ExpandHome(strings.TrimSpace(c.Path)))
		if path == "" || path == "." {
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: path kosong", i+1))
			continue
		}
		if !filepath.IsAbs(path) {
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: path harus absolut: %s", i+1, c.Path))
			continue
		}
		if IsProtected(path) {
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: path dilindungi: %s", i+1, path))
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true

		risk := c.Risk
		switch risk {
		case "", RiskWarn:
			risk = RiskWarn
		case RiskSafe, RiskDanger:
		default:
			warnings = append(warnings, fmt.Sprintf("aturan kustom #%d: risk tidak dikenal: %s", i+1, risk))
			risk = RiskWarn
		}
		label := strings.TrimSpace(c.Label)
		if label == "" {
			label = filepath.Base(path)
		}
		depth := c.Depth
		if depth <= 0 {
			depth = 1
		}
		if depth > 6 {
			depth = 6
		}
		rule := Rule{
			ID:       fmt.Sprintf("custom-%d-%s", i+1, slug(label)),
			Group:    "custom",
			Label:    label,
			Path:     path,
			Risk:     risk,
			Priority: 10,
			Note:     "aturan kustom",
		}
		if c.Whole {
			rule.Whole = true
		} else {
			rule.Depth = depth
			rule.IncludeFiles = true
		}
		out = append(out, rule)
	}
	return out, warnings
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "rule"
	}
	return out
}
