package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/config"
	"github.com/Juragan-Koding/cleanlie/internal/direnv"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

const direnvCacheTTL = 60 * time.Second

// DirenvCleanRequest memakai ID hasil scan terakhir, bukan path dari UI.
type DirenvCleanRequest struct {
	IDs              []string `json:"ids"`
	Mode             string   `json:"mode"`
	ConfirmPermanent bool     `json:"confirmPermanent"`
	DryRun           bool     `json:"dryRun"`
}

func (a *App) startDirenv() context.Context {
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.direnvMu.Lock()
	a.direnvCancel = cancel
	a.direnvMu.Unlock()
	return ctx
}

// DirenvCancel menghentikan pemindaian direnv yang sedang berjalan.
func (a *App) DirenvCancel() bool {
	a.direnvMu.Lock()
	cancel := a.direnvCancel
	a.direnvMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func direnvKey(roots, excluded []string) string {
	return strings.Join(roots, "\x00") + "\x01" + strings.Join(excluded, "\x00")
}

// DirenvScan mencari folder .direnv di setiap root yang ditambahkan pengguna,
// ditambah cache global ~/.cache/direnv dan ~/.cache/nix-direnv. Cache 60 detik.
func (a *App) DirenvScan() (direnv.Report, error) {
	cfg := a.settings()
	roots := append([]string(nil), cfg.DirenvRoots...)
	if len(roots) == 0 {
		return direnv.Report{}, errors.New("belum ada folder; tambahkan folder proyek dulu")
	}
	key := direnvKey(roots, cfg.Excluded)
	a.direnvMu.Lock()
	if a.direnvReport != nil && a.direnvKey == key && time.Since(a.direnvAt) < direnvCacheTTL {
		rep := *a.direnvReport
		a.direnvMu.Unlock()
		return rep, nil
	}
	a.direnvMu.Unlock()

	rep, err := direnv.Scan(a.startDirenv(), direnv.Options{
		Roots:    roots,
		Excluded: cfg.Excluded,
	})
	if err != nil {
		return rep, err
	}

	a.direnvMu.Lock()
	a.direnvReport = &rep
	a.direnvKey = key
	a.direnvAt = time.Now()
	a.direnvTargets = make(map[string]clean.Target, len(rep.Entries))
	for _, e := range rep.Entries {
		root := e.Project
		if e.Kind == "cache" {
			root = e.Path
		}
		label := filepath.Base(e.Path)
		if e.Kind == "project" {
			label = filepath.Base(e.Project) + "/" + direnv.DirName
		}
		a.direnvTargets[e.ID] = clean.Target{
			ID:         e.ID,
			Path:       e.Path,
			Root:       root,
			Label:      label,
			Whole:      true,
			Depth:      1,
			Bytes:      e.Bytes,
			FallbackOK: true,
		}
	}
	a.direnvMu.Unlock()
	return rep, nil
}

// DirenvRefresh membuang cache hasil scan direnv.
func (a *App) DirenvRefresh() bool {
	a.direnvMu.Lock()
	a.direnvReport = nil
	a.direnvKey = ""
	a.direnvAt = time.Time{}
	a.direnvTargets = nil
	if a.direnvCancel != nil {
		a.direnvCancel()
		a.direnvCancel = nil
	}
	a.direnvMu.Unlock()
	return true
}

// DirenvPickRoot membuka dialog folder native dan mengembalikan path yang
// dipilih. String kosong berarti pengguna membatalkan dialog.
func (a *App) DirenvPickRoot() (string, error) {
	cfg := a.settings()
	start := rules.Home()
	if len(cfg.DirenvRoots) > 0 {
		start = cfg.DirenvRoots[len(cfg.DirenvRoots)-1]
	}
	if fi, err := os.Stat(start); err != nil || !fi.IsDir() {
		start = rules.Home()
	}
	title := "Pilih folder proyek"
	if cfg.Language == config.LanguageEN {
		title = "Choose project folder"
	}
	dir, err := wruntime.OpenDirectoryDialog(a.baseCtx(), wruntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: start,
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// DirenvAddRoot menambahkan folder proyek ke config lalu menyimpannya.
func (a *App) DirenvAddRoot(path string) (SettingsView, error) {
	root, err := normalizeDirenvRoot(path)
	if err != nil {
		return a.settingsView(), err
	}
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return a.settingsView(), errors.New("folder tidak ditemukan: " + root)
	}
	cfg := a.settings()
	for _, r := range cfg.DirenvRoots {
		if filepath.Clean(r) == root {
			return a.settingsView(), nil
		}
	}
	if len(cfg.DirenvRoots) >= direnv.MaxRoots {
		return a.settingsView(), errors.New("maksimal 32 folder; hapus salah satu dulu")
	}
	cfg.DirenvRoots = append(cfg.DirenvRoots, root)
	return a.SaveSettings(cfg)
}

// DirenvRemoveRoot menghapus folder dari daftar scan direnv.
func (a *App) DirenvRemoveRoot(path string) (SettingsView, error) {
	root, err := normalizeDirenvRoot(path)
	if err != nil {
		return a.settingsView(), err
	}
	cfg := a.settings()
	out := cfg.DirenvRoots[:0]
	for _, r := range cfg.DirenvRoots {
		if filepath.Clean(r) != root {
			out = append(out, r)
		}
	}
	cfg.DirenvRoots = out
	return a.SaveSettings(cfg)
}

// DirenvClean memindahkan .direnv terpilih ke trash lewat guard clean yang sama.
func (a *App) DirenvClean(req DirenvCleanRequest) (clean.Result, error) {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()
	mode := clean.Mode(req.Mode)
	if mode != clean.ModeTrash && mode != clean.ModePermanent {
		return clean.Result{}, errors.New("mode harus trash atau permanent")
	}
	if mode == clean.ModePermanent && !req.ConfirmPermanent && !req.DryRun {
		return clean.Result{}, errors.New("hapus permanen butuh konfirmasi")
	}
	if len(req.IDs) == 0 {
		return clean.Result{}, errors.New("tidak ada item yang dipilih")
	}
	excluded := a.settings().Excluded
	a.direnvMu.Lock()
	targets := make([]clean.Target, 0, len(req.IDs))
	seen := map[string]bool{}
	for _, id := range req.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		t, ok := a.direnvTargets[id]
		if !ok {
			a.direnvMu.Unlock()
			return clean.Result{}, errors.New("item direnv tidak dikenal: " + id)
		}
		if excludedPath(t.Path, excluded) {
			a.direnvMu.Unlock()
			return clean.Result{}, errors.New("item berada di path yang dikecualikan: " + t.Path)
		}
		targets = append(targets, t)
	}
	a.direnvMu.Unlock()

	res := clean.Execute(a.baseCtx(), clean.Request{
		Targets: targets,
		Mode:    mode,
		DryRun:  req.DryRun,
	}, clean.Options{})
	if !req.DryRun {
		a.direnvMu.Lock()
		changed := false
		for _, it := range res.Items {
			if it.Status == clean.StatusRemoved {
				delete(a.direnvTargets, it.ID)
				changed = true
			}
		}
		if changed {
			a.direnvReport = nil
			a.direnvKey = ""
		}
		a.direnvMu.Unlock()
		a.invalidateAfterMutation()
		a.notifyCleanResult("Cleanlie — pembersihan direnv selesai", res)
	}
	return res, nil
}

func normalizeDirenvRoot(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path folder kosong")
	}
	path = rules.ExpandHome(path)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return "", errors.New("path folder harus absolut")
	}
	return path, nil
}
