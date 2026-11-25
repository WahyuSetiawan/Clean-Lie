# Cleanlie

Disk cleaner & analyzer GUI untuk NixOS. Go + Wails v2, UI statis (tanpa Node).

> Status: **M2 — bersihkan**. Scan katalog, dry-run CLI, dan eksekusi hapus (trash/perm)
> sudah jalan lewat layar Bersihkan. Aksi root (GC/generasi) masuk M3. Lihat
> [`docs/tasks/cleanlie-v1.md`](docs/tasks/cleanlie-v1.md),
> [`docs/implementation/m1-engine.md`](docs/implementation/m1-engine.md), dan
> [`docs/implementation/m2-clean.md`](docs/implementation/m2-clean.md).

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

Wrapper sudah membawa PATH untuk `nix`, `journalctl`, `pkexec`, `gio`, `xdg-open`
(karena `pkexec` membuang environment sesi).

## Mode headless (tanpa GUI)

```sh
./bin/cleanlie --dry-run              # laporan teks, tidak menghapus
./bin/cleanlie --dry-run --json       # JSON untuk integrasi / verifikasi
./bin/cleanlie --scan-only --path DIR # ukur satu direktori
./bin/cleanlie --version

make verify                           # bandingkan ukuran vs `du` (toleransi 2%)
```

Semua mode di atas read-only. Eksekusi hapus ada di GUI (layar **Bersihkan**): pilih item,
konfirmasi dialog, lalu `gio trash` (default) atau permanen dengan checkbox. Hasil menyimpan
`freed` terukur dan audit log di `~/.local/state/cleanlie/history.jsonl`.

## Test UI (opsional, tanpa build Node)

```sh
cd frontend/test && npm install && npm test
```

jsdom menjalankan `frontend/dist/index.html` dengan binding Go tiruan: scan → pilih → gate
permanen → trash → hasil. Ini juga menjaga agar `window.go` (binding Wails) tidak tertimpa.

## Struktur

```
main.go                 # wails.Run + embed frontend/dist + dispatch CLI
cli.go                  # --dry-run / --scan-only / --json
app.go                  # binding Go → window.go.main.App (Diagnostics)
app_clean.go            # ScanJunk/CancelScan/Clean/History + event progress
internal/fsx/           # walker paralel (blok disk, hardlink, one-fs) + usage/statfs
internal/rules/         # katalog rule + Plan read-only (linux, devtools, android, nix)
internal/clean/         # guard, trash freedesktop, deteksi proses, eksekutor, audit
internal/tools/         # jalankan perintah resmi dengan preview + timeout
frontend/dist/          # UI statis (sumber desain: docs/wireframe)
frontend/test/          # smoke test jsdom (opsional, bukan bagian build)
flake.nix               # devShell + package
docs/plan|tasks|wireframe|implementation
```

## Prinsip keamanan (v1)

1. Dry-run dulu; eksekusi hanya dari plan yang sudah ditampilkan.
2. Hapus default ke **trash** (`gio trash`), `permanent` harus eksplisit.
3. Whitelist root + protected list; item di luar aturan ditolak.
4. Cache tool yang prosesnya berjalan di-skip, bukan dipaksa.
5. GC & hapus generasi Nix wajib konfirmasi ketik `GC`.
6. `freed` diukur ulang (before − after), bukan diasumsikan; audit log di
   `~/.local/state/cleanlie/history.jsonl`.

## Catatan Wayland/Hyprland

WebKitGTK kadang butuh `WEBKIT_DISABLE_COMPOSITING_MODE=1` (sudah diset di devShell).
Kalau window hitam/berkedip, jalankan dengan env itu. Tanpa polkit agent, aksi root
otomatis jatuh ke dialog perintah `sudo` siap-copy.

## Lisensi

Belum ditentukan.
