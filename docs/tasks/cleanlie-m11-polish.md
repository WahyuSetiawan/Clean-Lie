# Cleanlie M11 — Polish: Task Breakdown

> Turunan dari [`../plan/cleanlie-m11-polish.md`](../plan/cleanlie-m11-polish.md).
> Satu task = satu commit yang bisa dites.

## Status

| Milestone | Status | Bukti |
|---|---|---|
| **M11** Polish | ✅ 2026-11-27 | 17 paket Go, UI smoke, `nix build` 0.11.0 |

## Ringkasan

| Task | Isi | Est. | Depends |
|---|---|---|---|
| T11.1 | Single-instance lock + main wiring | 0,25 d | — |
| T11.2 | Disk health (`fsx.ClassifyHealth` + chip) | 0,2 d | — |
| T11.3 | i18n label dinamis (helper + modul) | 0,5 d | T11.2 |
| T11.4 | CI test/lint/build | 0,2 d | T11.1–T11.3 |
| T11.5 | Docs, UI smoke, versi 0.11.0 | 0,2 d | semua |
| | **5 task** | **~1,35 d** | |

## T11.1 — Single instance

- `internal/singleinstance/singleinstance.go`: `Acquire`, `Release`, `Path`,
  `ErrRunning`; `flock(LOCK_EX|LOCK_NB)`, PID di file.
- `main.go`: acquire setelah CLI, sebelum `wails.Run`; `ErrRunning` → stderr +
  exit 1.
- **Acceptance:** acquire ganda → `ErrRunning`; release → acquire lagi sukses;
  CLI tidak membuat lock.
- **Files:** `internal/singleinstance/*`, `main.go`.

## T11.2 — Disk health

- `internal/fsx/health.go`: `Health`, `ClassifyHealth`.
- `Volume.Health`; `Statfs` mengisi.
- Dashboard: chip `dash-disk-health`; `renderDisk` memakai health + hint
  terlokalisasi.
- **Acceptance:** tabel batas (`0`, 4%, 6%, 11%, 13%, <2 GiB, <10 GiB); UI smoke
  chip `ok`.
- **Files:** `internal/fsx/health.go`, `health_test.go`, `usage.go`,
  `frontend/dist/index.html`, `frontend/dist/ui/analysis.js`.

## T11.3 — i18n label dinamis

- `window.__tf(key, vars)` di `core.js` + key ID/EN.
- Ganti teks dinamis di dashboard, analisis, Riwayat, preview, Nix, Android,
  direnv, dan settings (status/count/hint).
- `applyLanguage` memicu `refreshDynamic()` untuk layar aktif.
- **Acceptance:** UI smoke ganti EN → hint disk, status Riwayat, dan ringkasan
  analisis berbahasa Inggris; tidak ada `undefined`.
- **Files:** `frontend/dist/ui/*.js`, `frontend/dist/index.html`.

## T11.4 — CI

- `.github/workflows/test.yml` (go, ui, build).
- **Acceptance:** YAML parse; perintah identik dengan lokal; `nix build` hanya
  setelah go+ui hijau.
- **Files:** `.github/workflows/test.yml`.

## T11.5 — Docs + rilis

- `docs/implementation/m11-polish.md`, README, CHANGELOG 0.11.0, status v1.
- Versi 0.11.0: `Makefile`, `flake.nix`, `package.nix`, `main.go`.
- **Acceptance:** semua gate lokal hijau; `nix build` → 0.11.0.

## Definisi selesai (M11)

1. Membuka Cleanlie dua kali tidak membuat dua GUI.
2. Dashboard langsung menunjukkan kesehatan kapasitas disk.
3. Mode English konsisten untuk teks dinamis utama.
4. Setiap push/PR menjalankan test, vet, gofmt, UI smoke, dan build.
5. `nix build` 0.11.0 hijau.
