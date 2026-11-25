package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/nix"
	"github.com/Juragan-Koding/cleanlie/internal/rules"
)

// App adalah jembatan Go ↔ UI. Untuk M0 isinya hanya diagnostik;
// modul scan/clean/nix/android masuk di M1+.
type App struct {
	ctx       context.Context
	version   string
	commit    string
	startedAt time.Time

	scanMu     sync.Mutex
	scanCancel context.CancelFunc
	scanItems  map[string]rules.Item
	cleanMu    sync.Mutex

	nixMu       sync.Mutex
	nixStatus   *nix.Status
	nixStatusAt time.Time
	nixDeadMu   sync.Mutex
	nixDead     *nix.Dead
	nixDeadAt   time.Time
}

func NewApp(version, commit string) *App {
	return &App{version: version, commit: commit}
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
	StartedAt    string `json:"startedAt"`
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
		StartedAt:    a.startedAt.Format(time.RFC3339),
	}
}

// detectPolkitAgent mencari agent polkit di sesi ini. Hyprland tidak
// menjalankan agent secara default; tanpa agent, pkexec tidak memunculkan prompt.
func detectPolkitAgent() (bool, string) {
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
