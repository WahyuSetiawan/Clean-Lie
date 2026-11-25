# Changelog

Semua perubahan penting pada Cleanlie. Format mengikuti
[Keep a Changelog](https://keepachangelog.com/id/1.1.0/) dan versi mengikuti
[Semantic Versioning](https://semver.org/lang/id/).

## [0.5.0] — 2026-11-25

### Ditambahkan — M5 Analisis + Dashboard

- `internal/analysis`: tree disk usage berkedalaman N, file besar, file lama, artefak build,
  dan duplikat (group ukuran → hash 64 KiB → SHA-256). Semua read-only.
- Tree memakai inode set bersama dan `fsx.SkipNames`, jadi hardlink lintas subtree dihitung
  sekali dan `.git` tidak ikut di depth terakhir.
- Binding `AnalysisTree`, `AnalysisOverview`, `AnalysisDuplicates`, `AnalysisRefresh`,
  `AnalysisCancel`, `AnalysisClean`, `DiskStatus`; cache 60 detik; UI hanya mengirim ID.
- Aksi trash analisis memakai guard `clean.Execute` yang sama dengan layar Bersihkan.
- CLI `--analysis-tree` (`--path`, `--depth`), `--analysis-overview`, `--analysis-dups`.
- Layar **Analisis**: 5 tab (Disk usage + breadcrumb/drill-down, File besar, Duplikat,
  File lama, Artefak build), pilih aman, pindah ke trash, buka folder.
- **Dashboard** nyata: kartu Disk/Nix/cache dev/log, diskmap home, top-5 reclaim, tool dev,
  tombol scan ulang; tombol bersihkan memakai alur M2.

### Catatan

- Tree `~` depth 2 **179.182.256.128 B** vs `du --exclude=.git` selisih **0,0013%**;
  `~/.cache` **1,70 s** (<10 s). Duplikat **1.715 grup / 16,70 GiB**, 51,3 s;
  UI menampilkan 100 grup teratas.
- 406–423 error permission pada overview/duplikat dicatat sebagai hitungan, bukan crash.
- Aksi analisis hanya mode trash; gate teks `GC` tetap khusus Nix.
- Versi aplikasi menjadi **0.5.0** (`Makefile`, `flake.nix`).

## [0.4.0] — 2026-11-25

### Ditambahkan — M4 Android

- `internal/android`: inventaris AVD dari `~/.android/avd/*.ini` + `config.ini`
  (nama, API, ABI, system image, ukuran, terakhir dipakai), inventaris paket SDK
  (`system-images`, `ndk`, `build-tools`, `platforms`, `emulator`), deteksi emulator
  berjalan lewat `/proc`, dan saran otomatis AVD >90 hari / system image tidak dipakai.
- Aksi resmi: `avdmanager delete avd -n`, `sdkmanager --sdk_root=... --uninstall`,
  dan wipe headless `emulator -avd ... -wipe-data ... -quit-after-boot`.
- Guard: emulator harus mati, paket yang dipakai AVD tidak boleh di-uninstall, path
  divalidasi di dalam `AVDHome`/SDK, dan fallback hapus folder hanya untuk AVD yang
  tidak dikenali `avdmanager`.
- Binding Wails `AndroidStatus`, `AndroidRefresh`, `AndroidDeleteAVD`,
  `AndroidUninstallPackage`, `AndroidWipeAVD`, `AndroidPreview`; cache status 2 menit.
- Layar **Android** nyata: kartu AVD/SDK/emulator, tabel AVD + system image +
  NDK/build-tools, tombol pilih saran, hapus/wiped per baris dan massal, gate checkbox
  eksplisit, preview perintah, hasil `freed`, dan rescan.
- CLI `--android-status` (JSON) untuk verifikasi tanpa GUI.

### Diubah

- `internal/tools.Run` sekarang memakai `RunEnv`, sehingga aksi Android dapat membawa
  `ANDROID_HOME`/`ANDROID_SDK_ROOT` tanpa mengubah environment global.
- Versi aplikasi menjadi **0.4.0** (`Makefile`, `flake.nix`).

### Catatan

- UI tidak pernah mengirim path: AVD dikirim lewat nama, paket lewat ID inventory.
- Default tidak ada item Android yang dicentang; NDK/build-tools tidak disarankan otomatis.
- Kartu SDK tidak menjumlahkan `cmdline-tools`, `cmake`, `platform-tools` (di luar scope).
- Hapus AVD/SDK bersifat permanen, tidak masuk trash.

## [0.3.0] — 2026-11-25

### Ditambahkan — M3 Nix

- Layar **Nix**: ukuran store (NAR), jumlah path, profile/generasi system, home-manager,
  dan user, closure per generasi, dead paths, journal, serta free/total disk.
- Aksi hapus generasi, GC, dan vacuum journal dengan preview perintah dan gate ketik `GC`.
- `internal/privileged`: eksekusi root via `pkexec` dengan path absolut; tanpa polkit agent
  atau exit 126/127 → `NeedsSudo` + perintah `sudo` siap-copy, bukan kegagalan senyap.
- CLI `--nix-status` dan `--nix-dead` (JSON).
- Cache status/dead 2 menit; invalidasi setelah aksi sukses non-sudo.

### Catatan

- Gate M3: `--nix-dead` **624 path** == `nix-collect-garbage --dry-run` **624 path**
  (mesin target, 2026-11-25).
- Ukuran dead path memakai NAR (`nix path-info --json`), bukan allocated block.
- Tanggal generasi system memakai mtime symlink karena `system.lock` tidak bisa dibaca user.
- Closure dihitung batch `nix path-info -S` (300 path unik per panggilan).

## [0.2.0] — 2026-11-25

### Ditambahkan — M2 Bersihkan

- Layar **Bersihkan** nyata: scan katalog, kategori, checkbox per item, total terpilih,
  dan mode `safe`.
- Eksekusi `gio trash` dengan fallback manual `.trashinfo`; mode permanen hanya lewat
  checkbox eksplisit.
- Guard: path absolut, di dalam root rule, kedalaman sesuai `Rule.Depth`, protected paths,
  dan proses yang sedang berjalan.
- `freed` diukur ulang (`before − after`); status `kept` bila ukuran tidak berubah.
- Audit log `~/.local/state/cleanlie/history.jsonl` (0600) + tampilan error audit.
- Binding scan/cancel/clean/history/open trash/copy text + event progress.
- UI smoke test jsdom opsional (tanpa Node di build aplikasi).

### Diperbaiki

- Fungsi global `go()` menimpa `window.go` milik Wails; diganti menjadi `navigate()`.

## [0.1.0] — 2026-11-25

### Ditambahkan — M1 Engine

- Walker paralel `internal/fsx`: `Lstat`, allocated block (`st_blocks * 512`), dedupe
  hardlink `(dev, ino)`, one-file-system, cancel, progress, top-N heap.
- Katalog rule `internal/rules`: cache dev (npm/pnpm/gradle/cargo/go/uv/dart/composer/
  playwright/cypress dll.), Android SDK/AVD, log/journal, trash, Nix.
- `Plan` read-only + dedupe prioritas + `dropAncestors` + protected paths.
- CLI `--dry-run`, `--dry-run --json`, `--scan-only --path`, `--group`, `--workers`,
  `--version`, `--help`.
- `scripts/verify-sizes.sh` membandingkan item >100 MiB dengan `du` (toleransi 2%).

### Catatan

- Scan mesin target: ±88,4 GiB / 1,4 juta file / 3,5 detik.

## [0.0.1] — 2026-11-25

### Ditambahkan — M0 Scaffold

- Proyek Go + Wails v2 dengan frontend statis `//go:embed` (tanpa build Node).
- Flake NixOS: devShell (Go, Wails, GTK3, WebKitGTK 4.1) + package `cleanlie` dengan
  `makeWrapper` PATH untuk `nix`, `journalctl`, `pkexec`, `gio`, `xdg-open`.
- Shell UI: sidebar 6 layar, routing hash, tema dark/light, dan binding `Diagnostics()`.
- `Makefile`, `scripts/dev.sh`, `wails.json`, README, serta wireframe interaktif
  `docs/wireframe/index.html` sebagai acuan visual.

---

## Rencana berikutnya

- **M6** — Pengaturan, i18n, packaging, dan verifikasi akhir v1.
