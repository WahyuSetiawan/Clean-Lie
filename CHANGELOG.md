# Changelog

Semua perubahan penting pada Cleanlie. Format mengikuti
[Keep a Changelog](https://keepachangelog.com/id/1.1.0/) dan versi mengikuti
[Semantic Versioning](https://semver.org/lang/id/).

## [0.10.0] — 2026-11-27

### Ditambahkan — M10 Otomasi & Cakupan

- **Aturan kustom** di Pengaturan dan `config.json` (`customRules`): path, label,
  risk, `whole`, depth; maks 32, path protected ditolak, group `custom`.
- **Timer user systemd**: `--schedule-install`, `--schedule-remove`,
  `--schedule-status`, `--interval daily|weekly`; kartu Otomasi di Pengaturan.
  Unit hanya menulis `~/.config/systemd/user/` dan berjalan sebagai user.
- **`--auto-clean`**: hanya group `app`, `dev.*`, `custom` dengan risk `safe`, tanpa
  root, selalu mode trash, wajib `--yes` (kecuali `--dry-run`), audit batch `auto-*`.
- **Hardlink-aware duplicate**: `hardlinks`/`hardlinkBytes` di hasil analisis;
  `wasted` tidak menghitung hardlink; UI menampilkan jumlah yang dilewati.
- Test `internal/schedule`, aturan kustom, hardlink, `--auto-clean`, CLI schedule,
  dan UI smoke (textarea aturan, kartu Otomasi).

### Catatan

- Versi aplikasi menjadi **0.10.0** (`Makefile`, `flake.nix`, `package.nix`, `main.go`).
- Auto-clean sengaja tidak menyentuh `/var/tmp`/log sistem dan tidak pernah memakai
  `ModePermanent`. Timer user berhenti saat logout bila lingering tidak diaktifkan.
- Reflink belum dideteksi (butuh FIEMAP); hanya hardlink yang dihitung.

## [0.9.0] — 2026-11-27

### Ditambahkan — M9 Visibilitas: insight, ekspor, notifikasi

- `internal/insight`: snapshot disk harian di `trend.jsonl` (satu titik per tanggal,
  maks 90 hari, tulis atomik), ringkasan 7 hari dari audit (dibebaskan vs dipindah ke
  trash), dan laporan `md`/`json`/`csv`.
- Dashboard **Minggu ini**: freed, moved, sesi, trash, sparkline tren, dan tombol
  **Ekspor** ke `~/.local/state/cleanlie/reports/`.
- Tombol **Ekspor laporan** di Riwayat dengan format Markdown/JSON/CSV.
- Binding `Insight()` dan `ExportReport(format)`; CLI `--insight` dan
  `--report md|json|csv` (read-only, aman untuk integrasi).
- `internal/notify`: notifikasi desktop `notify-send` (best-effort) setelah Clean,
  AnalysisClean, DirenvClean, GC, hapus generasi, dan vacuum journal. Config baru
  `notify` (default aktif) + toggle di Pengaturan.
- `Session.RemovedBytes` untuk memisahkan ruang yang dipindah ke trash dari yang
  benar-benar dibebaskan.
- Test `internal/insight`, `internal/notify`, binding export, CLI, dan UI smoke
  (kartu minggu, sparkline, ekspor dashboard/riwayat, toggle notifikasi).

### Catatan

- Versi aplikasi menjadi **0.9.0** (`Makefile`, `flake.nix`, `package.nix`, `main.go`).
- Snapshot tren hanya membaca `statfs("/")`; tidak memindai home/Nix. Laporan tidak
  pernah memuat isi file, hanya path yang sudah terlihat di UI.

## [Unreleased]

### Ditambahkan — dukungan macOS
- Build & jalankan di macOS (arm64) lewat devShell `aarch64-darwin`; Wails memakai
  WKWebView sistem, tag `webkit2_41` hanya dipakai di Linux.
- Aksi root di macOS memakai dialog admin `osascript` (pengganti `pkexec`);
  pembatalan prompt jatuh ke perintah `sudo` siap-copy. `Diagnostics.adminKind`
  memberi tahu UI cara prompt yang dipakai.
- Katalog clean macOS: `~/Library/Caches`, `~/Library/Logs`, Saved Application
  State, Homebrew cache, cache dev di `~/Library` (Go, gopls, uv, pip, pnpm,
  Playwright, Cypress), serta Xcode DerivedData/iOS DeviceSupport/Archives dan
  CoreSimulator. Grup baru `dev.macos`.
- Guard proses membaca `ps` di macOS (tanpa `/proc`); deteksi emulator Android
  juga lewat `ps`.
- Trash memakai `~/.Trash` (Finder) dan tanpa metadata `.trashinfo`.
- Rule systemd journal/`/var/log`/`/var/tmp` dan vacuum journal tidak dimuat di
  macOS; kartu Journal disembunyikan, layar Nix tetap berfungsi.
- Path lindung macOS: `/System`, `/Library`, `/Applications`, `/private`,
  `/opt/homebrew`, `~/Library` (kecuali Caches/Logs/Developer), Keychains,
  Application Support, Containers, Mobile Documents, Preferences.

### Ditambahkan
- Output flake `overlays.default` dan `nixosModules.default`, plus `package.nix`
  yang bisa dipanggil dengan nixpkgs konsumen. Cleanlie sekarang bisa dipakai dari
  flake NixOS/Home Manager lain lewat `pkgs.cleanlie` atau modul siap pakai.
- Workflow `.github/workflows/cachix.yml`: build `.#cleanlie` di `main` dan push
  closure ke cache Cachix `juragankoding-cachix` (secret `CACHIX_AUTH_TOKEN`).

## [0.8.0] — 2026-11-27

### Ditambahkan — M8 Trust: Trash Manager, Undo, Riwayat, Preview

- `internal/trash`: baca trash freedesktop (home + volume `.Trash-$uid`), parsing
  `.trashinfo` yang aman, deteksi orphan, ukuran allocated dengan dedupe hardlink,
  restore tanpa menimpa (gio URI hanya untuk home; volume lewat `os.Rename`), hapus
  permanen dengan cek mount, dan `Empty` bergerbang `KOSONGKAN`/`EMPTY`.
- `internal/clean/history.go`: audit batch + aksi (`clean`, `restore`, `trash-delete`,
  `trash-empty`), pembacaan sesi 50 terakhir/200 item, rotasi 8 MiB, dan aturan
  `RestorableEntry` (hanya item `mode=trash`, `status=removed`, punya jejak trash).
- Binding baru: `TrashList`, `TrashRefresh`, `TrashRestore`, `TrashDelete`,
  `TrashEmpty`, `HistorySessions`, `UndoSession`, `PreviewCleanup`, `PreviewTrash`.
  UI hanya mengirim ID; restore selalu membaca ulang `.trashinfo` (audit hanya hint).
- `internal/preview`: preview file/folder read-only (20 entri, batas 1,5 s, symlink
  tidak diikuti, ukuran allocated, flag blocked/truncated/timed out).
- Layar **Riwayat** (layar ke-8) dengan tab **Aktivitas** (sesi, filter, detail,
  undo parsial) dan **Trash** (cari, pilih, restore, hapus, kosongkan, ukuran
  streaming, badge tidak-restorable). Tombol **Lihat isi** di Bersihkan/Analisis/Trash.
- Modal hasil `Clean` menampilkan **Undo sesi ini** bila batch punya item restorable.
- CLI: `--trash-list` (`--no-size`), `--trash-restore`, `--trash-delete`,
  `--trash-empty`, `--history` (`--limit`), `--undo`; tindakan destruktif butuh
  `--yes`, `--dry-run` tetap aman. `scripts/verify-trash.sh` menguji alur penuh di
  HOME/XDG sementara dengan `CLEANLIE_NO_GIO=1`.
- Test: `internal/trash`, `internal/preview`, `internal/clean` (audit/sesi), binding
  app, CLI, dan UI smoke (sesi, trash, preview, restore, kosongkan, undo).

### Catatan

- Versi aplikasi menjadi **0.8.0** (`Makefile`, `flake.nix`, `package.nix`, `main.go`).
- Restore tidak pernah menimpa file, tidak me-rename otomatis, dan tidak membuat parent
  yang hilang; konflik dilaporkan per item. `EXDEV` adalah error, bukan copy diam-diam.

## [0.7.0] — 2026-11-26

### Ditambahkan — M7 direnv
- `internal/direnv`: pemindai read-only folder `.direnv` per proyek. Pengguna
  menambahkan folder proyek; Cleanlie berhenti di `.direnv`, mengukur allocated
  (symlink tidak diikuti), dan melewati `.git`/`node_modules`/artefak build.
  Cache global `~/.cache/direnv` dan `~/.cache/nix-direnv` ikut diukur sebagai
  `kind=cache`.
- Config baru `direnvRoots` (maks 32, dinormalisasi `~`/absolut/clean).
- Binding `DirenvScan`, `DirenvCancel`, `DirenvRefresh`, `DirenvAddRoot`,
  `DirenvRemoveRoot`, `DirenvClean`; cache scan 60 detik dan UI hanya mengirim ID.
- Layar **direnv**: pilih folder lewat dialog native (atau ketik path), scan/batal,
  ringkasan ukuran, daftar proyek + cache, pilih aman (lewati proyek aktif < 7 hari),
  pindah ke trash lewat guard `clean.Execute` yang sama.
- CLI `--direnv-scan` (JSON; `--path DIR` atau root dari config).
- Test `internal/direnv`, `app_direnv_test.go`, dan alur UI smoke
  tambah→scan→pilih→trash.

### Diubah — UI dipecah per modul

- `frontend/dist/index.html` (150 KB) dipecah menjadi shell markup + `dist/ui/app.css` dan script per
  modul: `demo`, `diagnostics`, `core`, `cleanup`, `nix`, `android`, `screens`, `analysis`, `settings`.
  Tetap statis, tanpa bundler; urutan `<script>` = urutan dependensi.
- `frontend/test/ui-smoke.mjs` meng-inline aset `ui/*` sebelum dijalankan di jsdom.
- Test baru `frontend_embed_test.go` memastikan setiap `src`/`href` di `index.html` benar-benar
  ter-embed (mencegah UI blank bila aset lupa di-`git add` sebelum `nix build`).
- Versi aplikasi menjadi **0.7.0** (`Makefile`, `flake.nix`).

### Diperbaiki — viewport Wayland
- Di Wayland (Hyprland), GTK3/WebKitGTK bisa melaporkan `innerWidth`/`innerHeight`
  negatif setelah window di-tile/resize; UI jadi bertumpuk dan sisa window hitam.
  Binary sekarang otomatis memakai XWayland (`GDK_BACKEND=x11`) bila `WAYLAND_DISPLAY`
  dan `DISPLAY` tersedia. Set `GDK_BACKEND=wayland` untuk memaksa Wayland.

## [0.6.0] — 2026-11-25

### Ditambahkan — M6 Settings + packaging

- `internal/config`: config persisten di `~/.config/cleanlie/config.json` (XDG), tulis
  atomik, fallback default + warning bila file hilang/rusak. Field: tema, bahasa,
  tampilkan item berisiko, mode hapus, auto-centang item aman, target journal, ambang
  file besar/lama, root scan tambahan, dan path dikecualikan.
- Binding `Settings`, `SaveSettings`, `ResetSettings`; perubahan langsung membersihkan
  cache analisis. Path pengecualian juga menjadi guard kedua di `Clean`.
- Layar **Pengaturan** nyata: simpan/reset, tag path dikecualikan, status polkit, path config.
- Tema **system/light/dark** dengan `prefers-color-scheme`, fokus keyboard, reduced motion.
- i18n **ID/EN** untuk shell, Pengaturan, Dashboard, dan Analisis; ganti bahasa tanpa reload.
- Ambang analisis dan root scan tambahan dipakai oleh `AnalysisOverview`; exclusion dipakai
  oleh scan katalog, tree, overview, dan duplikat.
- `packaging/cleanlie.desktop` + ikon SVG; flake memasang keduanya ke `$out/share`.
- README: instalasi flake, desktop entry, pengaturan, restore trash, polkit, batasan.

### Catatan

- Versi aplikasi menjadi **0.6.0** (`Makefile`, `flake.nix`).
- i18n saat ini mencakup shell/settings/dashboard/analysis; beberapa label dinamis dari
  layar M2–M4 masih berbahasa Indonesia (akan diterjemahkan bertahap).
- Desktop entry memakai `StartupWMClass=cleanlie`; asosiasi ikon di compositor bergantung
  pada WM class yang dilaporkan WebKitGTK.

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

- **M9** — Visibilitas: ekspor laporan, notifikasi, tren disk, dan ringkasan mingguan.
- **M10** — Otomasi/cakupan: systemd timer, aturan kustom, dan dedupe hardlink.
- **M11** — Polish: i18n penuh, single-instance, CI test/lint, dan disk health.
