# M2 — Eksekusi bersih + layar Bersihkan

Status: ✅ 2026-11-25 · Branch `feat/m2-clean` · Lanjutan dari
[`m1-engine.md`](m1-engine.md)

M2 menutup jarak dari "laporan dry-run" ke "benar-benar membebaskan ruang": guard berlapis,
trash freedesktop yang bisa di-restore, deteksi proses, verifikasi `freed`, audit log, dan
layar Bersihkan yang memakai data scan nyata.

## Alur data

```
klik "Bersihkan" / navigasi ke layar
        │
        ▼
App.ScanJunk(scanPath?) ──► rules.Plan(DefaultRules)  (read-only)
        │                       │
        │◄──────────────────────┘  Report + cache map[id]rules.Item
        ▼
UI: applyReport() → tabel + badge + pilih default (safe)
        │
        ▼  CleanRequest{IDs, Mode, ConfirmPermanent}
App.Clean() ──► clean.Execute(targets, opts)
        │            ├─ Validate (root + depth + protected)
        │            ├─ proses berjalan? → skipped
        │            ├─ Command (tool resmi) → fallback trash
        │            ├─ ukur ulang → freed = before − after
        │            └─ AppendAudit(history.jsonl)
        ▼
UI: laporan hasil + "Buka trash" + "Salin log"
```

**Invariant:** UI hanya mengirim **ID**, tidak pernah path. `App.Clean` mengambil path dari
hasil scan terakhir (`scanItems`). Scan baru membatalkan cache lama; ID tak dikenal ditolak.

## Guard berlapis

| Lapis | Lokasi | Menolak |
|---|---|---|
| Path absolut | `clean.Validate` | path relatif/kosong |
| Root katalog | `clean.Validate` | item di luar `Item.Root` |
| Kedalaman | `clean.Validate` | item lebih dalam dari `Item.Depth` (default 1) |
| Root utuh | `clean.Validate` | menghapus root rule tanpa `Whole` |
| Protected | `rules.IsProtected` | `/`, `/nix/store`, `~/.ssh`, `~/.config`, home root, dll. |
| Proses hidup | `clean.BlockedReason` | cache Brave/Firefox/Chromium/JetBrains/emulator |
| Whitelist perintah | `rules` | perintah hanya dari katalog, bukan input UI |

`Item.Depth` adalah deviasi kecil dari plan §6.2: sistem image Android berada di
`system-images/<api>/<vendor>/<abi>` sehingga "anak langsung" akan menolak item yang sah.
Guard sekarang membatasi **kedalaman relatif terhadap root**, bukan hanya `Dir(path) == root`.
Test `TestDefaultCatalogValidates` memastikan seluruh 151 item katalog nyata lolos
(`CLEAN_TEST_SCAN=1`).

## Trash

`MoveToTrash(path)`:

1. `gio trash -- <path>` (freedesktop, restore-able, menangani mount lain).
2. Gagal → fallback manual: pindah ke `$XDG_DATA_HOME/Trash/files`, tulis `.trashinfo`
   (`[Trash Info]`, `Path=` ter-escape per segmen, `DeletionDate=`).
3. Beda filesystem → `ErrCrossDevice`; UI menawarkan hapus permanen (checkbox wajib).

Kalau `.trashinfo` gagal ditulis, file dikembalikan ke lokasi asal — tidak ada file
"hilang" tanpa metadata restore. Test `TestTrashRestoreWithGio` (gated
`CLEAN_TEST_REAL_TRASH=1`) membuktikan hasil fallback bisa dipulihkan dengan
`gio trash --restore trash:///...` (gio butuh URI, bukan path).

## Eksekutor

`clean.Execute(ctx, []Target, Request, Options)`:

- Mode `trash` (default) atau `permanent`.
- Status per item: `removed`, `failed`, `kept`, `skipped`, `planned`.
- Item dengan `Command` menjalankan perintah resmi dulu (timeout 10 menit); bila gagal dan
  `FallbackOK`, lanjut trash. Perintah sukses tapi ukuran tidak turun → `kept` (bukan klaim palsu).
- Kegagalan satu item tidak menghentikan item lain; error dikumpulkan per item.
- `before` diukur dari `Item.Bytes`; `after` dari `fsx.Usage`; `freed = max(0, before-after)`.
- `RequiresRoot` tanpa jalur root (M3) ditandai, bukan dieksekusi diam-diam.

## Deteksi proses

`procs.go` membaca `/proc/*/comm` (nama process di-truncate 15 karakter oleh kernel) dan
mencocokkan aturan:

| Rule | Process |
|---|---|
| browser | brave, firefox, chrome, chromium, msedge |
| JetBrains | idea, pycharm, webstorm, goland, studio |
| Android | emulator, qemu-system-* |
| Node/Java | node, java (hanya untuk cache terkait) |

Item yang terblokir → `skipped` + alasan yang tampil di laporan.

## Audit

`~/.local/state/cleanlie/history.jsonl` (mengikuti `XDG_STATE_HOME`), satu JSON per baris,
mode 0600, direktori 0700. Berisi `startedAt`, `mode`, `freed`, dan daftar item
(`path`, `status`, `freed`, `detail`). `ReadAudit(limit)` melewati baris korup agar satu
penulisan setengah jadi tidak mematahkan riwayat. UI memakai `History()` untuk "Salin log".

## UI Bersihkan

`frontend/dist/index.html` tetap satu file tanpa build Node. Blok M2:

- `ensureScan()` dipanggil saat masuk layar (menggantikan data demo `ITEMS`).
- `applyReport()` mengisi `ITEMS`, `selected` (hanya `safe`), dan badge kategori nyata.
- Event `scan:progress` memperbarui footer + tombol Cancel; `scan:done` menutup state.
- Dialog konfirmasi menampilkan jumlah item, perkiraan ukuran, mode, dan preview perintah.
- Hapus permanen: tombol OK terkunci sampai checkbox "saya mengerti" dicentang.
- Hasil: `freed` aktual, item gagal/alasan, tombol buka trash dan salin log.
- Setelah eksekusi sukses, cache scan di-invalidate dan UI rescans otomatis.

### Gotcha: `window.go` tidak boleh ditimpa

Wireframe memakai fungsi navigasi global bernama `go`, sedangkan binding Wails hidup di
`window.go.main.App`. Keduanya menempati nama global yang sama: binding Wails adalah module
yang dievaluasi setelah script halaman, jadi `window.go = {}` menimpa fungsi `go` (navigasi
mati); sebaliknya bila binding sudah ada lebih dulu, `function go()` menimpa binding (integrasi
diam-diam jatuh ke mode demo). Fungsi navigasi sudah diganti menjadi `navigate()`; smoke test
menjaga agar `window.go` tetap berisi binding.

## Verifikasi

```sh
nix develop -c bash -c 'gofmt -w . && go vet -tags desktop,production,webkit2_41 ./... \
  && go test -tags desktop,production,webkit2_41 ./... && make build'

CLEAN_TEST_REAL_TRASH=1 go test -tags desktop,production,webkit2_41 \
  -run TestTrashRestoreWithGio -v ./internal/clean

CLEAN_TEST_SCAN=1 go test -tags desktop,production,webkit2_41 \
  -run TestDefaultCatalogValidates -v ./internal/clean

cd frontend/test && npm install && npm test   # jsdom: scan→pilih→gate→trash→hasil
```

Hasil: 5 paket Go hijau, `bin/cleanlie` terbangun, UI smoke hijau, restore trash nyata
berhasil. `scripts/verify-sizes.sh` tetap dipakai untuk memastikan ukuran tidak berubah.

## Yang sengaja belum ada

- Eksekusi root (GC/generasi/journal vacuum) → M3 lewat `internal/privileged`.
- Cache browser per profil dan saran AVD tak terpakai → setelah M4.
- Undo otomatis dari trash → cukup petunjuk path; restore lewat file manager.
