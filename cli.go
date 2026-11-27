package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/analysis"
	"github.com/Juragan-Koding/cleanlie/internal/android"
	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/config"
	"github.com/Juragan-Koding/cleanlie/internal/direnv"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/insight"
	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/privileged"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

// runCLI menangani mode headless. Return handled=false berarti jalankan GUI.
func runCLI(args []string) (handled bool, code int) {
	if len(args) == 0 {
		return false, 0
	}

	headless := false
	for _, a := range args {
		switch a {
		case "--dry-run", "-dry-run", "--json", "-json", "--scan-only", "-scan-only",
			"--nix-status", "-nix-status", "--nix-dead", "-nix-dead",
			"--android-status", "-android-status",
			"--analysis-tree", "-analysis-tree", "--analysis-overview", "-analysis-overview",
			"--analysis-dups", "-analysis-dups",
			"--direnv-scan", "-direnv-scan",
			"--trash-list", "-trash-list", "--trash-restore", "-trash-restore",
			"--trash-delete", "-trash-delete", "--trash-empty", "-trash-empty",
			"--history", "-history", "--undo", "-undo",
			"--insight", "-insight", "--report", "-report",
			"--version", "-version", "--help", "-help", "-h":
			headless = true
		}
	}
	if !headless {
		return false, 0
	}

	fs := flag.NewFlagSet("cleanlie", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut    = fs.Bool("json", false, "cetak laporan sebagai JSON")
		dryRun     = fs.Bool("dry-run", false, "pindai dan tampilkan rencana (tidak menghapus)")
		scanOnly   = fs.Bool("scan-only", false, "pindai satu direktori")
		path       = fs.String("path", "", "direktori yang dipindai untuk --scan-only")
		group      = fs.String("group", "", "batasi ke satu group (mis. dev.node)")
		workers    = fs.Int("workers", 0, "jumlah worker pemindaian (0 = otomatis)")
		showVer    = fs.Bool("version", false, "cetak versi")
		nixStat    = fs.Bool("nix-status", false, "cetak status Nix sebagai JSON")
		nixDead    = fs.Bool("nix-dead", false, "hitung dead paths Nix sebagai JSON")
		andStat    = fs.Bool("android-status", false, "status AVD/SDK Android sebagai JSON")
		anTree     = fs.Bool("analysis-tree", false, "tree disk usage sebagai JSON")
		anOverview = fs.Bool("analysis-overview", false, "file besar/lama/artefak sebagai JSON")
		anDups     = fs.Bool("analysis-dups", false, "file duplikat sebagai JSON")
		dirScan    = fs.Bool("direnv-scan", false, "scan folder .direnv sebagai JSON")
		anDepth    = fs.Int("depth", 2, "kedalaman tree analisis")
		trashList  = fs.Bool("trash-list", false, "daftar isi trash (JSON)")
		noSize     = fs.Bool("no-size", false, "jangan ukur ukuran trash")
		trashRest  = fs.String("trash-restore", "", "pulihkan item trash berdasarkan ID (pisahkan koma)")
		trashDel   = fs.String("trash-delete", "", "hapus permanen item trash berdasarkan ID (pisahkan koma)")
		trashEmpty = fs.Bool("trash-empty", false, "kosongkan seluruh trash")
		history    = fs.Bool("history", false, "riwayat sesi pembersihan (JSON)")
		undoBatch  = fs.String("undo", "", "undo satu sesi berdasarkan batch")
		limit      = fs.Int("limit", 50, "jumlah sesi/entry maksimum")
		yes        = fs.Bool("yes", false, "izinkan tindakan destruktif tanpa prompt")
		insightOut = fs.Bool("insight", false, "ringkasan mingguan + tren disk (JSON)")
		reportFmt  = fs.String("report", "", "cetak laporan: md, json, atau csv")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Cleanlie %s

Pemakaian:
  cleanlie                         buka GUI
  cleanlie --dry-run               pindai dan cetak rencana
  cleanlie --dry-run --json        rencana dalam JSON
  cleanlie --scan-only --path DIR  ukur satu direktori
  cleanlie --nix-status            status store, generasi, journal (JSON)
  cleanlie --nix-dead              dry-run GC: jumlah + ukuran dead paths
  cleanlie --android-status        status AVD/SDK Android (JSON)
  cleanlie --analysis-tree         tree disk usage (JSON; --path, --depth)
  cleanlie --analysis-overview     file besar, lama, artefak (JSON)
  cleanlie --analysis-dups         file duplikat (JSON)
  cleanlie --direnv-scan           folder .direnv di --path (JSON)
  cleanlie --trash-list            isi trash (JSON; --no-size)
  cleanlie --trash-restore ID      pulihkan item trash (butuh --yes)
  cleanlie --trash-delete ID       hapus permanen item trash (butuh --yes)
  cleanlie --trash-empty           kosongkan trash (butuh --yes)
  cleanlie --history               riwayat sesi (JSON; --limit)
  cleanlie --undo BATCH            undo sesi trash (butuh --yes)
  cleanlie --insight               ringkasan mingguan + tren (JSON)
  cleanlie --report md|json|csv    cetak laporan ke stdout
  cleanlie --version

Opsi destruktif selalu butuh --yes; --dry-run menampilkan rencana.

Opsi:
`, version)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	if *showVer {
		fmt.Printf("cleanlie %s (%s)\n", version, commit)
		return true, 0
	}

	if *insightOut || *reportFmt != "" {
		rep, err := insight.Collect(context.Background(), version, time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return true, 1
		}
		if *reportFmt != "" {
			out, err := insight.Render(rep, *reportFmt)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cleanlie:", err)
				return true, 2
			}
			fmt.Print(out)
			return true, 0
		}
		printJSON(rep)
		return true, 0
	}

	if *trashList || *trashRest != "" || *trashDel != "" || *trashEmpty || *history || *undoBatch != "" {
		return true, runTrashCLI(*trashList, *noSize, *trashRest, *trashDel, *trashEmpty, *history, *undoBatch, *limit, *yes, *dryRun, *jsonOut)
	}

	if *andStat {
		printJSON(android.ReadStatus(context.Background(), android.Options{}))
		return true, 0
	}
	if *dirScan {
		roots := []string{}
		if *path != "" {
			roots = append(roots, *path)
		} else if cfg, _ := config.Load(); len(cfg.DirenvRoots) > 0 {
			roots = cfg.DirenvRoots
		}
		if len(roots) == 0 {
			fmt.Fprintln(os.Stderr, "cleanlie: belum ada folder direnv; pakai --path DIR atau tambahkan di layar direnv")
			return true, 1
		}
		rep, err := direnv.Scan(context.Background(), direnv.Options{Roots: roots})
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return true, 1
		}
		printJSON(rep)
		return true, 0
	}
	if *anTree || *anOverview || *anDups {
		root := *path
		if root == "" {
			root = rules.Home()
		}
		anOpts := analysis.Options{Root: root, Depth: *anDepth}
		switch {
		case *anTree:
			tr, err := analysis.ScanTree(context.Background(), anOpts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cleanlie:", err)
				return true, 1
			}
			printJSON(tr)
		case *anOverview:
			ov, err := analysis.ScanOverview(context.Background(), anOpts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cleanlie:", err)
				return true, 1
			}
			printJSON(ov)
		case *anDups:
			d, err := analysis.FindDuplicates(context.Background(), anOpts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "cleanlie:", err)
				return true, 1
			}
			printJSON(d)
		}
		return true, 0
	}

	if *nixStat || *nixDead {
		if *nixDead {
			d := nix.DeadPaths(context.Background(), nix.Options{})
			printJSON(d)
			if d.Error != "" {
				return true, 1
			}
			return true, 0
		}
		st := nix.ReadStatus(context.Background(), nix.Options{})
		st.Agent = privileged.DetectAgent()
		st.GCCommand = nix.GCPreview(st.Agent)
		printJSON(st)
		if len(st.Errors) > 0 {
			return true, 1
		}
		return true, 0
	}

	if *scanOnly && *path == "" {
		*path = rules.Home()
	}
	if *path != "" && !*scanOnly {
		*scanOnly = true
	}

	ctx := context.Background()
	list := rules.DefaultRules()
	if *scanOnly {
		root, err := filepath.Abs(rules.ExpandHome(*path))
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return true, 1
		}
		list = []rules.Rule{{
			ID: "scan", Group: "scan", Label: filepath.Base(root), Path: root,
			Depth: 1, IncludeFiles: true, Risk: rules.RiskSafe, Priority: 1,
			Note: "hasil scan manual",
		}}
	}
	if *group != "" {
		filtered := list[:0]
		for _, r := range list {
			if r.Group == *group {
				filtered = append(filtered, r)
			}
		}
		list = filtered
	}

	var lastProgress time.Time
	opts := rules.PlanOptions{Workers: *workers}
	if !*jsonOut {
		opts.Progress = func(p rules.Progress) {
			if time.Since(lastProgress) < 200*time.Millisecond || p.Total == 0 {
				return
			}
			lastProgress = time.Now()
			fmt.Fprintf(os.Stderr, "\r%d/%d item · %s", p.Done, p.Total, shorten(p.Path))
		}
	}

	report, err := rules.Plan(ctx, list, opts)
	if !*jsonOut && lastProgress != (time.Time{}) {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return true, 1
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return true, 1
		}
		return true, 0
	}

	printReport(report, *dryRun)
	return true, 0
}

func printReport(r rules.Report, dryRun bool) {
	if dryRun {
		fmt.Println("Cleanlie — dry run (tidak ada yang dihapus)")
	}
	fmt.Printf("Total: %s · %d file · %s\n", fsx.Human(r.Bytes), r.Files, time.Duration(r.Duration)*time.Millisecond)
	fmt.Println()

	for _, g := range r.Groups {
		fmt.Printf("[%s] %s — %s\n", g.ID, g.Label, fsx.Human(g.Bytes))
		for _, it := range g.Items {
			fmt.Printf("  %9s  %-28s %s", fsx.Human(it.Bytes), truncate(it.Label, 28), shorten(it.Path))
			if it.Risk != rules.RiskSafe {
				fmt.Printf("  (%s)", it.Risk)
			}
			fmt.Println()
			if it.CommandLine != "" {
				fmt.Printf("             → %s\n", it.CommandLine)
			} else if it.Note != "" {
				fmt.Printf("             %s\n", it.Note)
			}
		}
		fmt.Println()
	}
	if len(r.Errors) > 0 {
		fmt.Printf("%d path gagal dipindai:\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Printf("  %s: %s\n", shorten(e.Path), e.Error)
		}
	}
}

func shorten(p string) string {
	h := rules.Home()
	if h != "" && strings.HasPrefix(p, h+string(filepath.Separator)) {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
	}
}

// runTrashCLI menangani perintah trash/riwayat headless. Semua tindakan
// destruktif wajib --yes (kecuali --dry-run) dan tidak pernah menebak path.
func runTrashCLI(list, noSize bool, restore, del string, empty, history bool, undo string, limit int, yes, dryRun, jsonOut bool) int {
	ctx := context.Background()

	if list {
		rep, err := trash.List(ctx, trash.ListOptions{MaxEntries: 5000})
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return 1
		}
		if !noSize {
			rep.Entries = trash.SizeEntries(ctx, rep.Entries, trash.SizeOptions{Workers: 4})
			rep.Bytes = sumTrashBytes(rep.Entries)
		}
		if jsonOut {
			printJSON(rep)
		} else {
			printTrashReport(rep)
		}
		return 0
	}

	if history {
		page, err := clean.ReadSessions("", limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return 1
		}
		printJSON(page)
		return 0
	}

	destructive := restore != "" || del != "" || empty || undo != ""
	if destructive && !dryRun && !yes {
		fmt.Fprintln(os.Stderr, "cleanlie: tindakan ini butuh --yes (atau pakai --dry-run)")
		return 2
	}

	if restore != "" {
		return runRestoreCLI(ctx, restore, dryRun, jsonOut)
	}
	if del != "" {
		return runDeleteCLI(ctx, del, dryRun, jsonOut)
	}
	if empty {
		return runEmptyCLI(ctx, dryRun, jsonOut)
	}
	if undo != "" {
		return runUndoCLI(ctx, undo, dryRun, jsonOut)
	}
	return 0
}

func splitIDs(s string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, id := range strings.Split(s, ",") {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ID trash kosong")
	}
	return out, nil
}

func trashByID(ctx context.Context, ids []string) ([]trash.Entry, error) {
	rep, err := trash.List(ctx, trash.ListOptions{MaxEntries: 5000})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]trash.Entry, len(rep.Entries))
	for _, e := range rep.Entries {
		byID[e.ID] = e
	}
	out := make([]trash.Entry, 0, len(ids))
	for _, id := range ids {
		e, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("item trash tidak dikenal: %s", id)
		}
		out = append(out, e)
	}
	return out, nil
}

func printTrashReport(rep trash.Report) {
	fmt.Printf("Trash: %d item · %s\n", len(rep.Entries), fsx.Human(rep.Bytes))
	for _, e := range rep.Entries {
		size := "—"
		if e.Bytes >= 0 {
			size = fsx.Human(e.Bytes)
		}
		state := "pulih"
		if !e.Restorable {
			state = "tidak: " + e.Reason
		}
		fmt.Printf("  %-12s %9s  %-28s %s\n", e.ID, size, truncate(e.Name, 28), shorten(e.OriginalPath))
		if !e.Restorable {
			fmt.Printf("               %s\n", state)
		}
	}
}

func printAction(res trash.ActionResult) {
	fmt.Printf("Restored: %d · Deleted: %d · Skipped: %d · Failed: %d · %s\n",
		res.Restored, res.Deleted, res.Skipped, res.Failed, fsx.Human(res.Bytes))
	for _, it := range res.Items {
		fmt.Printf("  %-10s %-28s %s\n", it.Status, truncate(it.OriginalPath, 28), it.Reason)
	}
}

func runRestoreCLI(ctx context.Context, raw string, dryRun, jsonOut bool) int {
	ids, err := splitIDs(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	entries, err := trashByID(ctx, ids)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(entries))}
	if dryRun {
		for _, e := range entries {
			ir := trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: maxBytes(e.Bytes)}
			if !e.Restorable {
				ir.Status, ir.Reason = "skipped", e.Reason
				res.Skipped++
			} else {
				res.Restored++
				res.Bytes += ir.Bytes
			}
			res.Items = append(res.Items, ir)
		}
		return printActionResult(res, jsonOut)
	}
	if err := clean.RotateAudit(""); err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	batch := clean.NewBatch("restore-")
	started := time.Now().Format(time.RFC3339)
	var audits []clean.Entry
	for _, e := range entries {
		ir := trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath}
		if !e.Restorable {
			ir.Status, ir.Reason = "skipped", e.Reason
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		if err := trash.Restore(ctx, e); err != nil {
			ir.Status, ir.Reason = "failed", err.Error()
			res.Failed++
		} else {
			ir.Status, ir.Bytes = "restored", maxBytes(e.Bytes)
			res.Restored++
			res.Bytes += ir.Bytes
		}
		res.Items = append(res.Items, ir)
		status := clean.StatusRemoved
		if ir.Status == "failed" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{Time: started, Mode: clean.ModeTrash, ID: e.ID, Path: e.OriginalPath,
			BytesBefore: ir.Bytes, Status: status, Reason: ir.Reason, TrashPath: e.FilesPath, Batch: batch, Action: "restore"})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	return printActionResult(res, jsonOut)
}

func runDeleteCLI(ctx context.Context, raw string, dryRun, jsonOut bool) int {
	ids, err := splitIDs(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	entries, err := trashByID(ctx, ids)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	if dryRun {
		res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(entries))}
		for _, e := range entries {
			b := maxBytes(e.Bytes)
			res.Deleted++
			res.Bytes += b
			res.Items = append(res.Items, trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: b})
		}
		return printActionResult(res, jsonOut)
	}
	if err := clean.RotateAudit(""); err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	res := trash.Delete(ctx, entries)
	started := time.Now().Format(time.RFC3339)
	batch := clean.NewBatch("trash-")
	var audits []clean.Entry
	for _, ir := range res.Items {
		if ir.Status == "skipped" {
			continue
		}
		status := clean.StatusRemoved
		if ir.Status != "deleted" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{Time: started, Mode: clean.ModePermanent, ID: ir.ID, Path: ir.OriginalPath,
			BytesBefore: ir.Bytes, Freed: ir.Bytes, Status: status, Reason: ir.Reason, Batch: batch, Action: "trash-delete"})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	return printActionResult(res, jsonOut)
}

func runEmptyCLI(ctx context.Context, dryRun, jsonOut bool) int {
	roots := trash.DiscoverRoots()
	if dryRun {
		rep, err := trash.List(ctx, trash.ListOptions{Roots: roots})
		if err != nil {
			fmt.Fprintln(os.Stderr, "cleanlie:", err)
			return 1
		}
		res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(rep.Entries))}
		for _, e := range rep.Entries {
			b := maxBytes(e.Bytes)
			res.Deleted++
			res.Bytes += b
			res.Items = append(res.Items, trash.ItemResult{ID: e.ID, Name: e.Name, OriginalPath: e.OriginalPath, Status: "planned", Bytes: b})
		}
		return printActionResult(res, jsonOut)
	}
	if err := clean.RotateAudit(""); err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	res, err := trash.Empty(ctx, roots, "KOSONGKAN")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	started := time.Now().Format(time.RFC3339)
	batch := clean.NewBatch("trash-")
	var audits []clean.Entry
	for _, ir := range res.Items {
		if ir.Status == "skipped" {
			continue
		}
		status := clean.StatusRemoved
		if ir.Status != "deleted" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{Time: started, Mode: clean.ModePermanent, ID: ir.ID, Path: ir.OriginalPath,
			BytesBefore: ir.Bytes, Freed: ir.Bytes, Status: status, Reason: ir.Reason, Batch: batch, Action: "trash-empty"})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	return printActionResult(res, jsonOut)
}

func runUndoCLI(ctx context.Context, batch string, dryRun, jsonOut bool) int {
	items, ok, err := clean.ReadSessionEntries("", batch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "cleanlie: sesi tidak ditemukan:", batch)
		return 1
	}
	res := trash.ActionResult{Items: make([]trash.ItemResult, 0, len(items))}
	if dryRun {
		for _, e := range items {
			ir := trash.ItemResult{ID: e.ID, Name: e.ID, OriginalPath: e.Path}
			if !clean.RestorableEntry(e) {
				ir.Status, ir.Reason = "skipped", "item tidak bisa di-undo (permanen/perintah)"
				res.Skipped++
			} else {
				ir.Status = "planned"
				res.Restored++
			}
			res.Items = append(res.Items, ir)
		}
		return printActionResult(res, jsonOut)
	}
	if err := clean.RotateAudit(""); err != nil {
		fmt.Fprintln(os.Stderr, "cleanlie:", err)
		return 1
	}
	undoBatch := clean.NewBatch("undo-")
	started := time.Now().Format(time.RFC3339)
	var audits []clean.Entry
	for _, e := range items {
		ir := trash.ItemResult{ID: e.ID, Name: e.ID, OriginalPath: e.Path}
		if !clean.RestorableEntry(e) || e.Path == "" {
			ir.Status, ir.Reason = "skipped", "item tidak bisa di-undo"
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		tr, found := trash.Resolve(ctx, e.Path)
		if !found {
			ir.Status, ir.Reason = "skipped", "jejak trash tidak ditemukan"
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		if !tr.Restorable {
			ir.Status, ir.Reason = "skipped", tr.Reason
			res.Skipped++
			res.Items = append(res.Items, ir)
			continue
		}
		if err := trash.Restore(ctx, tr); err != nil {
			ir.Status, ir.Reason = "failed", err.Error()
			res.Failed++
		} else {
			ir.Status, ir.Bytes = "restored", maxBytes(e.BytesBefore)
			res.Restored++
			res.Bytes += ir.Bytes
		}
		res.Items = append(res.Items, ir)
		status := clean.StatusRemoved
		if ir.Status == "failed" {
			status = clean.StatusFailed
		}
		audits = append(audits, clean.Entry{Time: started, Mode: clean.ModeTrash, ID: e.ID, Path: e.Path,
			BytesBefore: ir.Bytes, Status: status, Reason: ir.Reason, TrashPath: tr.FilesPath, Batch: undoBatch, Action: "restore"})
	}
	if err := clean.AppendAuditBatch("", audits); err != nil {
		res.AuditError = err.Error()
	}
	return printActionResult(res, jsonOut)
}

func printActionResult(res trash.ActionResult, jsonOut bool) int {
	if jsonOut {
		printJSON(res)
	} else {
		printAction(res)
	}
	if res.Failed > 0 || res.AuditError != "" {
		if res.AuditError != "" {
			fmt.Fprintln(os.Stderr, "cleanlie: audit:", res.AuditError)
		}
		return 1
	}
	return 0
}
