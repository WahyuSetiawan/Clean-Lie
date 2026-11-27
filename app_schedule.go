package main

import (
	"strings"

	"github.com/Juragan-Koding/cleanlie/internal/schedule"
)

// ScheduleStatus mengembalikan kondisi timer user systemd.
func (a *App) ScheduleStatus() schedule.Status {
	st, _ := schedule.ReadStatus()
	return a.localizeSchedule(st)
}

// ScheduleInstall menulis unit lalu mengaktifkan timer. Hanya user timer.
func (a *App) ScheduleInstall(interval string) (schedule.Status, error) {
	exe, err := selfExecutable()
	if err != nil {
		return schedule.Status{}, err
	}
	st, err := schedule.Install(exe, interval, false)
	return a.localizeSchedule(st), err
}

// ScheduleRemove menonaktifkan dan menghapus unit timer.
func (a *App) ScheduleRemove() (schedule.Status, error) {
	st, err := schedule.Remove(false)
	return a.localizeSchedule(st), err
}

// localizeSchedule menerjemahkan detail status timer tanpa mengubah paket
// schedule (yang tidak tahu bahasa UI).
func (a *App) localizeSchedule(st schedule.Status) schedule.Status {
	if a.settings().Language != "en" {
		return st
	}
	st.Detail = scheduleDetailEN(st.Detail)
	return st
}

func scheduleDetailEN(s string) string {
	switch s {
	case "dry-run: unit tidak ditulis":
		return "dry-run: unit not written"
	case "unit ditulis":
		return "unit written"
	case "unit ditulis; systemctl dilewati (CLEANLIE_NO_SYSTEMCTL=1)":
		return "unit written; systemctl skipped (CLEANLIE_NO_SYSTEMCTL=1)"
	case "timer aktif":
		return "timer active"
	case "dry-run: unit tidak dihapus":
		return "dry-run: unit not removed"
	case "timer dihapus":
		return "timer removed"
	case "timer belum dipasang":
		return "timer not installed"
	case "unit terpasang; status systemctl dilewati":
		return "unit installed; systemctl status skipped"
	case "unit terpasang":
		return "unit installed"
	}
	for _, p := range []struct{ id, en string }{
		{"unit ditulis; daemon-reload gagal: ", "unit written; daemon-reload failed: "},
		{"unit ditulis; enable gagal: ", "unit written; enable failed: "},
	} {
		if strings.HasPrefix(s, p.id) {
			return p.en + strings.TrimPrefix(s, p.id)
		}
	}
	return s
}
