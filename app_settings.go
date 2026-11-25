package main

import (
	"path/filepath"
	"strings"

	"github.com/Juragan-Koding/cleanlie/internal/config"
)

// SettingsView adalah config yang sedang aktif plus lokasi file dan warning
// bila config lama tidak valid.
type SettingsView struct {
	Config  config.Config `json:"config"`
	Path    string        `json:"path"`
	Warning string        `json:"warning,omitempty"`
}

func (a *App) settings() config.Config {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()
	return a.cfg
}

func (a *App) settingsView() SettingsView {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()
	return SettingsView{Config: a.cfg, Path: a.cfgPath, Warning: a.cfgWarning}
}

// Settings membaca preferensi aktif. Config dibaca saat App dibuat, jadi
// binding ini tidak menyentuh disk pada setiap render UI.
func (a *App) Settings() SettingsView {
	return a.settingsView()
}

// SaveSettings menulis config secara atomik, lalu membersihkan cache analisis
// karena ambang batas atau path pengecualian bisa berubah.
func (a *App) SaveSettings(c config.Config) (SettingsView, error) {
	saved, warnings, err := config.Save(c)
	if err != nil {
		return a.settingsView(), err
	}
	a.cfgMu.Lock()
	a.cfg = saved
	a.cfgPath = config.Path()
	a.cfgWarning = ""
	if len(warnings) > 0 {
		a.cfgWarning = warnings[0]
		for _, w := range warnings[1:] {
			a.cfgWarning += "; " + w
		}
	}
	view := SettingsView{Config: saved, Path: a.cfgPath, Warning: a.cfgWarning}
	a.cfgMu.Unlock()
	a.AnalysisRefresh()
	return view, nil
}

// ResetSettings mengembalikan semua preferensi ke default dan menyimpannya.
func (a *App) ResetSettings() (SettingsView, error) {
	return a.SaveSettings(config.Default())
}

// excludedPath memakai aturan prefix yang sama dengan scanner: path yang
// berada di dalam folder pengecualian tidak boleh dihapus.
func excludedPath(path string, excluded []string) bool {
	path = filepath.Clean(path)
	for _, root := range excluded {
		root = filepath.Clean(root)
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
