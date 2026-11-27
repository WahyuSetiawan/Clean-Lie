package main

import (
	"github.com/Juragan-Koding/cleanlie/internal/schedule"
)

// ScheduleStatus mengembalikan kondisi timer user systemd.
func (a *App) ScheduleStatus() schedule.Status {
	st, _ := schedule.ReadStatus()
	return st
}

// ScheduleInstall menulis unit lalu mengaktifkan timer. Hanya user timer.
func (a *App) ScheduleInstall(interval string) (schedule.Status, error) {
	exe, err := selfExecutable()
	if err != nil {
		return schedule.Status{}, err
	}
	return schedule.Install(exe, interval, false)
}

// ScheduleRemove menonaktifkan dan menghapus unit timer.
func (a *App) ScheduleRemove() (schedule.Status, error) {
	return schedule.Remove(false)
}
