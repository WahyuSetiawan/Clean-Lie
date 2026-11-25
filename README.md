# Cleanlie

Disk cleaner & analyzer GUI untuk NixOS. Go + Wails v2, UI statis (tanpa Node).

> Status: **M6 — Settings + packaging**. Scan katalog, dry-run CLI, eksekusi hapus
> (trash/perm), audit, layar Nix (store, generasi, dead paths, GC, vacuum journal,
> fallback pkexec/sudo), layar Android (AVD/SDK, saran >90 hari, hapus/wipe/uninstall),
> layar Analisis (tree, file besar, duplikat, file lama, artefak), Dashboard nyata,
> Pengaturan persisten, tema system/light/dark, dan i18n ID/EN sudah jalan. Lihat
> [`docs/tasks/cleanlie-v1.md`](docs/tasks/cleanlie-v1.md),
> [`docs/implementation/`](docs/implementation/) (m1–m6), dan
> [`CHANGELOG.md`](CHANGELOG.md).

## Kenapa

Mesin ini punya `/nix/store` 82 GB (≈34 GB dead paths), cache tool dev 24.8 GB,
Android SDK/AVD 28.2 GB, dan journal 4 GB. Cleanlie menyatukan pembersihan itu dengan
dry-run + trash (bisa restore), bukan `rm -rf` buta.

## Jalankan (dev)

```sh
nix develop        # go, wails, gtk3, webkitgtk 4.1, pkg-config
make dev           # wails dev (hot reload)
make build         # bin/cleanlie
make test
```

Atau tanpa shell interaktif:

```sh
nix develop -c make build
./bin/cleanlie
```

## Build paket

```sh
nix build .#cleanlie
./result/bin/cleanlie
```

Hasil build juga memasang desktop entry dan ikon:
`result/share/applications/cleanlie.desktop` dan
`result/share/icons/hicolor/scalable/apps/cleanlie.svg`. Untuk memasang ke profil:

```sh
nix profile install .#cleanlie
# lalu jalankan "Cleanlie" dari launcher Hyprland
```

Wrapper sudah membawa PATH untuk `nix`, `journalctl`, `pkexec`, `gio`, `xdg-open`
(karena `pkexec` membuang environment sesi).

## Mode headless (tanpa GUI)

```sh
./bin/cleanlie --dry-run              # laporan teks, tidak menghapus
./bin/cleanlie --dry-run --json       # JSON untuk integrasi / verifikasi
./bin/cleanlie --scan-only --path DIR # ukur satu direktori
./bin/cleanlie --nix-status           # store, generasi, journal (JSON; ~8 s)
./bin/cleanlie --nix-dead             # dry-run GC: jumlah + ukuran dead path (JSON; ~32 s)
./bin/cleanlie --android-status       # AVD + paket SDK (JSON; ~0,14 s)
./bin/cleanlie --analysis-tree         # tree disk usage (JSON; ~5 s; --path, --depth)
./bin/cleanlie --analysis-overview     # file besar/lama/artefak (JSON; ~19 s)
./bin/cleanlie --analysis-dups         # file duplikat (JSON; ~51 s)
./bin/cleanlie --version

make verify                           # bandingkan ukuran vs `du` (toleransi 2%)
```

Semua mode di atas read-only. Eksekusi hapus ada di GUI (layar **Bersihkan**): pilih item,
konfirmasi dialog, lalu `gio trash` (default) atau permanen dengan checkbox. Layar **Nix**
menghapus generasi/GC/vacuum journal lewat pkexec bila ada polkit agent, atau menampilkan
perintah `sudo` siap-copy bila tidak ada. Layar **Android** menghapus AVD lewat
`avdmanager`, paket SDK lewat `sdkmanager`, dan wipe data lewat `emulator -wipe-data`
(headless), semuanya dengan gate checkbox dan tanpa mengirim path dari UI. Hasil menyimpan
`freed` terukur dan audit log di `~/.local/state/cleanlie/history.jsonl`.

Gate M3: `--nix-dead` dan `nix-collect-garbage --dry-run` harus memberi jumlah path yang sama
(624 pada mesin target, 2026-11-25).

Gate M4: hapus AVD lewat fixture harus menghilangkan folder `.avd` + `.ini` dan memberi
`freed > 0`; pada mesin target `test_devices` 5,01 GiB terdeteksi, sedangkan saran otomatis
hanya `system-images;android-35;google_apis;x86_64` (3,50 GiB, 154 hari, tidak dipakai).

Layar **Analisis** dan **Dashboard** read-only saat memindai. Tree `~` depth 2 pada mesin
target 179.182.256.128 B (selisih 0,0013% dari `du --exclude=.git`), `~/.cache` 1,7 s;
duplikat 1.715 grup/16,70 GiB (tampil 100 grup teratas). Tombol trash di analisis memakai
guard yang sama dengan layar Bersihkan; UI hanya mengirim ID hasil analisis, bukan path.

## Test UI (opsional, tanpa build Node)

```sh
cd frontend/test && npm install && npm test
```

jsdom menjalankan `frontend/dist/index.html` (aset `dist/ui/*` di-inline otomatis) dengan binding Go tiruan: scan → pilih → gate
permanen → trash → hasil → layar Nix → gate `GC` → hasil GC → layar Android → pilih saran →
gate hapus/uninstall → hasil → wipe → hasil → dashboard → analisis (tree, drill-down, file
besar, duplikat, artefak). Ini juga menjaga agar `window.go` (binding Wails) tidak tertimpa.

## Struktur

```
main.go                 # wails.Run + embed frontend/dist + dispatch CLI
cli.go                  # --dry-run / --scan-only / --json / Nix / Android / analisis
app.go                  # binding Go → window.go.main.App (Diagnostics + cache)
app_clean.go            # ScanJunk/CancelScan/Clean/History + event progress
app_nix.go              # NixStatus/NixDeadPaths/NixPreview/aksi + cache 2 menit
app_android.go          # AndroidStatus/Refresh/DeleteAVD/Uninstall/Wipe/Preview
app_analysis.go         # tree/overview/duplikat, cache 60 s, target ID, trash analisis
app_settings.go         # Settings/SaveSettings/ResetSettings + guard path dikecualikan
internal/fsx/           # walker paralel (blok disk, hardlink, one-fs) + usage/statfs
internal/rules/         # katalog rule + Plan read-only (linux, devtools, android, nix)
internal/clean/         # guard, trash freedesktop, deteksi proses, eksekutor, audit
internal/nix/           # generasi, closure, dead paths, aksi GC/vacuum
internal/android/       # parser AVD/SDK + aksi avdmanager/sdkmanager/wipe
internal/analysis/      # tree, file besar/lama, artefak, duplikat (read-only)
internal/config/        # config.json: tema, bahasa, ambang, exclusion, scan roots
internal/privileged/    # pkexec dengan path absolut, fallback sudo siap-copy
internal/tools/         # jalankan perintah resmi dengan preview + timeout
frontend/dist/          # UI statis (sumber desain: docs/wireframe)
frontend/test/          # smoke test jsdom (opsional, bukan bagian build)
packaging/              # desktop entry + ikon SVG
flake.nix               # devShell + package (desktop entry/ikon di $out/share)
docs/plan|tasks|wireframe|implementation
```

## Prinsip keamanan (v1)

1. Dry-run dulu; eksekusi hanya dari plan yang sudah ditampilkan.
2. Hapus default ke **trash** (`gio trash`), `permanent` harus eksplisit.
3. Whitelist root + protected list; item di luar aturan ditolak.
4. Cache tool yang prosesnya berjalan di-skip, bukan dipaksa.
5. GC & hapus generasi Nix wajib konfirmasi ketik `GC`; generasi current tidak pernah bisa dipilih.
6. `freed` diukur ulang (before − after), bukan diasumsikan; audit log di
   `~/.local/state/cleanlie/history.jsonl`.
7. Aksi root memakai `pkexec` dengan path absolut bila ada polkit agent; tanpa agent,
   perintah `sudo` siap-copy ditampilkan dan tidak dieksekusi diam-diam.
8. Aksi Android memakai perintah resmi (`avdmanager`, `sdkmanager`, `emulator`), menolak
   saat emulator berjalan, dan memvalidasi path di dalam `~/.android/avd`/SDK. UI mengirim
   nama AVD atau ID paket, bukan path.
9. Path di **Pengecualian** tidak dipindai dan tidak bisa dihapus, termasuk item lama yang
   masih ada di cache scan (guard kedua di `Clean`).

## Pengaturan

Preferensi disimpan di `~/.config/cleanlie/config.json` (XDG) dan dibaca saat app mulai.
File yang hilang/rusak jatuh ke default dengan warning di layar Pengaturan. Field yang tersedia:
tema (`system`/`light`/`dark`), bahasa (`id`/`en`), tampilkan item berisiko, mode hapus default,
auto-centang item aman, target journal, ambang file besar (MB), ambang file lama (hari), root scan
tambahan, dan path yang dikecualikan. Tombol **Simpan pengaturan** menulis atomik; **Kembalikan
default** mengembalikan semuanya.

## Catatan Wayland/Hyprland

WebKitGTK kadang butuh `WEBKIT_DISABLE_COMPOSITING_MODE=1` (sudah diset di devShell).
Kalau window hitam/berkedip, jalankan dengan env itu. Tanpa polkit agent, aksi root
otomatis jatuh ke dialog perintah `sudo` siap-copy.

## Lisensi

Belum ditentukan.
