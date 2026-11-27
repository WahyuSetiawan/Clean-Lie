# Cleanlie M10 — Otomasi & Cakupan

> Addendum `cleanlie-v1.md`. Status: **selesai** (v0.10.0, 2026-11-27).
> Prasyarat: M9 selesai (v0.9.0). Target rilis: **0.10.0**.
> Satu milestone = commit yang bisa dites tanpa GUI.

## 1. Latar belakang

Cleanlie sudah bisa memindai katalog tetap, tetapi tiga celah membuatnya belum
“jalan sendiri”:

- Folder cache milik pengguna (mis. `~/work/cache`) tidak ada di katalog.
- Duplikat yang berupa **hardlink** sudah tidak dihitung dua kali, tetapi UI tidak
  memberi tahu berapa banyak yang dilewati; reflink belum dilaporkan.
- Pembersihan harus dibuka manual; tidak ada timer user systemd.

M10 menambahkan cakupan yang bisa dikonfigurasi dan otomasi yang **aman secara
default**: hanya item `safe`, hanya ke trash, dan selalu lewat guard yang sama.

## 2. Tujuan

1. **Aturan kustom** dari config, tervalidasi, muncul di layar Bersihkan.
2. **Hardlink-aware** duplicate report: hitung hardlink yang dilewati dan jangan
   mengklaim ruang yang sebenarnya dipakai bersama.
3. **Timer user systemd** opsional: pasang/hapus/status dari GUI atau CLI.
4. **Auto-clean** non-interaktif yang hanya menyentuh item `safe`, mode trash,
   dengan `--yes` eksplisit dan audit batch.

## 3. Non-tujuan

- Auto-clean permanen atau item `warn`/`danger`.
- Timer system-wide/root; hanya user timer.
- Deteksi reflink tingkat extent (butuh FIEMAP); M10 hanya menandai hardlink.
- Editor aturan kustom berbasis form kompleks; cukup satu baris per aturan.

## 4. Aturan kustom

### 4.1 Config

```json
"customRules": [
  {"path": "~/work/cache", "label": "cache kerja", "risk": "safe", "whole": false, "depth": 1}
]
```

- Maks 32 aturan; path di-`~`-expand, absolut, clean, dedupe.
- `risk` hanya `safe`, `warn`, `danger`; default `warn`.
- `whole=true` = seluruh path satu item; `whole=false` = anak depth 1 (default).
- Path yang dilindungi (`rules.IsProtected`) ditolak saat normalisasi.
- Aturan kustom memakai group `custom`; label default nama folder.

### 4.2 UI

- Textarea di Pengaturan, satu baris:
  `path | label | risk | whole`
- `whole` opsional (`whole`/`folder`); baris kosong diabaikan.
- Baris tidak valid dilewati dengan warning di layar Pengaturan.

## 5. Hardlink / reflink

- `FindDuplicates` sudah melewati inode yang sama. Tambahkan `Hardlinks` dan
  `HardlinkBytes` ke hasil, dihitung saat inode duplikat dilewati.
- `Wasted` tetap hanya untuk salinan yang benar-benar terpisah.
- UI ringkasan duplikat menampilkan `N hardlink dilewati` bila > 0.
- Reflink tidak dideteksi; dokumentasikan sebagai batasan (extent sharing tidak
  terlihat dari `stat`).

## 6. Timer + auto-clean

### 6.1 Unit

`~/.config/systemd/user/cleanlie-clean.service`:

```ini
[Unit]
Description=Cleanlie — bersihkan cache aman ke trash

[Service]
Type=oneshot
ExecStart=/path/cleanlie --auto-clean --yes --json
Nice=10
IOSchedulingClass=idle
NoNewPrivileges=true
PrivateTmp=true
```

`cleanlie-clean.timer`: `OnCalendar=weekly` (atau `daily`), `Persistent=true`,
`RandomizedDelaySec=15m`.

### 6.2 CLI

- `--auto-clean` — pindai katalog + aturan kustom, pilih item `safe` dari group
  `app`/`dev.*`/`custom` yang tidak butuh root, eksekusi mode **trash**. Tanpa `--yes` → exit 2; `--dry-run` aman.
- `--schedule-install [--interval daily|weekly]` — tulis unit, `daemon-reload`,
  `enable --now` timer. `--dry-run` hanya mencetak unit.
- `--schedule-remove` — disable + hapus unit.
- `--schedule-status` — JSON status unit/timer.
- `CLEANLIE_NO_SYSTEMCTL=1` membuat instalasi hanya menulis file (untuk test).

### 6.3 GUI

- Kartu **Otomasi** di Pengaturan: pilih interval, tombol Pasang/Hapus timer,
  status terpasang/aktif, dan penjelasan “hanya item aman ke trash”.
- Binding `ScheduleStatus`, `ScheduleInstall`, `ScheduleRemove`.
- Tidak ada tombol “jalankan auto-clean sekarang” di GUI; aksi manual tetap
  lewat layar Bersihkan agar pengguna melihat rencana.

## 7. Keamanan

- Auto-clean **tidak pernah** memakai `ModePermanent`.
- Hanya `RiskSafe`, `RequiresRoot=false`; `trash` rule (`warn`) tidak ikut.
- Path `Excluded` dan `Protected` tetap ditolak oleh `clean.Execute`.
- Timer berjalan sebagai user, bukan root.
- `--schedule-install` tidak menulis di luar `$XDG_CONFIG_HOME/systemd/user`.
- `--auto-clean` menulis audit batch `auto-<timestamp>`.

## 8. Acceptance

- `go test ./internal/rules ./internal/analysis ./internal/schedule` hijau.
- Config round-trip custom rules; path protected ditolak; maks 32.
- Duplikat dengan hardlink: `Hardlinks > 0`, `Wasted` tidak menghitung hardlink.
- `--auto-clean --dry-run` tidak menghapus; tanpa `--yes` exit 2.
- `--schedule-install --dry-run` mencetak unit tanpa menulis.
- `--schedule-install --interval daily` menulis 2 file; status JSON benar;
  `--schedule-remove` menghapus keduanya.
- UI smoke: aturan kustom tersimpan; kartu Otomasi memasang/menghapus timer.
- `gofmt`, `go vet`, UI smoke, `nix build` 0.10.0 hijau.
