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
	"github.com/Juragan-Koding/cleanlie/internal/config"
	"github.com/Juragan-Koding/cleanlie/internal/direnv"
	"github.com/Juragan-Koding/cleanlie/internal/fsx"
	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/privileged"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
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
  cleanlie --version

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
