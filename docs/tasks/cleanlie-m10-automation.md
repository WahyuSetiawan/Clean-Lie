# Cleanlie M10 — Otomasi & Cakupan: Task Breakdown

> Turunan dari [`../plan/cleanlie-m10-automation.md`](../plan/cleanlie-m10-automation.md).
> Satu task = satu commit yang bisa dites.

## Status

| Milestone | Status | Bukti |
|---|---|---|
| **M10** Otomasi & cakupan | ✅ 2026-11-27 | 16 paket Go, UI smoke, `nix build` 0.10.0 |

## Ringkasan

| Task | Isi | Est. | Depends |
|---|---|---|---|
| T10.1 | Aturan kustom: config + `rules.Custom` + UI | 0,4 d | — |
| T10.2 | Hardlink counter di `analysis.FindDuplicates` + UI | 0,15 d | — |
| T10.3 | `internal/schedule` unit user systemd + CLI | 0,35 d | — |
| T10.4 | `--auto-clean` aman (safe + trash + `--yes`) | 0,2 d | T10.1 |
| T10.5 | Binding + kartu Otomasi di Pengaturan | 0,2 d | T10.3 |
| T10.6 | Docs, test, versi 0.10.0 | 0,2 d | semua |
| | **6 task** | **~1,5 d** | |

## T10.1 — Aturan kustom

- `config.CustomRule{Path,Label,Risk,Whole,Depth}` + `Config.CustomRules`.
- `Normalize`: maks 32, expand `~`, absolut, clean, dedupe, risk valid, depth 1–6,
  tolak path yang `rules.IsProtected`.
- `rules.CustomRules([]rules.CustomRule)` → `Rule` group `custom`, priority 10,
  label default basename, `IncludeFiles=true` bila `!Whole`; mapping config → rules
  di `custom_rules.go`.
- UI textarea `custom-rules` (format `path | label | risk | whole`) di Pengaturan;
  `readForm`/`fillSettings`/`DEFAULTS`.
- **Acceptance:** round-trip config; path `~/.ssh` ditolak; 33 aturan dipotong;
  `rules.Plan` memuat group `custom`; UI smoke menyimpan aturan.
- **Files:** `internal/config/config.go`, `internal/rules/custom.go`,
  `app_settings.go`, `frontend/dist/index.html`, `frontend/dist/ui/settings.js`,
  `internal/config/config_test.go`, `internal/rules/custom_test.go`.

## T10.2 — Hardlink counter

- `Duplicates.Hardlinks`, `Duplicates.HardlinkBytes`; naik saat inode sama dilewati.
- UI ringkasan duplikat menampilkan `N hardlink dilewati`.
- **Acceptance:** dua hardlink + satu salinan → hardlink=1, wasted hanya salinan.
- **Files:** `internal/analysis/analysis.go`, `duplicates.go`,
  `analysis_test.go`, `frontend/dist/ui/analysis.js`.

## T10.3 — Schedule

- `internal/schedule`: `Status`, `Paths`, `Render`, `Install`, `Remove`, `ReadStatus`.
- Unit user: `cleanlie-clean.service` + `.timer`; interval `daily|weekly`.
- `systemctl --user daemon-reload` + `enable --now`; `CLEANLIE_NO_SYSTEMCTL=1`
  hanya menulis file.
- CLI `--schedule-install`, `--schedule-remove`, `--schedule-status`,
  `--interval`; `--dry-run` mencetak unit.
- **Acceptance:** render unit benar; install/remove di XDG temp; status JSON.
- **Files:** `internal/schedule/schedule.go`, `schedule_test.go`, `cli.go`,
  `cli_test.go`.

## T10.4 — Auto-clean

- `--auto-clean`: Plan katalog + custom; filter `RiskSafe && !RequiresRoot`;
  `clean.Execute` mode trash; `--yes` wajib (exit 2), `--dry-run` aman.
- Batch audit `auto-<timestamp>`; output JSON bila `--json`.
- **Acceptance:** dry-run tidak mengubah file; tanpa `--yes` exit 2; dengan
  `CLEANLIE_NO_GIO=1` fixture pindah ke trash + audit.
- **Files:** `cli.go`, `cli_test.go`, `internal/clean` (bila perlu batch).

## T10.5 — Binding + GUI Otomasi

- `App.ScheduleStatus/Install/Remove` di `app_schedule.go`.
- Kartu **Otomasi** di Pengaturan: interval, status, tombol pasang/hapus,
  i18n ID/EN.
- UI smoke mock + assertion.
- **Acceptance:** tombol memanggil binding; status tampil; tidak ada aksi hapus.
- **Files:** `app_schedule.go`, `frontend/dist/index.html`,
  `frontend/dist/ui/settings.js`, `frontend/test/ui-smoke.mjs`.

## T10.6 — Docs + rilis

- `docs/implementation/m10-automation.md`, README, CHANGELOG 0.10.0,
  status v1.
- Versi 0.10.0: `Makefile`, `flake.nix`, `package.nix`, `main.go`.
- **Acceptance:** `go test ./...`, `go vet`, UI smoke, `nix build` 0.10.0.

## Definisi selesai (M10)

1. Pengguna bisa menambah folder cache sendiri tanpa mengubah kode.
2. Duplikat hardlink tidak lagi tampak seperti ruang yang bisa dibebaskan.
3. Timer user systemd bisa dipasang/dihapus dari GUI/CLI.
4. Auto-clean hanya `safe` + trash + `--yes`; tidak ada jalur permanen.
5. Semua tes, UI smoke, dan `nix build` 0.10.0 hijau.
