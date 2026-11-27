# M9 — Visibilitas: insight, ekspor, notifikasi

Status: **selesai** (2026-11-27, rilis 0.9.0).
Rencana: [`../plan/cleanlie-m9-visibility.md`](../plan/cleanlie-m9-visibility.md).
Task: [`../tasks/cleanlie-m9-visibility.md`](../tasks/cleanlie-m9-visibility.md).

## Kenapa

M8 membuat pembersihan bisa dipercaya, tetapi pengguna masih harus menjawab
pertanyaan jangka panjang secara manual: berapa yang benar-benar dibebaskan
minggu ini, bagaimana tren disk, dan bagaimana mengirim laporan. M9 menutup celah
itu tanpa menambah jalur hapus baru.

## File

| File | Isi |
|---|---|
| `internal/insight/insight.go` | `Point`, `Week`, `Report`; `Record`/`Read` tren; `Collect`; `Build`; `Render` md/json/csv |
| `internal/insight/insight_test.go` | satu titik/hari, dedupe, week 7 hari, 3 format, snapshot idempoten |
| `internal/notify/notify.go` | `Available`, `Send` via `notify-send` (argumen array, timeout 2 s) |
| `internal/notify/notify_test.go` | binary palsu, argumen, no-op saat tidak ada |
| `internal/clean/history.go` | `Session.RemovedBytes` (dipisah dari `Freed`) |
| `app_insight.go` | binding `Insight`, `ExportReport` (tulis atomik ke state dir) |
| `app_notify.go` | hook notifikasi best-effort untuk Clean/Analysis/Direnv/Nix |
| `internal/config/config.go` | field `notify` (default `true`) |
| `cli.go` | `--insight`, `--report md\|json\|csv` |
| `frontend/dist/ui/analysis.js` | kartu Minggu ini, sparkline, ekspor dashboard |
| `frontend/dist/ui/history.js` | ekspor laporan dari Riwayat |
| `frontend/dist/ui/settings.js` | toggle notifikasi + i18n ID/EN |

## Alur data

```
statfs("/") ──┐
audit JSONL ──┼─> insight.Collect ──> Report ──> dashboard / Render(md|json|csv)
trash.List ───┘        │
                       └─> Record(trend.jsonl): 1 titik per tanggal, maks 90
```

`Week` memakai audit 7 hari terakhir. `MovedBytes` dijumlahkan dari
`Session.RemovedBytes` (item `status=removed`), sedangkan `FreedBytes` hanya dari
`Freed` yang sudah diukur ulang. Ini mencegah UI mengklaim ruang yang baru
dipindah ke trash sebagai “dibebaskan”.

## Keputusan implementasi

1. **Tren = snapshot harian**, bukan pembacaan audit. Audit tidak menyimpan
   pemakaian disk, dan menghitung ulang dari audit akan salah untuk pemakaian
   non-Cleanlie.
2. **Satu titik per tanggal**; `Record` membaca file, membuang tanggal yang sama,
   menulis ulang atomik (temp + rename, 0600). Maks 90 titik agar file kecil.
3. **`Collect` tidak gagal karena trend/trash**: kegagalan mencatat tren diabaikan;
   `trash.List` error menghasilkan `Report.Trash` kosong. `statfs` dan audit tetap
   error karena keduanya inti laporan.
4. **Laporan di state dir**, bukan di folder pilihan user, agar tidak ada path
   output arbitrer dari UI. Format asing ditolak sebelum menulis.
5. **CSV datar** (`section,date,label,bytes,count`) supaya bisa langsung dibuka di
   spreadsheet; JSON memakai `Report` penuh.
6. **Notifikasi best-effort**: `notify-send` dipanggil dengan array argumen; jika
   binary hilang/gagal, operasi utama tetap sukses. Config `notify` default aktif.
7. **Hook hanya setelah non-dry-run** dan hanya pada operasi yang selesai
   (`Clean`, `AnalysisClean`, `DirenvClean`, GC, hapus generasi, vacuum journal).
8. **`RemovedBytes` ditambahkan ke `Session`** dan dihitung di `groupSessions`
   supaya kompatibel dengan audit lama (field absen = 0).
9. **Sparkline sederhana** (SVG polyline) agar tidak menambah dependensi chart.
10. **Ekspor dari Riwayat** memakai binding yang sama dengan dashboard; format
    dipilih di toolbar, bukan path.

## Verifikasi

- `go test -tags desktop,production,webkit2_41 ./...` → 15 paket hijau.
- `go vet -tags desktop,production,webkit2_41 ./...` bersih; `gofmt -l .` kosong.
- `./result/bin/cleanlie --insight --json` → `week.days=7`, `trend` satu titik/hari.
- `./result/bin/cleanlie --report md` → memuat “Cleanlie” dan “Tren disk”;
  `--report csv` → header `section,date,label,bytes,count`.
- UI smoke: kartu minggu (`3` sesi, freed terisi), sparkline, ekspor dashboard
  (`md`) dan Riwayat (`json`), toggle notifikasi tersimpan.
- `nix build .#cleanlie` → `cleanlie-0.9.0`.

## Batasan

- Tren baru terisi saat `Insight`/`--insight`/`--report` dipanggil; tidak ada
  daemon harian. Setelah beberapa hari tidak membuka aplikasi, grafik punya celah.
- Notifikasi tidak muncul di sesi tanpa D-Bus/notification daemon (mis. SSH);
  ini sengaja no-op.
- Laporan memuat path yang terlihat di UI; tidak ada redaksi otomatis.
- `notify` pada config lama yang belum punya field akan terbaca `false` sampai
  pengguna menyimpan Pengaturan (default baru tetap `true`).
