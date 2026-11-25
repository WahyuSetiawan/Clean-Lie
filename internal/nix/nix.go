// Package nix membaca status Nix (store, generasi, dead paths) dan
// menjalankan aksi yang butuh hak root lewat internal/privileged.
//
// Semua pembacaan bersifat read-only. Penghapusan generasi dan GC hanya
// dijalankan setelah konfirmasi teks "GC" dari UI.
package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/tools"
)

// StorePath adalah root store Nix.
const StorePath = "/nix/store"

// SuggestedDays adalah umur generasi yang disarankan untuk dihapus.
const SuggestedDays = 30

// Generation adalah satu generasi profile.
type Generation struct {
	Profile      string `json:"profile"`
	ProfilePath  string `json:"profilePath"`
	Number       int    `json:"number"`
	Date         string `json:"date"`
	AgeDays      int    `json:"ageDays"`
	Current      bool   `json:"current"`
	Suggested    bool   `json:"suggested"`
	StorePath    string `json:"storePath,omitempty"`
	ClosureBytes int64  `json:"closureBytes"`
	LinkMissing  bool   `json:"linkMissing,omitempty"`
}

// ProfileInfo merangkum satu profile.
type ProfileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Count   int    `json:"count"`
	Current int    `json:"current"`
	Oldest  string `json:"oldest,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Status adalah snapshot layar Nix.
type Status struct {
	StorePath    string        `json:"storePath"`
	StoreBytes   int64         `json:"storeBytes"`
	StorePaths   int           `json:"storePaths"`
	Generations  []Generation  `json:"generations"`
	Profiles     []ProfileInfo `json:"profiles"`
	JournalRaw   string        `json:"journalRaw,omitempty"`
	JournalBytes int64         `json:"journalBytes"`
	FreeBytes    int64         `json:"freeBytes"`
	TotalBytes   int64         `json:"totalBytes"`
	Agent        bool          `json:"agent"`
	GCCommand    string        `json:"gcCommand"`
	ScannedAt    string        `json:"scannedAt"`
	DurationMs   int64         `json:"durationMs"`
	Errors       []string      `json:"errors,omitempty"`
}

// Dead adalah hasil dry-run garbage collection.
type Dead struct {
	Count      int    `json:"count"`
	Bytes      int64  `json:"bytes"`
	Source     string `json:"source"`
	ScannedAt  string `json:"scannedAt"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

// Options mengatur pembacaan.
type Options struct {
	Run     func(ctx context.Context, cmd []string, timeout time.Duration) tools.Result
	Now     func() time.Time
	Home    string
	Timeout time.Duration
	// Agent bisa diganti di test; default deteksi polkit agent.
	Agent func() bool
}

func (o Options) run(ctx context.Context, cmd []string, timeout time.Duration) tools.Result {
	if o.Run == nil {
		o.Run = tools.Run
	}
	if timeout <= 0 {
		timeout = o.Timeout
	}
	return o.Run(ctx, cmd, timeout)
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) home() string {
	if o.Home != "" {
		return o.Home
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

type profileSpec struct {
	Name   string
	Path   string
	System bool
}

func profileSpecs(home string) []profileSpec {
	return []profileSpec{
		{Name: "system", Path: "/nix/var/nix/profiles/system", System: true},
		{Name: "home-manager", Path: filepath.Join(home, ".local/state/nix/profiles/home-manager")},
		{Name: "user", Path: filepath.Join(home, ".local/state/nix/profiles/profile")},
	}
}

// Status memindai store, profile, generasi, dan journal. Tidak menghitung
// dead paths (dipanggil terpisah karena bisa puluhan detik).
func ReadStatus(ctx context.Context, opts Options) Status {
	started := opts.now()
	st := Status{StorePath: StorePath, ScannedAt: started.Format(time.RFC3339)}

	if v, err := fsx.Statfs("/"); err == nil {
		st.FreeBytes, st.TotalBytes = v.Free, v.Total
	} else {
		st.Errors = append(st.Errors, "statfs: "+err.Error())
	}

	store, err := storeInfo(ctx, opts)
	if err != nil {
		st.Errors = append(st.Errors, "store: "+err.Error())
	} else {
		st.StoreBytes, st.StorePaths = store.bytes, store.paths
	}

	st.JournalBytes, st.JournalRaw, err = journalUsage(ctx, opts)
	if err != nil {
		st.Errors = append(st.Errors, "journal: "+err.Error())
	}

	for _, spec := range profileSpecs(opts.home()) {
		info, gens := loadProfile(ctx, spec, opts, true)
		st.Profiles = append(st.Profiles, info)
		st.Generations = append(st.Generations, gens...)
		if info.Error != "" {
			st.Errors = append(st.Errors, spec.Name+": "+info.Error)
		}
	}
	sort.SliceStable(st.Generations, func(i, j int) bool {
		if st.Generations[i].Profile != st.Generations[j].Profile {
			return profileOrder(st.Generations[i].Profile) < profileOrder(st.Generations[j].Profile)
		}
		return st.Generations[i].Number > st.Generations[j].Number
	})
	st.DurationMs = time.Since(started).Milliseconds()
	return st
}

func profileOrder(name string) int {
	switch name {
	case "system":
		return 0
	case "home-manager":
		return 1
	case "user":
		return 2
	}
	return 9
}

type storeSummary struct {
	bytes int64
	paths int
}

func storeInfo(ctx context.Context, opts Options) (storeSummary, error) {
	out := opts.run(ctx, []string{"nix", "path-info", "--json", "--all"}, 90*time.Second)
	if out.Err != nil {
		return storeSummary{}, fmt.Errorf("%v: %s", out.Err, firstLine(out.Stderr))
	}
	var raw map[string]struct {
		NarSize int64 `json:"narSize"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &raw); err != nil {
		return storeSummary{}, fmt.Errorf("parse json: %w", err)
	}
	var s storeSummary
	for _, v := range raw {
		s.bytes += v.NarSize
		s.paths++
	}
	return s, nil
}

func journalUsage(ctx context.Context, opts Options) (int64, string, error) {
	out := opts.run(ctx, []string{"journalctl", "--disk-usage"}, 15*time.Second)
	raw := strings.TrimSpace(out.Stdout)
	if out.Err != nil {
		return 0, raw, fmt.Errorf("%v: %s", out.Err, firstLine(out.Stderr))
	}
	n, ok := parseSize(raw)
	if !ok {
		return 0, raw, fmt.Errorf("tidak bisa parse: %s", firstLine(raw))
	}
	return n, raw, nil
}

// loadProfile membaca link generasi, tanggal, current, dan (opsional)
// closure size.
func loadProfile(ctx context.Context, spec profileSpec, opts Options, closures bool) (ProfileInfo, []Generation) {
	info := ProfileInfo{Name: spec.Name, Path: spec.Path}
	links, current, err := generationLinks(spec.Path)
	if err != nil {
		info.Error = err.Error()
		return info, nil
	}
	if len(links) == 0 {
		info.Error = "tidak ada generasi"
		return info, nil
	}

	dates := generationDates(ctx, spec, opts)
	now := opts.now()
	gens := make([]Generation, 0, len(links))
	info.Current = current
	for _, l := range links {
		g := Generation{
			Profile:     spec.Name,
			ProfilePath: spec.Path,
			Number:      l.number,
			Current:     l.number == current,
		}
		when := l.mtime
		if d, ok := dates[l.number]; ok {
			g.Date = d.Format("2006-01-02 15:04:05")
			when = d
		} else {
			g.Date = l.mtime.Format("2006-01-02 15:04:05")
		}
		g.AgeDays = int(now.Sub(when).Hours() / 24)
		if g.AgeDays < 0 {
			g.AgeDays = 0
		}
		g.Suggested = !g.Current && g.AgeDays >= SuggestedDays
		g.StorePath = l.store
		if l.store == "" {
			g.LinkMissing = true
		} else if _, err := os.Lstat(l.store); err != nil {
			g.LinkMissing = true
		}
		gens = append(gens, g)
	}

	if closures {
		fillClosures(ctx, gens, opts)
	}
	info.Count = len(gens)
	if len(gens) > 0 {
		oldest := gens[0].Date
		for _, g := range gens[1:] {
			if g.Date < oldest {
				oldest = g.Date
			}
		}
		info.Oldest = oldest
	}
	return info, gens
}

type genLink struct {
	number int
	store  string
	mtime  time.Time
}

func generationLinks(profilePath string) ([]genLink, int, error) {
	target, err := os.Readlink(profilePath)
	if err != nil {
		return nil, 0, err
	}
	base := filepath.Base(target)
	prefix, current, ok := splitGenLink(base)
	if !ok {
		base = filepath.Base(profilePath)
		prefix, current, ok = splitGenLink(base)
		if !ok {
			prefix = base
		}
	}
	dir := filepath.Dir(profilePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	matcher := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-(\d+)-link$`)
	var links []genLink
	for _, e := range entries {
		m := matcher.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		linkPath := filepath.Join(dir, e.Name())
		store, _ := os.Readlink(linkPath)
		if store != "" && !filepath.IsAbs(store) {
			store = filepath.Join(dir, store)
		}
		fi, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		links = append(links, genLink{number: n, store: store, mtime: fi.ModTime()})
	}
	sort.Slice(links, func(i, j int) bool { return links[i].number > links[j].number })
	return links, current, nil
}

func splitGenLink(name string) (prefix string, number int, ok bool) {
	name = strings.TrimSuffix(name, "-link")
	i := strings.LastIndexByte(name, '-')
	if i <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(name[i+1:])
	if err != nil {
		return "", 0, false
	}
	return name[:i], n, true
}

func generationDates(ctx context.Context, spec profileSpec, opts Options) map[int]time.Time {
	outDates := map[int]time.Time{}
	if spec.System {
		// nix-env membuka lock profile system dan gagal sebagai non-root.
		// Tanggal link (mtime) adalah waktu generasi dibuat.
		return outDates
	}
	out := opts.run(ctx, []string{"nix-env", "-p", spec.Path, "--list-generations"}, 30*time.Second)
	if out.Err != nil {
		return outDates
	}
	for _, line := range strings.Split(out.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", fields[1]+" "+fields[2], time.Local)
		if err != nil {
			continue
		}
		outDates[n] = t
	}
	return outDates
}

func fillClosures(ctx context.Context, gens []Generation, opts Options) {
	uniq := map[string]bool{}
	var paths []string
	for _, g := range gens {
		if g.StorePath == "" || uniq[g.StorePath] {
			continue
		}
		uniq[g.StorePath] = true
		paths = append(paths, g.StorePath)
	}
	if len(paths) == 0 {
		return
	}
	sizes := map[string]int64{}
	for start := 0; start < len(paths); start += 300 {
		end := start + 300
		if end > len(paths) {
			end = len(paths)
		}
		cmd := append([]string{"nix", "path-info", "-S"}, paths[start:end]...)
		out := opts.run(ctx, cmd, 90*time.Second)
		if out.Err != nil {
			continue
		}
		for p, n := range parsePathInfoS(out.Stdout) {
			sizes[p] = n
		}
	}
	for i := range gens {
		gens[i].ClosureBytes = sizes[gens[i].StorePath]
	}
}

// DeadPaths menjalankan dry-run GC dan menghitung ukuran path mati.
func DeadPaths(ctx context.Context, opts Options) Dead {
	started := opts.now()
	d := Dead{Source: "nix-store --gc --print-dead", ScannedAt: started.Format(time.RFC3339)}

	out := opts.run(ctx, []string{"nix-store", "--gc", "--print-dead"}, 5*time.Minute)
	paths := parseDeadPaths(out.Stdout)
	if out.Err != nil {
		// Fallback: nix-collect-garbage hanya memberi jumlah, bukan ukuran.
		fallback := opts.run(ctx, []string{"nix-collect-garbage", "--dry-run"}, 5*time.Minute)
		d.Source = "nix-collect-garbage --dry-run"
		d.Count = parseGCDeletedCount(fallback.Stdout + "\n" + fallback.Stderr)
		if fallback.Err != nil {
			d.Error = fmt.Sprintf("%v: %s", fallback.Err, firstLine(fallback.Stderr))
		} else if d.Count == 0 {
			d.Error = "tidak bisa parse jumlah path"
		}
		d.DurationMs = time.Since(started).Milliseconds()
		return d
	}

	d.Count = len(paths)
	if d.Count == 0 {
		d.DurationMs = time.Since(started).Milliseconds()
		return d
	}
	var size int64
	for start := 0; start < len(paths); start += 300 {
		end := start + 300
		if end > len(paths) {
			end = len(paths)
		}
		cmd := append([]string{"nix", "path-info", "--json"}, paths[start:end]...)
		info := opts.run(ctx, cmd, 60*time.Second)
		if info.Err != nil {
			if d.Error == "" {
				d.Error = "ukur dead paths: " + firstLine(info.Stderr)
			}
			continue
		}
		for _, n := range parseNarSizes(info.Stdout) {
			size += n
		}
	}
	d.Bytes = size
	d.DurationMs = time.Since(started).Milliseconds()
	return d
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 240 {
		s = s[:240] + "…"
	}
	return s
}
