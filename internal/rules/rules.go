// Package rules mendefinisikan katalog item yang bisa dibersihkan dan
// merencanakan pemindaian (read-only). Paket ini tidak pernah menghapus file.
package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// Risk adalah tingkat bahaya sebuah item.
type Risk string

const (
	RiskSafe    Risk = "safe"
	RiskWarn    Risk = "warn"
	RiskDanger  Risk = "danger"
	RiskBlocked Risk = "blocked"
)

func riskRank(r Risk) int {
	switch r {
	case RiskBlocked:
		return 4
	case RiskDanger:
		return 3
	case RiskWarn:
		return 2
	default:
		return 1
	}
}

// Rule mendeskripsikan satu sumber item yang bisa dibersihkan.
type Rule struct {
	ID    string
	Group string
	Label string
	Path  string
	Whole bool // seluruh root = satu item
	Depth int  // kedalaman item bila !Whole (default 1)
	// IncludeFiles menyertakan file (bukan hanya direktori) pada depth terakhir.
	IncludeFiles bool
	Match        func(name string) bool
	Risk         Risk
	Note         string
	NoteFunc     func(path string) string
	Command      []string
	RequiresRoot bool
	Priority     int
	// Probe menghitung ukuran tanpa walk (mis. journalctl --disk-usage).
	Probe func(ctx context.Context) (bytes, files int64, err error)
}

// Item adalah satu baris di UI / laporan.
type Item struct {
	ID           string   `json:"id"`
	Group        string   `json:"group"`
	Path         string   `json:"path"`
	Label        string   `json:"label"`
	Bytes        int64    `json:"bytes"`
	Files        int64    `json:"files"`
	Risk         Risk     `json:"risk"`
	Note         string   `json:"note,omitempty"`
	Command      []string `json:"command,omitempty"`
	CommandLine  string   `json:"commandLine,omitempty"`
	RequiresRoot bool     `json:"requiresRoot,omitempty"`
	FallbackOK   bool     `json:"fallbackDelete"`
}

// Group mengelompokkan item untuk tampilan.
type Group struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Risk  Risk   `json:"risk"`
	Bytes int64  `json:"bytes"`
	Files int64  `json:"files"`
	Items []Item `json:"items"`
}

// ScanError mencatat path yang gagal diukur.
type ScanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// Report adalah hasil Plan.
type Report struct {
	ScannedAt string      `json:"scannedAt"`
	Duration  int64       `json:"durationMs"`
	Bytes     int64       `json:"bytes"`
	Files     int64       `json:"files"`
	Groups    []Group     `json:"groups"`
	Errors    []ScanError `json:"errors,omitempty"`
}

// PlanOptions mengatur pemindaian.
type PlanOptions struct {
	Workers  int
	Progress func(Progress)
}

// Progress adalah progres per item.
type Progress struct {
	Done  int
	Total int
	Path  string
}

// GroupOrder menentukan urutan grup di UI.
var GroupOrder = []string{
	"dev.node", "dev.jvm", "dev.python", "dev.dart", "dev.rustgo", "dev.android",
	"app", "sys.log", "trash",
}

var groupLabels = map[string]string{
	"dev.node":    "Node (npm/pnpm)",
	"dev.jvm":     "JVM (Gradle)",
	"dev.python":  "Python (uv/pip)",
	"dev.dart":    "Dart / Flutter",
	"dev.rustgo":  "Rust / Go",
	"dev.android": "Android SDK/AVD",
	"app":         "Cache aplikasi",
	"sys.log":     "Log & journal",
	"trash":       "Trash",
	"scan":        "Hasil scan",
}

// GroupLabel mengembalikan label tampilan untuk sebuah group id.
func GroupLabel(id string) string {
	if l, ok := groupLabels[id]; ok {
		return l
	}
	if id == "" {
		return "Lainnya"
	}
	return id
}

// Home mengembalikan direktori home saat ini ($HOME), atau "" bila tidak ada.
func Home() string {
	return os.Getenv("HOME")
}

// ExpandHome mengganti awalan "~" dengan $HOME.
func ExpandHome(p string) string {
	if p == "~" {
		return Home()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(Home(), p[2:])
	}
	return p
}

type candidate struct {
	rule  Rule
	path  string
	label string
}

// Plan memindai semua rule dan mengembalikan laporan read-only.
func Plan(ctx context.Context, list []Rule, opts PlanOptions) (Report, error) {
	start := time.Now()
	if opts.Workers <= 0 {
		opts.Workers = fsx.DefaultWorkers()
	}

	// Prioritas tinggi diproses lebih dulu; path duplikat mempertahankan yang pertama.
	sorted := make([]Rule, 0, len(list))
	for _, r := range list {
		if r.ID == "" || r.Path == "" || r.Group == "" {
			continue
		}
		sorted = append(sorted, r)
	}
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Priority > sorted[j].Priority })

	var candidates []candidate
	seen := map[string]bool{}

	for _, r := range sorted {
		root := filepath.Clean(ExpandHome(r.Path))

		if r.Probe == nil {
			if _, err := os.Lstat(root); err != nil {
				continue
			}
		}

		if r.Whole || r.Probe != nil {
			label := r.Label
			if label == "" {
				label = filepath.Base(root)
			}
			if !seen[root] {
				seen[root] = true
				candidates = append(candidates, candidate{rule: r, path: root, label: label})
			}
			continue
		}

		depth := r.Depth
		if depth <= 0 {
			depth = 1
		}
		for _, p := range enumerate(root, depth, r.IncludeFiles, r.Match) {
			if seen[p] {
				continue
			}
			seen[p] = true
			candidates = append(candidates, candidate{rule: r, path: p, label: filepath.Base(p)})
		}
	}

	// Buang item yang merupakan induk dari item lain, supaya tidak dihitung dua kali.
	candidates = dropAncestors(candidates)

	type measured struct {
		bytes int64
		files int64
		err   error
	}
	results := make([]measured, len(candidates))
	total := len(candidates)
	var done int
	var mu sync.Mutex

	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := opts.Workers
	if workers > total {
		workers = total
	}
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				c := candidates[idx]
				var m measured
				if c.rule.Probe != nil {
					m.bytes, m.files, m.err = c.rule.Probe(ctx)
				} else {
					m.bytes, m.files, m.err = fsx.Usage(ctx, c.path)
				}
				results[idx] = m

				mu.Lock()
				done++
				if opts.Progress != nil {
					opts.Progress(Progress{Done: done, Total: total, Path: c.path})
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for i := range candidates {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}

	report := Report{
		ScannedAt: time.Now().Format(time.RFC3339),
		Duration:  time.Since(start).Milliseconds(),
	}
	groups := map[string]*Group{}

	for i, c := range candidates {
		m := results[i]
		if m.err != nil {
			report.Errors = append(report.Errors, ScanError{Path: c.path, Error: m.err.Error()})
			continue
		}
		if m.bytes <= 0 {
			continue
		}
		item := Item{
			ID:           c.rule.ID + "/" + c.label,
			Group:        c.rule.Group,
			Path:         c.path,
			Label:        c.label,
			Bytes:        m.bytes,
			Files:        m.files,
			Risk:         c.rule.Risk,
			Note:         c.rule.Note,
			FallbackOK:   c.rule.Probe == nil,
			RequiresRoot: c.rule.RequiresRoot,
		}
		if item.Risk == "" {
			item.Risk = RiskSafe
		}
		if c.rule.NoteFunc != nil {
			item.Note = c.rule.NoteFunc(c.path)
		}
		if len(c.rule.Command) > 0 && tools.Available(c.rule.Command[0]) {
			item.Command = c.rule.Command
			item.CommandLine = tools.Preview(c.rule.Command)
		} else if len(c.rule.Command) > 0 {
			if item.Note != "" {
				item.Note += " · "
			}
			item.Note += "tool tidak ada di PATH, hapus folder manual"
		}

		g := groups[item.Group]
		if g == nil {
			g = &Group{ID: item.Group, Label: GroupLabel(item.Group), Risk: RiskSafe}
			groups[item.Group] = g
		}
		g.Items = append(g.Items, item)
		g.Bytes += item.Bytes
		g.Files += item.Files
		if riskRank(item.Risk) > riskRank(g.Risk) {
			g.Risk = item.Risk
		}
		report.Bytes += item.Bytes
		report.Files += item.Files
	}

	report.Groups = orderGroups(groups)
	return report, nil
}

func orderGroups(groups map[string]*Group) []Group {
	out := make([]Group, 0, len(groups))
	for _, id := range GroupOrder {
		if g, ok := groups[id]; ok {
			sort.Slice(g.Items, func(i, j int) bool { return g.Items[i].Bytes > g.Items[j].Bytes })
			out = append(out, *g)
			delete(groups, id)
		}
	}
	rest := make([]Group, 0, len(groups))
	for _, g := range groups {
		sort.Slice(g.Items, func(i, j int) bool { return g.Items[i].Bytes > g.Items[j].Bytes })
		rest = append(rest, *g)
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].Bytes > rest[j].Bytes })
	return append(out, rest...)
}

func dropAncestors(cands []candidate) []candidate {
	paths := make([]string, len(cands))
	for i, c := range cands {
		paths[i] = c.path + string(filepath.Separator)
	}
	out := cands[:0]
	for i, c := range cands {
		ancestor := false
		for j, p := range paths {
			if i == j {
				continue
			}
			if strings.HasPrefix(p, paths[i]) {
				ancestor = true
				break
			}
		}
		if !ancestor {
			out = append(out, c)
		}
	}
	return out
}

// enumerate mengumpulkan path pada depth tertentu di bawah root.
// depth=1 berarti anak langsung. File hanya disertakan bila includeFiles.
func enumerate(root string, depth int, includeFiles bool, match func(string) bool) []string {
	var out []string
	var walk func(dir string, level int)
	walk = func(dir string, level int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if match != nil && !match(name) {
				continue
			}
			p := filepath.Join(dir, name)
			isDir := e.IsDir()
			if !isDir && !includeFiles {
				continue
			}
			if level+1 >= depth {
				out = append(out, p)
				continue
			}
			if isDir {
				walk(p, level+1)
			}
		}
	}
	walk(root, 0)
	return out
}

// TotalByRisk menghitung total byte per tingkat risiko.
func TotalByRisk(r Report) map[Risk]int64 {
	out := map[Risk]int64{}
	for _, g := range r.Groups {
		for _, it := range g.Items {
			out[it.Risk] += it.Bytes
		}
	}
	return out
}

func (r Report) String() string {
	return fmt.Sprintf("%d grup · %s · %d file", len(r.Groups), fsx.Human(r.Bytes), r.Files)
}
