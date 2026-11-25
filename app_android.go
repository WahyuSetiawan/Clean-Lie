package main

import (
	"time"

	"github.com/Juragan-Koding/cleanlie/internal/android"
)

// androidCacheTTL membatasi pemindaian AVD/SDK ulang saat layar dibuka lagi.
const androidCacheTTL = 2 * time.Minute

// AndroidStatus mengembalikan daftar AVD dan paket SDK. Hasil di-cache.
func (a *App) AndroidStatus() (android.Status, error) {
	a.androidMu.Lock()
	defer a.androidMu.Unlock()
	if a.androidStatus != nil && time.Since(a.androidStatusAt) < androidCacheTTL {
		return *a.androidStatus, nil
	}
	st := android.ReadStatus(a.baseCtx(), android.Options{})
	a.androidStatus = &st
	a.androidStatusAt = time.Now()
	return st, nil
}

// AndroidRefresh membuang cache status Android.
func (a *App) AndroidRefresh() bool {
	a.invalidateAndroid()
	return true
}

func (a *App) invalidateAndroid() {
	a.androidMu.Lock()
	a.androidStatus = nil
	a.androidMu.Unlock()
}

// AndroidDeleteAVD menghapus AVD berdasarkan nama (bukan path).
func (a *App) AndroidDeleteAVD(name string) (android.Result, error) {
	res, err := android.DeleteAVD(a.baseCtx(), name, android.Options{})
	if err != nil {
		return res, err
	}
	if res.Deleted {
		a.invalidateAndroid()
	}
	return res, nil
}

// AndroidUninstallPackage meng-uninstall paket SDK berdasarkan ID inventory.
func (a *App) AndroidUninstallPackage(id string) (android.Result, error) {
	res, err := android.UninstallPackage(a.baseCtx(), id, android.Options{})
	if err != nil {
		return res, err
	}
	if res.Deleted {
		a.invalidateAndroid()
	}
	return res, nil
}

// AndroidWipeAVD menjalankan wipe data resmi pada AVD yang tidak berjalan.
func (a *App) AndroidWipeAVD(name string) (android.Result, error) {
	res, err := android.WipeAVD(a.baseCtx(), name, android.Options{})
	if err != nil {
		return res, err
	}
	if res.ExitCode == 0 {
		a.invalidateAndroid()
	}
	return res, nil
}

// AndroidPreview menampilkan perintah untuk modal konfirmasi.
func (a *App) AndroidPreview(kind, target string) string {
	return android.Preview(kind, target, android.Options{})
}
