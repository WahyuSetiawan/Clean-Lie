package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/analysis"
	"github.com/Juragan-Koding/cleanlie/internal/android"
	"github.com/Juragan-Koding/cleanlie/internal/clean"
	"github.com/Juragan-Koding/cleanlie/internal/config"
	"github.com/Juragan-Koding/cleanlie/internal/direnv"
	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
	"github.com/Juragan-Koding/cleanlie/internal/trash"
)

// App adalah jembatan Go ↔ UI: diagnostik, scan/clean, Nix, Android,
// analisis, dan preferensi pengguna.
type App struct {
	ctx       context.Context
	version   string
	commit    string
	startedAt time.Time

	scanMu     sync.Mutex
	scanCancel context.CancelFunc
	scanItems  map[string]rules.Item
	cleanMu    sync.Mutex

	cfgMu      sync.RWMutex
	cfg        config.Config
	cfgPath    string
	cfgWarning string

	nixMu       sync.Mutex
	nixStatus   *nix.Status
	nixStatusAt time.Time
	nixDeadMu   sync.Mutex
	nixDead     *nix.Dead
	nixDeadAt   time.Time

	androidMu       sync.Mutex
	androidStatus   *android.Status
	androidStatusAt time.Time

	analysisMu          sync.Mutex
	analysisCancel      context.CancelFunc
	analysisTrees       map[string]analysis.Tree
	analysisTreesAt     map[string]time.Time
	analysisOverviews   map[string]analysis.Overview
	analysisOverviewsAt map[string]time.Time
	analysisDups        map[string]analysis.Duplicates
	analysisDupsAt      map[string]time.Time
	analysisTargets     map[string]analysisTarget
	direnvMu            sync.Mutex
	direnvCancel        context.CancelFunc
	direnvReport        *direnv.Report
	direnvKey           string
	direnvAt            time.Time
	direnvTargets       map[string]clean.Target

	trashMu         sync.Mutex
	trashReport     *trash.Report
	trashEntries    map[string]trash.Entry
	trashAt         time.Time
	trashGen        int
	trashSizeCancel context.CancelFunc
}

func NewApp(version, commit string) *App {
	c, warn := config.Load()
	return &App{
		version: version, commit: commit,
		cfg: c, cfgPath: config.Path(), cfgWarning: warn,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startedAt = time.Now()
}

func (a *App) shutdown(_ context.Context) {}

// Diagnostics dipakai UI untuk memastikan binding Go ↔ JS hidup dan
// untuk menampilkan status lingkungan (polkit, nix store, config dir).
type Diagnostics struct {
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	GoVersion    string `json:"goVersion"`
	Platform     string `json:"platform"`
	Home         string `json:"home"`
	ConfigDir    string `json:"configDir"`
	NixStore     bool   `json:"nixStore"`
	PolkitAgent  bool   `json:"polkitAgent"`
	PolkitDetail string `json:"polkitDetail"`
	// AdminKind menjelaskan cara aksi root diprompt: "polkit" (Linux) atau
	// "osascript" (macOS).
	AdminKind string `json:"adminKind"`
	StartedAt string `json:"startedAt"`
}

func (a *App) Diagnostics() Diagnostics {
	home, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	polkit, detail := detectPolkitAgent()

	_, nixErr := os.Stat("/nix/store")

	return Diagnostics{
		Version:      a.version,
		Commit:       a.commit,
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
		Home:         home,
		ConfigDir:    configDir,
		NixStore:     nixErr == nil,
		PolkitAgent:  polkit,
		PolkitDetail: detail,
		AdminKind:    adminKind(),
		StartedAt:    a.startedAt.Format(time.RFC3339),
	}
}

func adminKind() string {
	if runtime.GOOS == "darwin" {
		return "osascript"
	}
	return "polkit"
}

// detectPolkitAgent mencari cara prompt admin yang tersedia. Di macOS,
// osascript selalu ada sehingga dialog admin native bisa dipakai.
func detectPolkitAgent() (bool, string) {
	if runtime.GOOS == "darwin" {
		return true, "dialog admin macOS (osascript) — setara pkexec"
	}
	out, err := exec.Command("pgrep", "-af", "polkit.*agent|hyprpolkitagent").Output()
	if err != nil {
		return false, "tidak terdeteksi — aksi root akan memakai perintah sudo siap-copy"
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if line == "" {
		return false, "tidak terdeteksi — aksi root akan memakai perintah sudo siap-copy"
	}
	return true, line
}
