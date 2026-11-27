# M8 — Trust: Trash Manager, Undo, Riwayat, Preview (implementasi)

Status: **selesai** 2026-11-27 · versi **0.8.0** · branch `feat/m8-trust`.

M8 membuat pembersihan bisa dipercaya dan dibatalkan: isi trash freedesktop
dibaca ulang (home + volume), item bisa dipulihkan tanpa menimpa file, setiap
aksi punya batch audit yang bisa di-undo, dan isi file/folder bisa dilihat
sebelum dipulihkan atau dihapus. UI hanya mengirim ID; path tidak pernah datang
dari JavaScript.

## Kenapa

Sebelum M8, Cleanlie sudah memindahkan item ke trash dan menulis audit, tetapi
pengguna tidak bisa melihat isi trash, memulihkan satu item, atau mengembalikan
satu sesi. Audit hanya catatan; `.trashinfo` adalah sumber kebenaran. Tanpa
pembacaan ulang metadata, restore berisiko menimpa file atau salah path.

## Isi

| File | Peran |
|---|---|
| `internal/trash/trash.go` | `Root`, `Entry`, `Report`, `EntryID`, `DiscoverRoots` (home + `.Trash-$uid`), validasi nama/path |
| `internal/trash/list.go` | baca `.trashinfo`, normalisasi path, orphan, alasan tidak-restorable, batas 5.000 |
| `internal/trash/size.go` | ukur allocated paralel + dedupe hardlink (`fsx.UsageSeen`) |
| `internal/trash/restore.go` | restore aman: validasi, cek protected/konflik/parent, gio home, `os.Rename` volume, `EXDEV` error |
| `internal/trash/remove.go` | hapus permanen per item, cek mount asing, `Empty` gate `KOSONGKAN`/`EMPTY` |
| `internal/trash/resolve.go` | cari entry berdasarkan path asal (home dulu, lalu volume) |
| `internal/clean/history.go` | `Batch`/`Action` audit, sesi 50×200, rotasi, `RestorableEntry` |
| `internal/preview/preview.go` | preview read-only 20 entri, 1,5 s, symlink tidak diikuti |
| `app_history.go` | `TrashList/Refresh/Restore/Delete/Empty`, `HistorySessions`, `UndoSession`, cache + event |
| `app_preview.go` | `PreviewCleanup`/`PreviewTrash` dari ID |
| `frontend/dist/ui/history.js` | layar Riwayat: tab Aktivitas + Trash, search, bulk, undo, gate |
| `frontend/dist/ui/preview.js` | modal preview + tombol buka folder |
| `cli.go` | `--trash-list/--no-size`, `--trash-restore`, `--trash-delete`, `--trash-empty`, `--history`, `--undo` |
| `scripts/verify-trash.sh` | uji penuh di HOME/XDG sementara + `CLEANLIE_NO_GIO=1` |
| `frontend/test/ui-smoke.mjs` | mock binding baru + alur Riwayat/Trash/Preview/Undo |

## Alur data

```
Buka Riwayat
  → HistorySessions(50)  (baca audit + arsip .1)
  → TrashList
       DiscoverRoots → List (metadata + .trashinfo)
       → tampilkan metadata cepat
       → SizeEntries background (trash:size batch, trash:done)

Pulihkan
  → TrashRestore(ids) → resolveTrashIDs (cache, re-list sekali bila asing)
      → baca ulang .trashinfo → cek protected/konflik/parent
      → home: gio trash:///<name> (kecuali CLEANLIE_NO_GIO=1), verifikasi
      → volume: os.Rename; EXDEV = gagal, bukan copy
      → hapus .trashinfo sisa, audit batch restore-

Undo sesi
  → UndoSession(batch) → ReadSessionEntries (baca file audit penuh)
      → RestorableEntry: mode=trash, status=removed, punya jejak trash
      → Resolve(path) → Restore; item permanen/perintah dilewati
```

## Keputusan implementasi

- **`Entry.ID = sha256(root + NUL + name)[:12]`.** UI/CLI tidak pernah mengirim
  path. `resolveTrashIDs` melakukan re-list sekali bila ID tidak ada di cache
  (mis. file manager lain mengubah trash).
- **`.trashinfo` dibaca ulang saat restore.** Audit hanya hint; path asal,
  `DeletionDate`, dan nama file diverifikasi ulang. Ini mencegah audit basi
  memulihkan ke path yang salah.
- **Restore tidak menimpa dan tidak me-rename otomatis.** Konflik = error per
  item. Parent yang hilang juga error (tidak `MkdirAll` diam-diam).
- **Cek protected menargetkan tujuan, bukan parent.** Parent boleh `$HOME`
  (yang ada di daftar protected), sedangkan file tujuan tidak.
- **`gio trash:///<name>` hanya untuk home trash.** Volume trash memakai
  `os.Rename` karena URI gio tidak menangani `.Trash-$uid` di mount lain.
- **Orphan tetap terlihat.** File tanpa `.trashinfo` masuk daftar dengan
  `Restorable=false`, `InfoPath=""`, alasan “metadata .trashinfo hilang”; bisa
  dihapus tetapi tidak dipulihkan.
- **Ukuran streaming.** Metadata tampil instan; ukuran diukur background dengan
  generation guard. `Bytes=-1` berarti belum diukur/gagal, bukan 0.
- **Audit batch + rotasi.** Batch `<UTC>-<hex>`; rotasi 8 MiB sekali sebelum
  operasi tulis (bukan dry-run). Baca ekor 4 MiB/10.000 baris; sesi tertua
  dibuang bila ekor terpotong.
- **Undo membaca file audit penuh.** `SessionPage` yang terpotong tidak boleh
  membuat undo kehilangan item.
- **Preview tidak pernah mengikuti symlink** dan dibatasi 1,5 s/20 entri agar
  folder besar tidak membekukan UI.
- **CLI destruktif butuh `--yes`.** Tanpa `--yes` exit 2 dan tidak ada
  perubahan; `--dry-run` tetap aman. `--trash-empty --yes` meneruskan
  `KOSONGKAN`.

## Verifikasi

- `go test ./internal/... .` — semua paket hijau (termasuk `trash`, `preview`,
  `clean`).
- `go test -tags desktop,production,webkit2_41 ./...` — hijau.
- `cd frontend/test && npm test` — alur Riwayat/Trash/Preview/Undo hijau.
- `scripts/verify-trash.sh ./bin/cleanlie` — list → restore → undo → delete →
  empty pada 4 fixture, tanpa menyentuh trash asli.
- `./bin/cleanlie --trash-list --json | jq .` valid;
  `--trash-empty` tanpa `--yes` exit 2.
- `./bin/cleanlie --history --limit 5 --json | jq .sessions` valid.
- `nix build .#cleanlie` — versi 0.8.0.

## Batasan

- Trash pada filesystem yang tidak mendukung `.Trash-$uid` (mis. FAT/exFAT)
  tidak ditemukan; Cleanlie tidak membuat trash di sana.
- Restore lintas filesystem mengembalikan `EXDEV` dan tidak menyalin data.
- Item yang dihapus permanen atau dibuat lewat perintah resmi tidak bisa
  di-undo.
- Preview hanya 20 anak pertama (urut ukuran) dan tidak membaca isi file.
- Ukuran volume trash bisa lambat pada disk lambat; UI tetap responsif karena
  pengukuran berjalan di background.
- `CLEANLIE_NO_GIO=1` hanya untuk test/escape hatch; tanpa gio, restore home
  memakai `os.Rename` dan tetap memverifikasi hasil.
