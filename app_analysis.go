package main

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/analysis"
	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
)

const analysisCacheTTL = 60 * time.Second

// AnalysisCleanRequest memakai ID dari hasil analisis terakhir, bukan path.
type AnalysisCleanRequest struct {
	IDs              []string `json:"ids"`
	Mode             string   `json:"mode"`
	ConfirmPermanent bool     `json:"confirmPermanent"`
	DryRun           bool     `json:"dryRun"`
}

type analysisTarget struct {
	Target clean.Target
}

func analysisKey(root string, depth int) string {
	return filepath.Clean(root) + "|" + strconv.Itoa(depth)
}

func (a *App) startAnalysis() context.Context {
	ctx, cancel := context.WithCancel(a.baseCtx())
	a.analysisMu.Lock()
	a.analysisCancel = cancel
	a.analysisMu.Unlock()
	return ctx
}

// AnalysisCancel menghentikan pemindaian analisis yang sedang berjalan.
func (a *App) AnalysisCancel() bool {
	a.analysisMu.Lock()
	cancel := a.analysisCancel
	a.analysisMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// AnalysisTree mengembalikan tree disk usage. Cache 60 detik per root/depth.
func (a *App) AnalysisTree(root string, depth int) (analysis.Tree, error) {
	key := analysisKey(root, depth)
	a.analysisMu.Lock()
	if tr, ok := a.analysisTrees[key]; ok {
		if at := a.analysisTreesAt[key]; time.Since(at) < analysisCacheTTL {
			a.analysisMu.Unlock()
			return tr, nil
		}
	}
	a.analysisMu.Unlock()

	tr, err := analysis.ScanTree(a.startAnalysis(), analysis.Options{
		Root:     root,
		Depth:    depth,
		Excluded: a.settings().Excluded,
	})
	if err != nil {
		return tr, err
	}
	a.analysisMu.Lock()
	if a.analysisTrees == nil {
		a.analysisTrees = map[string]analysis.Tree{}
		a.analysisTreesAt = map[string]time.Time{}
	}
	a.analysisTrees[key] = tr
	a.analysisTreesAt[key] = time.Now()
	a.analysisMu.Unlock()
	return tr, nil
}

// AnalysisOverview mengembalikan file besar, file lama, dan artefak build.
func (a *App) AnalysisOverview(root string) (analysis.Overview, error) {
	key := filepath.Clean(root)
	a.analysisMu.Lock()
	if ov, ok := a.analysisOverviews[key]; ok {
		if at := a.analysisOverviewsAt[key]; time.Since(at) < analysisCacheTTL {
			a.analysisMu.Unlock()
			return ov, nil
		}
	}
	a.analysisMu.Unlock()

	ov, err := analysis.ScanOverview(a.startAnalysis(), analysis.Options{
		Root:       root,
		LargeBytes: a.settings().LargeBytes,
		StaleDays:  a.settings().StaleDays,
		Excluded:   a.settings().Excluded,
		ExtraRoots: a.settings().ArtifactRoots,
	})
	if err != nil {
		return ov, err
	}
	a.analysisMu.Lock()
	if a.analysisOverviews == nil {
		a.analysisOverviews = map[string]analysis.Overview{}
		a.analysisOverviewsAt = map[string]time.Time{}
	}
	a.analysisOverviews[key] = ov
	a.analysisOverviewsAt[key] = time.Now()
	a.indexOverview(ov)
	a.analysisMu.Unlock()
	return ov, nil
}

func (a *App) indexOverview(ov analysis.Overview) {
	if a.analysisTargets == nil {
		a.analysisTargets = map[string]analysisTarget{}
	}
	for _, f := range ov.Large {
		a.indexAnalysisFile("large", f.Path, ov.Root, f.Bytes)
	}
	for _, f := range ov.Stale {
		a.indexAnalysisFile("stale", f.Path, ov.Root, f.Bytes)
	}
	for _, art := range ov.Artifacts {
		a.indexAnalysisFile("artifact", art.Path, ov.Root, art.Bytes)
	}
}

func (a *App) indexAnalysisFile(prefix, path, root string, bytes int64) {
	id := prefix + ":" + path
	a.analysisTargets[id] = analysisTarget{
		Target: clean.Target{
			ID:    id,
			Path:  path,
			Root:  root,
			Label: filepath.Base(path),
			Depth: 64,
			Bytes: bytes,
		},
	}
}

// AnalysisDuplicates mencari file identik. Pencarian isi file lebih mahal,
// jadi cache-nya tetap 60 detik dan tidak dijalankan saat layar dibuka.
func (a *App) AnalysisDuplicates(root string) (analysis.Duplicates, error) {
	key := filepath.Clean(root)
	a.analysisMu.Lock()
	if d, ok := a.analysisDups[key]; ok {
		if at := a.analysisDupsAt[key]; time.Since(at) < analysisCacheTTL {
			a.analysisMu.Unlock()
			return d, nil
		}
	}
	a.analysisMu.Unlock()

	d, err := analysis.FindDuplicates(a.startAnalysis(), analysis.Options{Root: root, Excluded: a.settings().Excluded})
	if err != nil {
		return d, err
	}
	a.analysisMu.Lock()
	if a.analysisDups == nil {
		a.analysisDups = map[string]analysis.Duplicates{}
		a.analysisDupsAt = map[string]time.Time{}
	}
	a.analysisDups[key] = d
	a.analysisDupsAt[key] = time.Now()
	if a.analysisTargets == nil {
		a.analysisTargets = map[string]analysisTarget{}
	}
	for _, g := range d.Groups {
		for _, f := range g.Files {
			a.indexAnalysisFile("dup", f.Path, d.Root, f.Bytes)
		}
	}
	a.analysisMu.Unlock()
	return d, nil
}

// AnalysisRefresh membuang cache analisis.
func (a *App) AnalysisRefresh() bool {
	a.analysisMu.Lock()
	a.analysisTrees = nil
	a.analysisTreesAt = nil
	a.analysisOverviews = nil
	a.analysisOverviewsAt = nil
	a.analysisDups = nil
	a.analysisDupsAt = nil
	a.analysisTargets = nil
	if a.analysisCancel != nil {
		a.analysisCancel()
		a.analysisCancel = nil
	}
	a.analysisMu.Unlock()
	return true
}

// AnalysisClean menghapus item analisis memakai guard clean yang sama.
func (a *App) AnalysisClean(req AnalysisCleanRequest) (clean.Result, error) {
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
	a.analysisMu.Lock()
	targets := make([]clean.Target, 0, len(req.IDs))
	seen := map[string]bool{}
	for _, id := range req.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		t, ok := a.analysisTargets[id]
		if !ok {
			a.analysisMu.Unlock()
			return clean.Result{}, errors.New("item analisis tidak dikenal: " + id)
		}
		targets = append(targets, t.Target)
	}
	a.analysisMu.Unlock()

	res := clean.Execute(a.baseCtx(), clean.Request{
		Targets: targets,
		Mode:    mode,
		DryRun:  req.DryRun,
	}, clean.Options{})
	if !req.DryRun {
		a.analysisMu.Lock()
		for _, it := range res.Items {
			if it.Status == clean.StatusRemoved {
				delete(a.analysisTargets, it.ID)
			}
		}
		a.analysisMu.Unlock()
	}
	return res, nil
}

// DiskStatus membaca pemakaian filesystem untuk kartu dashboard.
func (a *App) DiskStatus() (fsx.Volume, error) {
	return fsx.Statfs("/")
}
