# Cleanlie M9 — Visibilitas: Task Breakdown

> Turunan dari [`../plan/cleanlie-m9-visibility.md`](../plan/cleanlie-m9-visibility.md).
> Satu task = satu commit yang bisa dites. Module: `github.com/Juragan-Koding/cleanlie`.

## Status

| Milestone | Status | Bukti |
|---|---|---|
| **M9** Visibilitas (insight, ekspor, notifikasi) | ✅ 2026-11-27 | rilis 0.9.0 |

## Aturan main

- T9.1 → T9.2 → T9.3 wajib berurutan; T9.4 boleh paralel setelah T9.1.
- UI menunggu binding T9.3; jangan menambah path dari JS.
- Snapshot tren append-only secara logis (satu titik/hari), tulis atomik.
- Notifikasi tidak boleh menggagalkan operasi utama.
- Commit kecil, prefix: `feat(m9):`, `fix(m9):`, `test(m9):`, `docs(m9):`.

## Ringkasan

| Task | Isi | Est. | Depends |
|---|---|---|---|
| T9.1 | `internal/insight`: snapshot, trend, week, report | 0,35 d | — |
| T9.2 | `internal/notify` + config `notify` + hook | 0,2 d | — |
| T9.3 | Binding `Insight`/`ExportReport` + CLI `--insight`/`--report` | 0,25 d | T9.1 |
| T9.4 | Dashboard kartu Minggu ini + sparkline + ekspor | 0,3 d | T9.3 |
| T9.5 | Riwayat ekspor + toggle notifikasi + i18n | 0,2 d | T9.2, T9.4 |
| T9.6 | Test, docs, versi 0.9.0 | 0,2 d | semua |
| | **6 task** | **~1,5 d** | |

## T9.1 — `internal/insight`

- `Point{Date,At,Total,Free,Used}`, `Week{...}`, `Report{Version,GeneratedAt,Disk,Week,Trend,Sessions,Trash}`.
- `DefaultPath()` = `<state>/cleanlie/trend.jsonl`; `Record` satu titik/hari,
  maks 90, tulis atomik; `Read` dedupe + urut.
- `Collect(ctx, version, now)` baca `fsx.Statfs("/")`, `clean.ReadSessions("",50)`,
  `trash.List`, trend; catat snapshot bila perlu.
- `Build` memfilter sesi 7 hari, hitung `MovedBytes`/`FreedBytes`, isi trash count.
- `Render(report, "md|json|csv")`.
- **Acceptance:** dua `Record` di hari yang sama → 1 titik; report md/json/csv valid;
  sesi >7 hari tidak masuk week.
- **Files:** `internal/insight/insight.go`, `insight_test.go`; tambah
  `RemovedBytes` di `clean.Session`.

## T9.2 — Notifikasi

- `internal/notify`: `Available`, `Send(ctx,title,body)` via `notify-send`
  (array argumen, timeout 2 s, error tidak fatal).
- Config `notify` default `true`; toggle Pengaturan.
- Hook di `Clean`, `AnalysisClean`, `DirenvClean`, `NixCollectGarbage`,
  `NixDeleteGenerations`, `VacuumJournal` setelah non-dry-run.
- **Acceptance:** fake `notify-send` di PATH menerima satu panggilan; config
  `notify=false` tidak memanggil; operasi tetap sukses bila binary hilang.
- **Files:** `internal/notify/notify.go`, `notify_test.go`, `app_notify.go`,
  `internal/config/config.go`, `app_clean.go`, `app_analysis.go`, `app_direnv.go`,
  `app_nix.go`.

## T9.3 — Binding + CLI

- `App.Insight() (insight.Report, error)`, `App.ExportReport(format) (string, error)`.
- Export ke `<state>/cleanlie/reports/cleanlie-<timestamp>.<ext>`, mode 0600.
- CLI `--insight`, `--report FORMAT` (stdout; error format → exit 2).
- **Acceptance:** `--insight --json | jq .week`; `--report md` memuat “Cleanlie”;
  `ExportReport("csv")` menghasilkan file valid; format asing ditolak.
- **Files:** `app_insight.go`, `cli.go`, `cli_test.go`.

## T9.4 — Dashboard

- Kartu **Minggu ini** di kolom kanan: freed, moved, sessions, trash entries.
- Sparkline 14 titik; label rentang tanggal.
- Select format + tombol Ekspor; hasil path + tombol buka folder.
- `analysis.js`: panggil `Insight` paralel dengan disk/Nix; render; ekspor.
- **Acceptance:** UI smoke mock `Insight` → angka terisi; klik Ekspor →
  `ExportReport` dipanggil; path tampil.
- **Files:** `frontend/dist/index.html`, `frontend/dist/ui/analysis.js`,
  `frontend/dist/ui/app.css` (sparkline), `frontend/test/ui-smoke.mjs`.

## T9.5 — Riwayat + Pengaturan + i18n

- Tombol **Ekspor laporan** di toolbar Riwayat (format dari select dashboard).
- Toggle **Notifikasi desktop** di Pengaturan (config `notify`).
- i18n ID/EN untuk dashboard week/export, settings notify, error export.
- **Acceptance:** toggle tersimpan di `SaveSettings`; UI smoke settings tetap hijau;
  tombol Riwayat memanggil `ExportReport`.
- **Files:** `frontend/dist/index.html`, `frontend/dist/ui/history.js`,
  `frontend/dist/ui/settings.js`, `frontend/test/ui-smoke.mjs`.

## T9.6 — Docs + rilis

- `docs/implementation/m9-visibility.md`, README (insight/export/notify + CLI),
  CHANGELOG 0.9.0, tautan plan/tasks v1.
- Versi 0.9.0: `Makefile`, `flake.nix`, `package.nix`, `main.go` (`0.9.0-dev`).
- **Acceptance:** `go test ./...`, `go vet`, UI smoke, `nix build` 0.9.0.

## Definisi selesai (M9)

1. Dashboard menjawab “minggu ini” dengan angka freed vs moved yang jujur.
2. Tren disk 30 hari tersimpan lokal dan tidak duplikat per hari.
3. Laporan bisa diekspor md/json/csv dan dibuka dari UI/CLI.
4. Notifikasi desktop opsional, tidak pernah menggagalkan operasi.
5. Semua tes, UI smoke, dan `nix build` 0.9.0 hijau.

## Catatan implementasi (2026-11-27)

- Semua task T9.1–T9.6 selesai dalam satu rangkaian commit; versi 0.9.0.
- `insight.Collect` sengaja tidak gagal bila `trash.List`/tren error; `statfs` dan
  audit tetap error karena inti laporan.
- `notify` default `true`; config lama tetap terbaca `true` karena `Load` memulai
  dari `Default()` sebelum unmarshal.
- `libnotify` ditambahkan ke wrapper PATH agar `notify-send` tersedia di build Nix.
- Deviasi kecil: tidak ada endpoint/CLI untuk membuka folder laporan; UI memakai
  `OpenInFileManager` yang sudah ada atau menampilkan path.
- Verifikasi: 15 paket Go hijau, `gofmt`/`go vet` bersih, UI smoke hijau,
  `--insight`/`--report` hijau, `nix build` 0.9.0, `scripts/verify-trash.sh` tetap hijau.
