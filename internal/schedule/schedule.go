// Package schedule mengelola user systemd timer untuk auto-clean Cleanlie.
// Hanya unit user (~/.config/systemd/user), tidak pernah system-wide.
package schedule

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ServiceName = "cleanlie-clean.service"
	TimerName   = "cleanlie-clean.timer"

	Daily  = "daily"
	Weekly = "weekly"
)

// Status adalah kondisi unit timer saat ini.
type Status struct {
	Installed   bool   `json:"installed"`
	Enabled     bool   `json:"enabled"`
	Active      bool   `json:"active"`
	Interval    string `json:"interval"`
	ServicePath string `json:"servicePath"`
	TimerPath   string `json:"timerPath"`
	Detail      string `json:"detail,omitempty"`
}

// Dir mengembalikan folder unit user systemd.
func Dir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "systemd", "user")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "systemd", "user")
	}
	return filepath.Join("systemd", "user")
}

// Paths mengembalikan path service dan timer.
func Paths() (service, timer string) {
	dir := Dir()
	return filepath.Join(dir, ServiceName), filepath.Join(dir, TimerName)
}

// NormalizeInterval membatasi nilai ke daily/weekly.
func NormalizeInterval(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", Weekly:
		return Weekly, nil
	case Daily:
		return Daily, nil
	default:
		return "", fmt.Errorf("interval tidak dikenal: %s (pakai daily atau weekly)", v)
	}
}

// Render membuat isi unit. ExecStart selalu memakai --auto-clean --yes.
func Render(execPath, interval string) (service, timer string, err error) {
	interval, err = NormalizeInterval(interval)
	if err != nil {
		return "", "", err
	}
	execPath = strings.TrimSpace(execPath)
	if execPath == "" {
		return "", "", errors.New("path binary kosong")
	}
	if strings.ContainsAny(execPath, "\n\r") {
		return "", "", errors.New("path binary mengandung baris baru")
	}
	quoted := strings.ReplaceAll(execPath, "%", "%%")
	if strings.ContainsAny(quoted, " \t\"'") {
		quoted = `"` + strings.ReplaceAll(quoted, `"`, `\"`) + `"`
	}
	service = `[Unit]
Description=Cleanlie — bersihkan cache aman ke trash
Documentation=man:cleanlie(1)

[Service]
Type=oneshot
ExecStart=` + quoted + ` --auto-clean --yes --json
Nice=10
IOSchedulingClass=idle
NoNewPrivileges=true
PrivateTmp=true
`
	onCalendar := "weekly"
	if interval == Daily {
		onCalendar = "daily"
	}
	timer = `[Unit]
Description=Cleanlie — jadwal auto-clean (` + interval + `)

[Timer]
OnCalendar=` + onCalendar + `
Persistent=true
RandomizedDelaySec=15m
Unit=` + ServiceName + `

[Install]
WantedBy=timers.target
`
	return service, timer, nil
}

// Install menulis unit lalu mengaktifkan timer. Bila systemctl tidak tersedia
// (atau CLEANLIE_NO_SYSTEMCTL=1), unit tetap ditulis dan Detail menjelaskan.
func Install(execPath, interval string, dryRun bool) (Status, error) {
	interval, err := NormalizeInterval(interval)
	if err != nil {
		return Status{}, err
	}
	service, timer, err := Render(execPath, interval)
	if err != nil {
		return Status{}, err
	}
	servicePath, timerPath := Paths()
	st := Status{Interval: interval, ServicePath: servicePath, TimerPath: timerPath}
	if dryRun {
		st.Detail = "dry-run: unit tidak ditulis"
		return st, nil
	}
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o700); err != nil {
		return st, err
	}
	if err := writeAtomic(servicePath, service); err != nil {
		return st, err
	}
	if err := writeAtomic(timerPath, timer); err != nil {
		return st, err
	}
	st.Installed = true
	st.Detail = "unit ditulis"
	if os.Getenv("CLEANLIE_NO_SYSTEMCTL") == "1" {
		st.Detail = "unit ditulis; systemctl dilewati (CLEANLIE_NO_SYSTEMCTL=1)"
		return st, nil
	}
	if _, err := runSystemctl("daemon-reload"); err != nil {
		st.Detail = "unit ditulis; daemon-reload gagal: " + err.Error()
		return st, nil
	}
	if _, err := runSystemctl("enable", "--now", TimerName); err != nil {
		st.Detail = "unit ditulis; enable gagal: " + err.Error()
		return st, nil
	}
	now, _ := ReadStatus()
	now.Interval = interval
	now.Detail = "timer aktif"
	return now, nil
}

// Remove menonaktifkan timer lalu menghapus unit. File yang tidak ada tidak
// dianggap error.
func Remove(dryRun bool) (Status, error) {
	servicePath, timerPath := Paths()
	st := Status{ServicePath: servicePath, TimerPath: timerPath}
	if dryRun {
		st.Detail = "dry-run: unit tidak dihapus"
		return st, nil
	}
	if os.Getenv("CLEANLIE_NO_SYSTEMCTL") != "1" {
		_, _ = runSystemctl("disable", "--now", TimerName)
	}
	for _, p := range []string{timerPath, servicePath} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return st, err
		}
	}
	if os.Getenv("CLEANLIE_NO_SYSTEMCTL") != "1" {
		_, _ = runSystemctl("daemon-reload")
	}
	st.Detail = "timer dihapus"
	return st, nil
}

// ReadStatus membaca keberadaan unit dan (bila ada systemctl) status enable/active.
func ReadStatus() (Status, error) {
	servicePath, timerPath := Paths()
	st := Status{ServicePath: servicePath, TimerPath: timerPath, Interval: Weekly}
	_, errS := os.Stat(servicePath)
	_, errT := os.Stat(timerPath)
	st.Installed = errS == nil && errT == nil
	if !st.Installed {
		st.Detail = "timer belum dipasang"
		return st, nil
	}
	if data, err := os.ReadFile(timerPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "OnCalendar=") {
				if strings.TrimPrefix(line, "OnCalendar=") == "daily" {
					st.Interval = Daily
				} else {
					st.Interval = Weekly
				}
			}
		}
	}
	if os.Getenv("CLEANLIE_NO_SYSTEMCTL") == "1" {
		st.Detail = "unit terpasang; status systemctl dilewati"
		return st, nil
	}
	if out, err := runSystemctl("is-enabled", TimerName); err == nil {
		st.Enabled = strings.TrimSpace(out) == "enabled"
	}
	if out, err := runSystemctl("is-active", TimerName); err == nil {
		st.Active = strings.TrimSpace(out) == "active"
	}
	st.Detail = "unit terpasang"
	return st, nil
}

func writeAtomic(path, data string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cleanlie-unit-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func runSystemctl(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
