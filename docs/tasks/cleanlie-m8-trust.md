# Cleanlie M8 — Trust: Task Breakdown

> Turunan dari [`../plan/cleanlie-m8-trust.md`](../plan/cleanlie-m8-trust.md).
> Satu task = satu commit yang bisa dites. Tidak ada task yang hanya “kelihatan benar”.
> Module: `github.com/Juragan-Koding/cleanlie` · UI: Wails v2 · Binary: `cleanlie`

## Status

| Milestone | Status | Bukti |
|---|---|---|
| **M8** Trust (trash, undo, riwayat, preview) | ✅ 2026-11-27 | `internal/trash` + `internal/preview` + Riwayat 8; **13 paket Go** hijau; UI smoke Riwayat/Trash/Preview/Undo hijau; `scripts/verify-trash.sh` hijau; `nix build` 0.8.0 |

## Aturan main

- Urutan **T8.1 → T8.2 → T8.3 → T8.4 → T8.5 → T8.6** wajib. T8.7 boleh paralel
  setelah T8.1; T8.8 menunggu T8.6; T8.9 menunggu T8.7 + T8.8.
- Semua test berjalan tanpa GUI: `go test ./internal/...` + flag CLI.
- File baru wajib `git add` sebelum `nix build` (flake hanya menyalin file tracked).
- `internal/trash` **tidak boleh** import `internal/clean` (boleh `rules` hanya
  untuk `rules.IsProtected`, dan `tools` untuk `gio`).
- UI tidak pernah mengirim path filesystem; hanya ID.
- Tidak ada operasi hapus/restore yang melewati validasi root.
- Commit kecil, prefix: `feat(m8):`, `fix(m8):`, `test(m8):`, `docs(m8):`.

## Ringkasan

| Task | Isi | Est. | Depends |
|---|---|---|---|
| T8.1 | `internal/trash`: tipe, discovery, parser, list metadata | 0,25 d | — |
| T8.2 | Ukuran streaming + cache + cancel | 0,15 d | T8.1 |
| T8.3 | Restore / delete / empty / resolve + safety | 0,35 d | T8.1, T8.2 |
| T8.4 | Integrasi `clean.MoveToTrash` + resolve `TrashPath` | 0,1 d | T8.3 |
| T8.5 | Audit batch/action, sesi, tail read, rotasi | 0,3 d | — |
| T8.6 | Undo + binding App + invalidasi cache | 0,4 d | T8.3, T8.4, T8.5 |
| T8.7 | `internal/preview` + binding | 0,2 d | T8.1 |
| T8.8 | UI Riwayat (Aktivitas + Trash) + i18n | 0,45 d | T8.6 |
| T8.9 | UI preview (Bersihkan + Trash) | 0,25 d | T8.7, T8.8 |
| T8.10 | CLI, script verifikasi, docs, rilis 0.8.0 | 0,25 d | T8.6, T8.9 |
| | **10 task** | **~2,7 d** | |

---

## T8.1 — `internal/trash`: tipe, discovery, parser, list metadata

- Buat paket `internal/trash`:
  - `Root{Path, MountPoint, Kind}`, `Entry`, `Report` (termasuk `Truncated`).
  - `DiscoverRoots() []Root`: home trash (`$XDG_DATA_HOME/Trash` atau
    `~/.local/share/Trash`) + per-volume (`$mnt/.Trash/$uid` yang `.Trash`-nya
    sticky, atau `$mnt/.Trash-$uid`) dari `/proc/self/mounts` (decode escape
    `\040` dll.); dedupe device+path; lewati pseudo-FS/read-only dan
    ntfs/exfat/fuseblk/network. Root volume harus dir nyata (bukan symlink); home
    trash boleh symlink (resolve sekali). Native: ext4/btrfs/xfs/f2fs/zfs/bcachefs.
  - `List(ctx, opts ListOptions) (Report, error)`: `opts.Roots` untuk test,
    `opts.MaxEntries`; parse `info/*.trashinfo`, decode `Path=`,
    parse `DeletionDate=`, pasangkan `files/`, skip nama kosong/`.`/`..`/mengandung
    `/`, tandai orphan/missing,
    `Restorable`, `Reason` (termasuk tujuan sudah ada/protected, best-effort),
    urut `DeletedAt` desc. Maks 5.000 info; lebih dari itu `Truncated=true`.
    `Path=` absolut untuk home, relatif untuk volume (join `Root.MountPoint`,
    tolak escape).
  - `ID = sha256(root+"\x00"+name)[:12]`.
- Path decode toleran: `url.PathUnescape`; kalau gagal pakai mentah. Tolak path
  relatif/kosong/`..`.
- **Acceptance:**
  - `go test ./internal/trash` hijau untuk fixture: home trash, `.Trash-$uid`,
    `.Trash/$uid`, `Path=` ter-encode dan mentah, `Path=` relatif volume,
    file tanpa info, info tanpa file, `DeletionDate` rusak.
  - `List` 1.000 entry metadata < 200 ms (`testing.B`).
  - `trash.go`/`list.go` tidak melakukan tulis (`os.Remove|os.Rename|os.WriteFile`);
    operasi tulis hanya ada di `restore.go`/`remove.go` (T8.3) dan tetap guarded.
- **Files:** `internal/trash/trash.go`, `internal/trash/list.go`,
  `internal/trash/trash_test.go`.

## T8.2 — Ukuran streaming + cancel

- `SizeEntries(ctx, entries []Entry, opts)`: worker pool 4, `fsx.UsageSeen` dengan
  satu `sync.Map` bersama (hardlink lintas entry dihitung sekali), symlink tidak
  diikuti, `Bytes=-1` sampai selesai, progress callback. `List` mengisi `Bytes=-1`
  untuk semua entry (bukan 0).
- Batas aman: 5.000 entry; `context` cancel; error per entry dicatat, bukan fatal.
- **Acceptance:**
  - fixture 100 entry: total byte == `du -s --block-size=1` (toleransi 1%).
  - cancel di tengah → < 500 ms, tidak ada goroutine bocor (`goleak` tidak wajib;
    cukup `WaitGroup` selesai).
  - entry yang hilang saat diukur → `Bytes=-1`, `Reason` diisi.
- **Files:** `internal/trash/size.go`, `internal/trash/size_test.go`.

## T8.3 — Restore / delete / empty / resolve + safety

- `Restore(ctx, entry) error`:
  - validasi `FilesPath`/`InfoPath` di dalam `Root` (lexical + `EvalSymlinks` pada
    root dan parent); `Name` basename valid (bukan `.`/`..`, tanpa `/`);
  - baca ulang `.trashinfo`, tujuan dari info;
  - tolak `ErrDestinationExists`, `ErrParentMissing`, `ErrProtected`
    (`rules.IsProtected` pada path lexical dan parent hasil `EvalSymlinks`);
  - `gio trash --restore trash:///<url.PathEscape(name)>` hanya untuk home trash;
    volume trash langsung `os.Rename`; fallback `os.Rename`;
  - `EXDEV` → error jelas, bukan copy; symlink dipindah sebagai symlink;
  - verifikasi tujuan ada + info hilang; hapus info bila fallback.
- `Delete(ctx, entries) ActionResult` dan `Empty(ctx, roots, confirm) ActionResult`:
  - hanya isi `files/` + `info/`, tidak pernah root;
  - `removeAllOneFS`: tolak/skip entry yang device-nya beda (mount point di
    dalam trash) agar tidak menghapus filesystem lain;
  - `Empty` menerima `KOSONGKAN` atau `EMPTY` (case-insensitive);
  - hitung byte sebelum hapus; item gagal tidak menghentikan sisanya.
- `Resolve(ctx, originalPath) (Entry, bool)`: targeted lookup
  `info/<basename>*.trashinfo` di root kandidat (home dulu, lalu mount point
  path tersebut; bukan `List` penuh; `ReadDir` + `HasPrefix`, bukan `Glob`);
  untuk volume, `Path=` relatif di-join `MountPoint`; tolak info symlink; pilih
  `DeletionDate` terbaru.
- **Acceptance:**
  - restore mengembalikan file + info hilang; `gio` opsional (skip bila tidak ada);
    volume trash memakai `os.Rename` (tanpa URI).
  - konflik tujuan → error, file lama utuh.
  - parent hilang → error, tidak ada `MkdirAll`.
  - `TrashPath`/`FilesPath` di luar root → ditolak.
  - symlink di `files/` tidak diikuti (target di luar tidak tersentuh); restore
    symlink menghasilkan symlink, bukan salinan target.
  - `info/`/`files/` symlink atau `.trashinfo` symlink → ditolak.
  - mount point di dalam entry → skip, filesystem lain tidak tersentuh.
  - `Empty` tanpa konfirmasi → error; dengan `KOSONGKAN`/`EMPTY` → isi kosong,
    direktori `files/`/`info/` tetap ada.
- **Files:** `internal/trash/restore.go`, `internal/trash/remove.go`,
  `internal/trash/resolve.go`, `internal/trash/*_test.go`.

## T8.4 — Integrasi `clean.MoveToTrash`

- `MoveToTrash()` setelah `gio` sukses memanggil `trash.Resolve(path)`; isi
  `TrashInfo.TrashPath`/`InfoPath` bila ketemu. Jangan gagalkan pembersihan bila
  resolve gagal. Resolve harus memakai targeted lookup (T8.3), bukan `List` penuh.
  `CLEANLIE_NO_GIO=1` melewati `gio` (test terisolasi).
- Tidak ada perubahan daftar protected: `clean.Validate` dan `trash` sama-sama
  memakai `rules.IsProtected`.
- **Acceptance:**
  - `go test ./internal/clean` tetap hijau.
  - dengan `gio` tersedia: `TrashPath` non-kosong untuk file di home (test di-skip
    bila `gio` tidak ada); tanpa `gio`: fallback manual tetap benar; volume native
    (`Path=` relatif) ter-resolve bila fixture disediakan.
  - clean 100 item fixture tidak memanggil `List()` penuh (dibuktikan dengan
    counter/injectable lister di test).
- **Files:** `internal/clean/trash.go`, `internal/clean/trash_test.go`.

## T8.5 — Audit batch/action, sesi, tail read, rotasi

- Tambah `Entry.Batch`, `Entry.Action`, `Result.Batch`, `Options.Batch`.
  `Execute()` membuat batch sekali (`<UTC>-<hex>`), dipakai semua entry.
- `Action`: `""` = clean (log lama), `restore`, `trash-delete`, `trash-empty`.
- `internal/clean/history.go`:
  - `SessionPage` + `ReadSessions(path, limit)` (terbaru dulu, grouping `Batch`);
  - fallback `(Time, Mode)` untuk log lama;
  - hitung `Removed`, `Failed`, `Freed`, `Restorable`, `RestorableBytes`;
  - maksimal 200 item per sesi di JSON (`ItemsTruncated=true`); `limit <= 0` →
    50, `> 200` → 200; `Truncated=true` bila `limit` tercapai.
- `ReadAuditTail(path, lines)` membaca dari belakang (tidak memuat seluruh file);
  lanjut ke `history.jsonl.1` bila sesi belum cukup; buang sesi tertua yang
  mungkin terpotong dan set `Truncated=true`.
- `RotateAudit(path)` dipanggil sekali di awal operasi tulis (bukan dry-run),
  bukan di tengah batch;
  rotasi ke `history.jsonl.1` bila > 8 MiB.
- `AppendAuditBatch(path, entries)` untuk undo/delete/empty (satu open/write).
- **Acceptance:**
  - satu `Execute` 5 item → 5 entry dengan `Batch` sama; `Result.Batch` sama.
  - log lama tanpa batch (fixture) tetap menghasilkan sesi yang benar.
  - log 10 MB: `ReadSessions(50)` < 100 ms; file lama tetap utuh setelah rotasi.
  - rotasi di tengah batch tidak memecah sesi: batch tetap utuh di satu file.
  - sesi dengan 300 item → 200 tampil, `ItemsTruncated=true`; undo tetap memakai
    data lengkap dari file.
  - tail terpotong → sesi tertua dibuang, `Truncated=true`.
  - `AppendAuditBatch` 1.000 entry < 50 ms.
  - `ReadAudit` lama tetap kompatibel.
- **Files:** `internal/clean/clean.go`, `internal/clean/audit.go`,
  `internal/clean/history.go`, `internal/clean/*_test.go`.

## T8.6 — Undo + binding App + invalidasi cache

- `app_history.go`:
  - `TrashList() (trash.Report, error)` mengembalikan metadata (`Bytes=-1`) lalu
    memulai `SizeEntries` di background; kirim event `trash:size` batch
    (throttle ~100 ms) dan `trash:done` saat selesai. `TrashRefresh()`
    membatalkan ukuran lama (`trashSizeCancel`).
  - `TrashRestore(ids []string, dryRun bool) (trash.ActionResult, error)`,
    `TrashDelete(ids []string, confirm bool, dryRun bool) (trash.ActionResult, error)`,
    `TrashEmpty(confirm string, dryRun bool) (trash.ActionResult, error)`.
  - `HistorySessions(limit int) (clean.SessionPage, error)`.
  - `UndoSession(batch string, dryRun bool) (trash.ActionResult, error)`.
- `App` menyimpan cache `trashEntries map[string]trash.Entry`, `trashAt`,
  `trashSizes`; `trashMu` untuk cache, `cleanMu` untuk semua mutasi.
- `UndoSession` hanya memproses sesi `Action in {"", "clean"}` dan item
  `Action in {"", "clean"}`, `Mode=trash`, `Status=removed`; resolve `TrashPath`
  atau `trash.Resolve(Path)` (fallback bila `TrashPath` hilang/tidak valid); skip
  item permanen/command dengan alasan; tulis audit `Action=restore` dengan batch
  `undo-<timestamp>-<hex>`; `dryRun` tidak menulis audit.
- `TrashDelete`/`TrashEmpty` menulis audit `Action=trash-delete`/`trash-empty`
  dengan batch `trash-<timestamp>-<hex>` memakai `AppendAuditBatch`;
  `Mode=permanent`, `BytesBefore=bytes`, `Freed=bytes` (byte yang benar-benar
  hilang; orphan/missing = 0).
- `TrashRestore` menulis audit `Action=restore` dengan batch
  `restore-<timestamp>-<hex>` (satu batch untuk satu panggilan);
  `Mode=trash`, `BytesBefore=bytes`, `BytesAfter=0`, `Freed=0`.
- Setiap operasi tulis memanggil `RotateAudit` di awal; `dryRun` melewati
  konfirmasi (`confirm`/`KOSONGKAN`) dan tidak menulis audit.
- Resolusi ID dari UI: bila ID tidak ada di cache, panggil `trash.List()` ulang
  sekali sebelum menolak (trash bisa berubah dari luar aplikasi). Dedupe ID
  sebelum diproses.
- Setelah mutasi: invalidasi `scanItems`, cache analisis (`analysisTrees`,
  `analysisOverviews`, `analysisDups`), cache direnv, cache trash; emit event
  `trash:changed`. Kirim `trash:progress` selama delete/empty/undo.
- **Acceptance:**
  - `go test . -run 'TestTrash|TestUndo|TestHistory'` hijau.
  - undo 3 item (2 punya trash path, 1 permanen) → `Restored=2`, `Skipped=1`,
    audit berisi 2 `Action=restore` dengan batch yang sama.
  - undo `dryRun=true` → tidak ada file/audit berubah, hasil `skipped`/`planned` jelas.
  - undo batch tidak dikenal → error jelas, bukan `Restored=0` sukses.
  - audit yang diubah manual (`TrashPath=/etc/passwd`) → skip, tidak ada file
    yang berpindah.
  - restore satu item → item hilang dari `TrashList`, `scanItems` invalid.
  - `TrashDelete` tanpa `confirm` → error; `TrashEmpty` tanpa `KOSONGKAN`/`EMPTY`
    → error; `TrashRestore`/`TrashDelete` tanpa ID → error.
- **Files:** `app_history.go`, `app_history_test.go`, `app.go`,
  `app_clean.go` (invalidasi), `app_analysis.go`/`app_direnv.go` (helper invalidasi
  bila perlu).

## T8.7 — `internal/preview` + binding

- `Preview(ctx, path, limit)`:
  - file: `Lstat` saja;
  - dir: baca maks 1.000 anak pertama, anak langsung, maks 20 tampil, ukuran via
    `fsx.Usage`, urut terbesar, `Truncated`, `TimedOut`, batas 1,5 s, symlink
    tidak diikuti.
- Binding `PreviewCleanup(id)` (path dari `a.scanItems`) dan `PreviewTrash(id)`
  (path dari cache trash). Tambah `Blocked` dari `clean.BlockedReason`.
- **Acceptance:**
  - fixture: 25 anak → 20 tampil, `Truncated=true`, urut menurun.
  - symlink ke `/etc` → `Kind=symlink`, ukuran target tidak dihitung.
  - file 10 MB → `Kind=file`, `Entries` kosong.
  - path hilang → `Kind=missing`, bukan error 500.
  - ID asing → error jelas; path dari JS tidak pernah diterima.
- **Files:** `internal/preview/preview.go`, `internal/preview/preview_test.go`,
  `app_preview.go` (atau gabung `app_history.go`).

## T8.8 — UI Riwayat (Aktivitas + Trash) + i18n

- Nav baru `Riwayat` (`#/history`) setelah Analisis; `META` + `page.history.*` +
  CSS tab/table/badge di `index.html`.
- `frontend/dist/ui/history.js`:
  - tab **Aktivitas**: daftar sesi (waktu, mode, item, freed, restorable),
    filter teks/mode, detail, tombol Undo + modal ringkasan hasil parsial.
  - tab **Trash**: cari, pilih, restore, hapus, kosongkan (gate `KOSONGKAN`/`EMPTY`
    sesuai bahasa), badge `Restorable=false`, indikator ukuran streaming
    (`trash:size`, `trash:done`).
  - setelah mutasi: refresh + toast + `C.ensureScan(true)` bila perlu; dengar
    `trash:changed` untuk refresh tab aktif.
- Urutan `<script>`: `history.js` + `preview.js` dimuat sebelum `screens.js`
  (agar preload `C.ensureHistory` tersedia); `preview.js` boleh setelah
  `cleanup.js` karena dipanggil saat klik.
- Modal hasil `Clean` (`cleanup.js`): tombol **Undo sesi ini** bila `res.batch`
  punya item restorable; panggil `UndoSession(res.batch)`.
- Pengaturan: tombol Riwayat → `navigate('history')`; “Buka folder audit” tetap.
- `demo.js`: data sesi/trash palsu; `screens.js`: preload `HistorySessions` +
  `TrashList` saat masuk.
- i18n ID/EN untuk semua string baru; tidak ada hardcoded di `history.js`.
- **Acceptance:** `cd frontend/test && npm test` hijau dengan alur:
  - buka Riwayat → sesi tampil; filter; undo → binding dipanggil + hasil parsial;
  - tab Trash → list tampil; item non-restorable tetap bisa dipilih untuk Hapus
    tetapi dilewati oleh Pulihkan; restore/hapus/kosongkan memanggil binding yang
    benar; gate `KOSONGKAN` mengunci tombol;
  - hasil `Clean` menampilkan tombol Undo dan memanggil `UndoSession(batch)`;
  - event `trash:size` batch + `trash:done` memperbarui ukuran tanpa reload;
    mock `EventsOn` menyimpan handler agar test bisa memancarkan event;
  - `trash:progress` menampilkan progres delete/empty.
- **Files:** `frontend/dist/index.html`, `frontend/dist/ui/history.js`,
  `frontend/dist/ui/demo.js`, `frontend/dist/ui/screens.js`,
  `frontend/dist/ui/settings.js`, `frontend/dist/ui/cleanup.js`,
  `frontend/test/ui-smoke.mjs`.

## T8.9 — UI preview (Bersihkan + Trash)

- `frontend/dist/ui/preview.js`: modal preview (total, mtime, 20 entri, badge
  blocked, `TimedOut`, tombol buka folder).
- Tombol **Lihat isi** di baris Bersihkan dan baris Trash; panggil
  `PreviewCleanup(id)` / `PreviewTrash(id)`.
- Loading state + Esc menutup + fokus kembali ke tombol pemicu.
- **Acceptance:** UI smoke:
  - klik preview di baris `_cacache` → `PreviewCleanup` dipanggil dengan ID,
    entri tampil, `Blocked` tampil bila ada;
  - klik preview di baris trash → `PreviewTrash` dipanggil; bila cache trash kosong,
    UI menampilkan pesan “buka tab Trash dulu” (bukan error mentah);
  - Esc menutup modal dan fokus kembali ke tombol pemicu.
- **Files:** `frontend/dist/ui/preview.js`, `frontend/dist/ui/cleanup.js`,
  `frontend/dist/ui/history.js`, `frontend/dist/index.html`,
  `frontend/test/ui-smoke.mjs`.

## T8.10 — CLI, script verifikasi, docs, rilis 0.8.0

- Flag baru di `cli.go`: `--trash-list`, `--no-size`, `--trash-restore`,
  `--trash-delete`, `--trash-empty`, `--history`, `--undo`, `--limit`, `--yes`.
- Aturan:
  - `--trash-list` menghitung ukuran secara sinkron; `--no-size` melewati ukuran;
  - destruktif tanpa `--yes` → exit 2, tidak ada perubahan; `--dry-run` boleh
    tanpa `--yes`;
  - `--dry-run` untuk undo/restore/delete/empty → rencana saja;
  - semua output JSON valid; error ke stderr.
- `scripts/verify-trash.sh <binary>`: `HOME`/`XDG_DATA_HOME`/`XDG_STATE_HOME` di
  `mktemp -d`, `CLEANLIE_NO_GIO=1` (escape hatch test agar `gio`/gvfs tidak
  menyentuh trash asli). Seed fixture: pindahkan file ke `Trash/files`, tulis
  `.trashinfo` + satu entry audit `history.jsonl` (batch uji) — bukan lewat GUI.
  Seed 3 entry; uji `--trash-list` → `--trash-restore` (1) → `--undo` (1) →
  `--trash-delete` (1) → `--trash-empty` (sisa); cek byte dan keberadaan file.
  Tidak menyentuh trash asli. CLI meneruskan `KOSONGKAN` saat `--yes`.
- Docs: `docs/implementation/m8-trust.md`, README (bagian trash/undo/riwayat,
  8 layar, flag CLI, istilah `guard` → `procs`/proteksi), CHANGELOG 0.8.0
  (sekalian perbaiki bagian “Rencana berikutnya” yang masih M6), tautan M8 di
  `docs/plan/cleanlie-v1.md` + `docs/tasks/cleanlie-v1.md`, versi 0.8.0 di
  `Makefile`, `flake.nix`, `package.nix`, dan `main.go` (`0.8.0-dev`).
- **Acceptance:**
  - `./bin/cleanlie --trash-list --json | jq .` valid;
  - `--trash-empty` tanpa `--yes` exit 2 dan trash tidak berubah;
  - `--history --limit 5 --json | jq .sessions` valid;
  - `scripts/verify-trash.sh ./bin/cleanlie` hijau;
  - `nix build .#cleanlie` 0.8.0;
  - `go test -tags desktop,production,webkit2_41 ./...` hijau;
  - `cd frontend/test && npm test` hijau;
  - `gofmt -l` kosong untuk file Go baru/berubah; `go vet ./...` bersih.
- **Files:** `cli.go`, `cli_test.go`, `scripts/verify-trash.sh`, `README.md`,
  `CHANGELOG.md`, `Makefile`, `flake.nix`, `package.nix`, `main.go`,
  `docs/implementation/m8-trust.md`, `docs/plan/cleanlie-v1.md`,
  `docs/tasks/cleanlie-v1.md`.

---

## Definisi selesai (M8)

1. Item yang masuk trash bisa dilihat, dipulihkan, dihapus, dan trash bisa
   dikosongkan dari dalam aplikasi — tanpa file manager.
2. Satu sesi `Clean` bisa di-undo; laporan jujur untuk item yang tidak bisa
   di-undo (permanen/command/jejak hilang).
3. Riwayat sesi bisa dibaca dan difilter di UI; audit lama tetap terbaca.
4. Preview isi item tampil sebelum hapus, termasuk alasan diblokir proses.
5. Semua operasi tersedia headless (`--trash-*`, `--history`, `--undo`) dengan JSON.
6. Audit tetap append-only, berotasi, dan **tidak dipercaya** untuk path.
7. Tidak ada jalur restore/hapus yang melewati validasi root/protected.
8. `go test ./...`, UI smoke, `gofmt`, `go vet`, dan `nix build` hijau.

## Catatan implementasi (diisi saat M8 dikerjakan)

- Tanggal mulai: 2026-11-26; selesai: 2026-11-27.
- Temuan saat implementasi:
  - `gio trash --restore <path>` tidak ada; `trash:///<name>` hanya untuk home
    trash. Volume `.Trash-$uid` memakai `os.Rename` (relative `Path=`).
  - Pengecekan protected harus menargetkan **file tujuan**, bukan parent:
    `$HOME` ada di daftar protected, sehingga parent yang sah ikut ditolak.
  - `glob` tidak aman untuk nama trash yang mengandung `*`/`?`; resolusi
    memakai `ReadDir` + `HasPrefix`.
  - File tanpa `.trashinfo` (orphan) tetap harus terlihat dan bisa dihapus.
  - Test UI memerlukan mock `HistorySessions`/`Trash*`/`Preview*`; event
    `trash:size`/`trash:done` diverifikasi lewat handler runtime.
- Deviasi dari plan:
  - `--preview` tidak ditambahkan ke CLI (preview tetap UI-only) agar path dari
    CLI tidak menjadi jalur baru; flag sesuai daftar T8.10.
  - `scripts/verify-trash.sh` memakai 4 fixture (bukan 3) agar restore, undo,
    delete, dan empty semuanya teruji.
- Hasil verifikasi nyata:
  - `go test ./internal/... .` → 13 paket hijau; `-tags desktop,production,webkit2_41 ./...` hijau.
  - `cd frontend/test && npm test` → UI smoke OK (termasuk Riwayat/Trash/Preview/Undo).
  - `scripts/verify-trash.sh /tmp/cleanlie-m8` → list 4 entry, restore 1, undo 1,
    delete 1, empty 1, tanpa menyentuh trash asli.
  - `--trash-empty` tanpa `--yes` → exit 2; `--history --limit 5 --json` valid.
  - `nix build .#cleanlie` → 0.8.0.
