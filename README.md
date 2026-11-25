# Cleanlie

Disk cleaner & analyzer GUI untuk NixOS. Go + Wails v2, UI statis (tanpa Node).

> Status: **M0 — scaffold**. Window, navigasi, dan binding diagnostik sudah jalan.
> Engine scan/clean baru masuk M1–M2. Lihat [`docs/tasks/cleanlie-v1.md`](docs/tasks/cleanlie-v1.md).

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

## Struktur

```
main.go                 # wails.Run + embed frontend/dist
app.go                  # binding Go → window.go.main.App (M0: Diagnostics)
frontend/dist/          # UI statis (sumber desain: docs/wireframe)
flake.nix               # devShell + package
docs/plan|tasks|wireframe
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
