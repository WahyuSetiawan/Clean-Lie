# Cleanlie M9 — Visibilitas (Insight, Ekspor, Notifikasi)

> Addendum `cleanlie-v1.md`. Status: **selesai** (2026-11-27, rilis 0.9.0).
> Prasyarat: M8 selesai (v0.8.0). Target rilis: **0.9.0**.
> Satu milestone = satu rangkaian commit yang bisa dites tanpa GUI.

## 1. Latar belakang

M8 membuat pembersihan bisa dipercaya, tetapi pengguna belum punya jawaban
untuk pertanyaan yang lebih panjang:

- “Apakah disk saya membaik minggu ini?”
- “Berapa yang benar-benar dibebaskan vs hanya dipindah ke trash?”
- “Bisakah saya mengirim laporan ke orang lain?”
- “Saya menjalankan scan/GC lama; bagaimana tahu kalau sudah selesai?”

Audit `history.jsonl` sudah menyimpan sesi, tetapi hanya dipakai layar Riwayat.
Tidak ada snapshot tren disk, tidak ada ekspor, dan tidak ada notifikasi desktop.

## 2. Tujuan

1. **Insight** — ringkasan 7 hari + tren disk 30 hari dari snapshot harian.
2. **Ekspor** — laporan Markdown/JSON/CSV yang bisa dibaca manusia atau diproses.
3. **Notifikasi** — notifikasi desktop opsional saat pembersihan/GC selesai.

## 3. Non-tujuan

- Auto-clean terjadwal (kandidat M10).
- Cloud sync / upload laporan.
- Notifikasi real-time per item; cukup satu notifikasi per operasi.
- Mengubah audit menjadi database; tetap append-only JSONL.

## 4. Data

### 4.1 Snapshot tren

File: `~/.local/state/cleanlie/trend.jsonl` (XDG state, satu baris per hari).

```json
{"date":"2026-11-27","at":"2026-11-27T11:00:00Z","total":200000000000,"free":80000000000,"used":120000000000}
```

- Satu titik per tanggal; record ulang mengganti titik hari itu.
- Maksimal 90 titik; titik tertua dibuang.
- Tulis atomik (temp + rename), mode 0600.
- Snapshot hanya membaca `statfs("/")`; tidak memindai home/Nix.

### 4.2 Ringkasan minggu

`Week` dihitung dari audit 7 hari terakhir:

| Field | Arti |
|---|---|
| `sessions` | jumlah sesi audit |
| `removed` | item berstatus `removed` |
| `failed` | item berstatus `failed` |
| `movedBytes` | total `bytesBefore` item `removed` (dipindah ke trash) |
| `freedBytes` | total `freed` (hanya permanen/GC yang benar-benar membebaskan) |
| `restorable` | item yang masih bisa di-undo |
| `trashEntries` | jumlah entry trash saat ini |

`movedBytes` dan `freedBytes` dipisah supaya UI tidak berbohong: pindah ke trash
belum membebaskan ruang sampai trash dikosongkan.

### 4.3 Laporan

`insight.Report` berisi versi, waktu generate, disk, week, trend, 10 sesi
terakhir, dan ringkasan trash. Format:

- `md` — laporan manusia (judul, tabel ringkasan, tren, sesi, trash).
- `json` — objek `Report` (integrasi).
- `csv` — baris `section,date,label,bytes,count` untuk spreadsheet.

Laporan ditulis ke `~/.local/state/cleanlie/reports/` dengan nama
`cleanlie-YYYYMMDD-HHMMSS.<ext>`.

## 5. Notifikasi

- `internal/notify` memanggil `notify-send --app-name=Cleanlie --urgency=low`.
- Jika `notify-send` tidak ada atau gagal, operasi tetap sukses (no-op).
- Config baru `notify` (default `true`); toggle di Pengaturan.
- Dipanggil setelah operasi non-dry-run yang selesai: Clean, AnalysisClean,
  DirenvClean, GC, hapus generasi, vacuum journal.
- Isi singkat: judul + jumlah item/byte; tidak memuat path sensitif.

## 6. API binding

| Binding | Input | Output |
|---|---|---|
| `Insight()` | — | `insight.Report` |
| `ExportReport(format)` | `md\|json\|csv` | path file |

`Insight` membaca audit + trend + trash; mencatat snapshot hari ini bila belum ada.
`ExportReport` memakai `Insight`, merender, menulis atomik, mengembalikan path.
UI membuka folder lewat `OpenInFileManager(path)` yang sudah ada.

## 7. CLI

- `--insight` — cetak `insight.Report` JSON.
- `--report FORMAT` — cetak laporan (`md`, `json`, `csv`) ke stdout.
- Keduanya read-only kecuali pencatatan snapshot harian (idempotent).

## 8. UI

- Dashboard kanan: kartu **Minggu ini** — freed, dipindah, sesi, item trash,
  sparkline tren 14 hari, select format + tombol **Ekspor**.
- Riwayat toolbar: tombol **Ekspor laporan** (format sama) memakai binding.
- Pengaturan: toggle **Notifikasi desktop**.
- i18n ID/EN untuk semua string baru.

## 9. Keamanan & batasan

- Laporan tidak pernah berisi isi file; hanya path yang sudah terlihat di UI.
- Export menolak format asing; path output selalu di state dir Cleanlie.
- Snapshot tidak menyentuh home; biaya rendah.
- `notify-send` dipanggil dengan argumen array (bukan shell).
- Tren kosong (instalasi baru) ditampilkan sebagai “belum ada data”, bukan error.

## 10. Acceptance

- `go test ./internal/insight ./internal/notify` hijau.
- `Insight()` mengembalikan week/trend; dua panggilan di hari yang sama tidak
  menambah titik trend.
- `--insight --json | jq .week` valid; `--report md` berisi “Cleanlie”.
- `ExportReport("csv")` membuat file dan CSV valid.
- Toggle notifikasi tersimpan; `notify-send` palsu dipanggil sekali saat Clean.
- UI smoke: kartu Minggu ini terisi, ekspor memanggil binding, path tampil.
- `gofmt`, `go vet`, UI smoke, `nix build` 0.9.0 hijau.
