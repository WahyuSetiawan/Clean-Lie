# Cleanlie M8 — Trust (Trash Manager, Undo, Riwayat, Preview)

> Addendum `cleanlie-v1.md`. Status: **selesai** (2026-11-27, v0.8.0).
> Prasyarat: M7 selesai (v0.7.0). Target rilis: **0.8.0**.
> Satu milestone = satu rangkaian commit yang bisa dites tanpa GUI.

## 1. Latar belakang

M2 sudah memindahkan item ke trash freedesktop dan menulis audit
`~/.local/state/cleanlie/history.jsonl`. Tapi setelah item masuk trash, aplikasi
**menyerahkan tanggung jawab restore ke file manager**:

- `OpenTrash()` (`app_clean.go`) hanya menjalankan `xdg-open` ke `Trash/files`.
- `History()` sudah ada tetapi tidak dipakai UI; tombol "Riwayat" di Pengaturan hanya
  membuka folder audit.
- `clean.Entry` sudah menyimpan `TrashPath`, **tetapi `MoveToTrash()` lewat `gio`
  mengembalikan `TrashPath` kosong** — jadi undo berbasis audit belum bisa jalan
  untuk jalur normal.
- Tidak ada cara melihat isi item sebelum menghapus; `BlockedReason()` baru muncul
  saat eksekusi.
- `ReadAudit()` membaca seluruh file; log akan tumbuh tanpa batas.

M8 menutup celah ini: **trash bisa dikelola dan di-undo dari dalam aplikasi**,
riwayat bisa dibaca, dan user bisa melihat isi item sebelum menekan tombol hapus.

## 2. Tujuan & non-goal

### Tujuan

1. **Trash Manager** — daftar isi trash (home + volume), ukuran, path asal, tanggal,
   restore, hapus permanen per item, kosongkan trash.
2. **Undo per sesi** — satu klik mengembalikan semua item dari satu sesi `Clean`
   yang masih bisa di-restore, dengan laporan parsial yang jujur.
3. **Riwayat in-app** — sesi audit (waktu, mode, jumlah item, freed, status) + filter
   + detail per item.
4. **Preview** — isi teratas sebuah item (nama, ukuran, mtime), total, dan alasan
   diblokir proses, sebelum item dihapus.
5. **CLI parity** — semua operasi di atas bisa dipakai headless dengan JSON.
6. **Keamanan** — tidak ada jalur restore/hapus baru yang melewati validasi root,
   dan audit diperlakukan sebagai *petunjuk*, bukan sumber kebenaran path.

### Non-goal M8

- Auto-clean terjadwal, tray, notifikasi (kandidat M9).
- Export laporan, hardlink/reflink duplikat, rule Docker/Flatpak (M9/M10).
- Restore untuk hapus permanen dan aksi berbasis perintah (`npm cache clean`):
  secara teknis tidak mungkin; UI hanya menandai **tidak bisa di-undo**.
- Restore ke path alternatif / “restore as…”. Konflik = ditolak dengan alasan.
- Preview untuk item Analisis (ID analisis berbeda; ditunda).
- Filesystem eksternal/network (NTFS/exFAT/FUSE/NFS) dan trash milik user lain;
  M8 hanya mengelola volume Linux native yang sudah ter-mount (ext4/btrfs/xfs/f2fs).

Catatan: non-goal v1 “restore dari trash” **tidak berlaku lagi** untuk M8, tetapi
hanya untuk volume Linux native; pemindaian filesystem eksternal tetap non-goal.

## 3. Arsitektur

```
internal/
├── trash/                 # BARU — sumber kebenaran isi trash
│   ├── trash.go           # Root, Entry, Report, DiscoverRoots
│   ├── list.go            # parse .trashinfo + files (metadata)
│   ├── size.go            # ukuran streaming + cancel
│   ├── restore.go         # gio restore → fallback rename
│   ├── remove.go          # hapus item / kosongkan (guarded)
│   ├── resolve.go         # cari entry dari path asal (post-gio)
│   └── *_test.go
├── preview/               # BARU — isi item, read-only
│   ├── preview.go
│   └── preview_test.go
├── clean/
│   ├── clean.go           # + Batch di Result, Action/Batch di Entry
│   ├── history.go         # BARU — Session, ReadSessions, tail read, rotasi
│   ├── procs.go           # tidak berubah; BlockedReason tetap di sini
│   └── trash.go           # MoveToTrash → resolve TrashPath setelah gio
app_history.go             # BARU — binding trash/history/undo/preview
app_clean.go               # Clean memakai batch + invalidasi cache
cli.go                     # flag trash/history/undo
frontend/dist/ui/history.js # BARU — layar Riwayat (2 tab)
frontend/dist/ui/preview.js # BARU — modal preview (dipakai cleanup + history)
```

Aturan dependency:

- `internal/trash` → `internal/fsx`, `internal/rules` (hanya `rules.IsProtected`)
  dan `internal/tools` (untuk `gio`). **Tidak** import `internal/clean` (mencegah
  siklus karena `clean` justru import `trash` untuk resolve setelah `gio trash`).
- `internal/preview` → `internal/fsx` saja. Alasan blokir proses diisi oleh `App`
  (memanggil `clean.BlockedReason`) agar preview tidak bergantung pada `clean`.
- `internal/clean` → `internal/trash` (hanya untuk resolve setelah `gio trash`).
- `app_*` → semua paket. UI tidak pernah mengirim path; hanya ID.

## 4. Model data

### 4.1 `internal/trash`

```go
type RootKind string // "home" | "volume"

type Root struct {
    Path       string // .../Trash atau /mnt/x/.Trash-1000
    MountPoint string // titik mount volume; "" untuk home trash
    Kind       RootKind
}

type Entry struct {
    ID           string    `json:"id"`           // hash(root+name), stabil antar-run
    Name         string    `json:"name"`         // nama di files/
    OriginalPath string    `json:"originalPath"` // hasil decode .trashinfo
    DeletedAt    time.Time `json:"deletedAt"`
    Bytes        int64     `json:"bytes"`        // -1 = belum diukur
    FilesPath    string    `json:"-"`
    InfoPath     string    `json:"-"`
    Root         string    `json:"root"`
    RootKind     RootKind  `json:"rootKind"`
    Restorable   bool      `json:"restorable"`
    Reason       string    `json:"reason,omitempty"` // alasan tidak bisa restore
}

type Report struct {
    Roots      []Root   `json:"roots"`
    Entries    []Entry  `json:"entries"`
    Bytes      int64    `json:"bytes"` // jumlah entry yang sudah diukur (Bytes >= 0)
    Errors     []string `json:"errors,omitempty"`
    Truncated  bool     `json:"truncated"` // > 5.000 entry
    ScannedAt  string   `json:"scannedAt"`
    DurationMs int64    `json:"durationMs"`
}

// Hasil restore/delete/empty (dipakai App + CLI).
type ItemResult struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    OriginalPath string `json:"originalPath"`
    Status       string `json:"status"` // restored | deleted | skipped | failed | planned
    Bytes        int64  `json:"bytes"`
    Reason       string `json:"reason,omitempty"`
}

type ActionResult struct {
    Items      []ItemResult `json:"items"`
    Restored   int          `json:"restored"`
    Deleted    int          `json:"deleted"`
    Skipped    int          `json:"skipped"`
    Failed     int          `json:"failed"`
    Bytes      int64        `json:"bytes"`
    AuditError string       `json:"auditError,omitempty"`
}
```

ID memakai `sha256(root + "\x00" + name)[:12]`; bukan path mentah, supaya UI/CLI
tidak bisa menyusupkan path. Nama file tetap ditampilkan.

### 4.2 Audit (`internal/clean`)

Tambahan field **backward-compatible** (`omitempty`):

```go
type Entry struct {
    // ... field lama ...
    Batch  string `json:"batch,omitempty"`  // satu nilai per Execute
    Action string `json:"action,omitempty"` // "" = clean (log lama), restore, trash-delete, trash-empty
}

type Result struct {
    // ... field lama ...
    Batch string `json:"batch"`
}
```

`Execute()` membuat batch sekali:

```
<UTC 20060102T150405.000Z>-<4 byte hex>
```

Bila `crypto/rand` gagal, fallback ke timestamp + `os.Getpid()`. `Options.Batch`
bisa diisi test. Semua entry dari satu `Execute` memakai batch yang sama; entry
lama tanpa batch dikelompokkan berdasarkan `Time` yang memang sama untuk satu
eksekusi.

Entry untuk `trash-delete`/`trash-empty` memakai `Mode=permanent` +
`Action=trash-delete`/`trash-empty`; entry `restore` memakai `Mode=trash` +
`Action=restore`.

### 4.3 Sesi

```go
type Session struct {
    ID          string  `json:"id"`
    StartedAt   string  `json:"startedAt"`
    Mode        Mode    `json:"mode"`
    Action      string  `json:"action"`
    Items       []Entry `json:"items"`
    Removed     int     `json:"removed"`
    Failed      int     `json:"failed"`
    Freed       int64   `json:"freed"`
    Restorable  int     `json:"restorable"`
    RestorableBytes int64 `json:"restorableBytes"`
    ItemsTruncated bool `json:"itemsTruncated"` // item > 200 dipotong
}
```

`ReadSessions(path, limit) (SessionPage, error)` mengembalikan **terbaru dulu**.
`limit` membatasi jumlah sesi (bukan baris); `limit <= 0` → 50, `> 200` → 200;
`SessionPage.Truncated` true bila batas baca tail **atau** `limit` tercapai. Default
UI: 50 sesi.

```go
type SessionPage struct {
    Sessions  []Session `json:"sessions"`
    Truncated bool      `json:"truncated"`
}
```

## 5. Trash: discovery, list, restore, delete, empty

### 5.1 Discovery

1. **Home trash**: `$XDG_DATA_HOME/Trash` atau `~/.local/share/Trash`.
2. **Per-volume trash** (freedesktop): baca `/proc/self/mounts`, lalu cek
   `$mnt/.Trash/$uid` (harus direktori `.Trash` dengan sticky bit) dan
   `$mnt/.Trash-$uid`. Hanya fstype Linux native (ext4/btrfs/xfs/f2fs); lewati
   ntfs/exfat/fuseblk/network. `Root.MountPoint = $mnt`; root valid bila `files/`
   ada dan bisa dibaca. Root volume harus direktori nyata (bukan symlink); home
   trash boleh symlink dan di-`EvalSymlinks` sekali. Fstype native: ext4/btrfs/
   xfs/f2fs/zfs/bcachefs.
3. Dedupe berdasarkan `(device, path)`, lewati pseudo-FS dan mount read-only.
4. Bila root tidak bisa dibaca: catat di `Report.Errors`, jangan gagal total.

### 5.2 List

- Baca `info/*.trashinfo` (maks 5.000 file), parse `Path=` dan `DeletionDate=`.
  Bila lebih, `Report.Truncated=true` + catatan di `Errors`.
- `Path=` di-decode `url.PathUnescape`; kalau gagal, pakai nilai mentah.
  - **Home trash:** `Path=` absolut; tolak path relatif/kosong/`.`/`..`.
  - **Volume trash:** `Path=` relatif terhadap `Root.MountPoint` (spesifikasi
    freedesktop); tolak kosong/`.`; join lalu `filepath.Clean`; tolak bila hasil
    keluar dari `MountPoint`. Sebagian implementasi menulis absolut; terima bila
    masih di dalam `MountPoint`.
- `Entry.OriginalPath` selalu absolut setelah normalisasi.
- Pasangkan dengan file di `files/` bernama sama (basename info tanpa `.trashinfo`).
  Skip nama kosong/`.`/`..` dan nama yang mengandung `/`.
- `info/` dan `files/` harus direktori nyata di dalam root (bukan symlink);
  `.trashinfo` yang symlink ditolak. Entry di `files/` boleh symlink (itu isi trash).
- File tanpa info → tampil sebagai **orphan**, `Restorable=false`,
  `Reason="metadata .trashinfo hilang"`.
- Info tanpa file → `Restorable=false`, `Reason="file trash hilang"`.
- Tujuan restore sudah ada atau protected → `Restorable=false` + alasan
  (best-effort `Lstat`; dicek ulang saat restore).
- `DeletionDate` di-parse `2006-01-02T15:04:05` di `time.Local` (fallback
  `RFC3339`); gagal → pakai mtime info file, tandai `Reason` bila perlu.
- Urutkan entry terbaru dulu (`DeletedAt` desc), lalu nama.

**Ukuran tidak dihitung saat list.** `List()` hanya metadata (target < 200 ms untuk
1.000 entry). Ukuran dihitung terpisah oleh `SizeEntries(ctx, entries, progress)`
dengan worker pool 4, `fsx.UsageSeen` + satu set inode bersama (hardlink lintas
entry tidak dihitung dua kali), `context` cancel, dan cache 60 detik di `App`. UI
menampilkan `—` lalu terisi lewat event `trash:size` yang di-throttle ~100 ms
(payload batch `{updates:[{id,bytes}], done, total}`), bukan satu event per item.

### 5.3 Restore

Urutan:

1. Re-validasi `FilesPath`/`InfoPath` berada di dalam salah satu `Root` yang
   ditemukan (lexical + `EvalSymlinks` pada parent). Tolak bila di luar. `Name`
   harus basename valid (bukan `.`/`..`, tanpa `/`).
2. Baca ulang `.trashinfo`; **path tujuan diambil dari info, bukan dari audit**.
3. Tolak bila tujuan ada (`ErrDestinationExists`) atau parent tidak ada
   (`ErrParentMissing`). Tidak ada auto-rename, tidak ada `MkdirAll` diam-diam.
4. Tolak bila tujuan protected: `rules.IsProtected` pada path lexical **dan**
   parent hasil `EvalSymlinks` (mencegah parent symlink ke `~/.ssh`).
5. Coba `gio trash --restore trash:///<url.PathEscape(name)>` untuk **home trash**
   bila `gio` tersedia. Untuk **volume trash**, langsung `os.Rename` (trash root
   satu filesystem dengan tujuan) karena URI `trash:///` untuk topdir trash tidak
   portabel. Fallback semua kasus: `os.Rename`; `EXDEV` → error, bukan copy.
   `os.Rename` pada symlink memindahkan symlink, bukan targetnya.
   `CLEANLIE_NO_GIO=1` memaksa jalur manual (dipakai test terisolasi).
6. Setelah sukses: verifikasi file tujuan ada dan info file hilang (gio) atau hapus
   manual (fallback); bila info masih ada padahal file sudah pindah, hapus info
   sisa. Bila verifikasi gagal → laporkan error, jangan klaim sukses.
7. Tulis audit `Action=restore`, `Status=removed`, `BytesBefore=bytes`, `BytesAfter=0`,
   `Freed=0`. Restore tidak boleh menambah “freed” pada total sesi clean. Satu
   operasi restore/undo memakai satu `Batch` (`restore-`/`undo-`).

### 5.4 Hapus item & kosongkan trash

- `Delete(ctx, entries)` hanya menerima `Entry` hasil `List()` (ID dicocokkan ulang).
  Hapus `files/name` lalu `info/name.trashinfo`. Jangan pernah menghapus `files/`
  atau `info/` itu sendiri.
- `Empty(ctx, roots)` menghapus **isi** semua root yang dikenal, bukan root-nya.
  Wajib konfirmasi literal `KOSONGKAN` atau `EMPTY` (case-insensitive) di UI;
  CLI memakai `--yes`.
- `os.RemoveAll` pada symlink hanya menghapus link (tidak mengikuti); tetap
  divalidasi berada di bawah root.
- Hitung byte sebelum hapus; laporkan per item + total. Item gagal tidak
  menghentikan sisanya. Setiap operasi delete/empty memakai satu `Batch`
  berprefix `trash-` agar auditnya tampil sebagai satu sesi.

### 5.5 Resolve setelah `gio trash`

`MoveToTrash()` saat ini mengembalikan `TrashPath` kosong bila `gio` berhasil.
M8 menambahkan `trash.Resolve(ctx, originalPath)` yang **tidak** memanggil `List()`
penuh (mahal bila dipanggil per item):

- tentukan kandidat root dari path asal (home atau mount point path tersebut);
- untuk volume, `Path=` di `.trashinfo` relatif; join `MountPoint` sebelum
  membandingkan;
- cari `info/<basename>*.trashinfo` di root itu, parse, dan cocokkan `Path=`;
  gunakan `ReadDir` + `HasPrefix`, **bukan** `filepath.Glob` (basename bisa
  mengandung `*`/`?`);
- pilih `DeletionDate` paling baru; isi `TrashInfo.TrashPath` + `InfoPath`;
- bila tidak ketemu (mis. per-volume tidak terbaca), biarkan kosong dan audit
  mencatat `Reason` tambahan **tanpa** menggagalkan pembersihan.

Resolve penuh (scan semua root) hanya dipakai saat undo, bukan saat clean.

## 6. Riwayat & undo

### 6.1 Sesi

- `ReadSessions` membaca tail file (lihat §7) dan mengelompokkan per `Batch`.
- Log lama tanpa `Batch`: kelompokkan per `(Time, Mode)`.
- `Session.ID` = `Batch`; untuk log lama = hash(`Time|Mode`).
- Satu sesi memuat maksimal 200 item di JSON (`ItemsTruncated=true` bila lebih);
  detail lengkap tetap ada di `history.jsonl` untuk CLI/undo.
- `Restorable` dihitung dari item `Action in {"", "clean"}`, `Mode=trash`,
  `Status=removed`, dan `(TrashPath != "" || CommandLine == "")`. Item dengan
  `CommandLine` sukses tidak punya jejak trash, jadi tidak ditawarkan.
- `TrashPath` kosong tetap boleh; undo akan mencoba `trash.Resolve(Path)`.
- Item `permanent`, `failed`, `skipped`, `kept`, dan aksi perintah tidak dihitung.

### 6.2 Undo

`UndoSession(batch)`:

1. Ambil sesi dari audit (bukan dari UI); hanya sesi `Action in {"", "clean"}`.
   Batch tidak ditemukan → error jelas, bukan sukses kosong.
2. Untuk tiap item yang memenuhi syarat: resolve `TrashPath` → validasi root →
   restore (§5.3). Bila `TrashPath` hilang/tidak valid, coba `trash.Resolve(Path)`
   sebelum menyerah.
3. Item tanpa trash path dicoba `trash.Resolve(Path)`; kalau tetap tidak ada,
   `skipped` dengan alasan “jejak trash tidak ditemukan”.
4. Item permanen/command: `skipped` “tidak bisa di-undo”.
5. Kembalikan `trash.ActionResult{Items, Restored, Skipped, Bytes, AuditError}`.
6. Tulis audit `Action=restore` per item dengan satu `Batch` berprefix `undo-`
   untuk seluruh operasi undo (supaya tampil sebagai satu sesi restore).
7. Setelah selesai: invalidasi cache scan/analisis/direnv agar UI tidak menampilkan
   item yang sudah kembali.

Undo **tidak** meminta konfirmasi teks; ini operasi restoratif. Tetap tampilkan
modal ringkasan (apa yang akan dipulihkan dan apa yang akan dilewati).

### 6.3 Keamanan audit

Audit bisa ditulis user. Karena itu:

- Jangan percaya `Entry.TrashPath`/`Entry.Path` untuk operasi filesystem.
- Selalu cocokkan dengan entry hasil `trash.List()` dan baca ulang `.trashinfo`.
- Tolak bila path di luar root trash, tujuan bukan path absolut, atau tujuan
  berada di protected list.

## 7. Audit: skala & rotasi

- `ReadSessions` membaca **tail** file: cari newline dari belakang sampai jumlah
  baris yang dibutuhkan, lalu parse. Tidak membaca seluruh file. Bila sesi yang
  diminta belum cukup, lanjutkan ke `history.jsonl.1`.
- Bila batas baca tail tercapai, buang sesi tertua yang mungkin terpotong dan set
  `Truncated=true`; jangan pernah menawarkan undo untuk sesi yang tidak lengkap.
- `RotateAudit(path)` dipanggil **sekali di awal operasi tulis** (`Execute`, undo,
  trash-delete, trash-empty; bukan dry-run), bukan di tengah batch. Bila file > 8 MiB:
  `history.jsonl` → `history.jsonl.1` (timpa arsip lama), lalu buat baru.
  Ini mencegah satu batch terpecah dua file.
- Batas baca tail: 4 MiB atau 10.000 baris, mana yang lebih dulu; bila terpotong,
  `ReadSessions` mengembalikan sesi yang terbaca + `Truncated=true`.
- `AppendAuditBatch(path, entries)` menulis banyak entry dalam satu `open`/`write`
  (dipakai undo/delete/empty) agar operasi 5.000 item tidak membuka file 5.000 kali.
- `ReadAudit` lama tetap ada untuk kompatibilitas, tetapi UI memakai `ReadSessions`.
- Acceptance: log 10 MB, `ReadSessions(50)` < 100 ms dan tidak memuat seluruh file
  ke memori (diverifikasi dengan `testing.AllocsPerRun`/benchmark sederhana).

## 8. Preview

```go
type Entry struct {
    Name    string `json:"name"`
    Path    string `json:"path"`
    Bytes   int64  `json:"bytes"`
    Dir     bool   `json:"dir"`
    Symlink bool   `json:"symlink"`
    ModTime string `json:"modTime"`
}

type Result struct {
    Path       string  `json:"path"`
    Kind       string  `json:"kind"` // file | dir | symlink | missing
    Bytes      int64   `json:"bytes"`
    Files      int64   `json:"files"`
    ModTime    string  `json:"modTime"`
    Entries    []Entry `json:"entries"`   // maks 20, urut terbesar
    Truncated  bool    `json:"truncated"`
    TimedOut   bool    `json:"timedOut"`
    Blocked    string  `json:"blocked,omitempty"`
    DurationMs int64   `json:"durationMs"`
}
```

- Direktori: baca maks 1.000 anak pertama (`Readdirnames`), anak langsung via
  `Lstat`, ukuran per anak dengan `fsx.Usage` (symlink tidak diikuti), maks 20
  tampil, batas waktu 1,5 detik, `context` cancel. `Truncated`/`TimedOut` jujur.
- File: `stat` saja (tanpa membaca isi).
- `Blocked` diisi `App` dari `clean.BlockedReason(path)`.
- Binding hanya menerima ID:
  - `PreviewCleanup(id)` → path dari `a.scanItems` (hasil scan terakhir).
  - `PreviewTrash(id)` → path dari cache `trash.List` terakhir.
- Path dari JS tidak pernah diterima.

## 9. UI

### 9.1 Navigasi

Tambah satu layar **Riwayat** (`#/history`) setelah Analisis. Dua tab:

```
┌ Riwayat ──────────────────────────────────────────────────────────────┐
│ 12 sesi · 148 item · 21,4 GiB dibebaskan · 96 item bisa di-undo      │
│ [ Aktivitas ] [ Trash ]                         [ Segarkan ]          │
├───────────────────────────────────────────────────────────────────────┤
│ 2026-11-27 09:12  trash   12 item   3,2 GiB   [Undo]  [Detail]       │
│ 2026-11-26 18:04  permanent 3 item  1,1 GiB   (tidak bisa di-undo)   │
└───────────────────────────────────────────────────────────────────────┘
```

Tab **Trash**:

```
[ cari… ]  [ pilih semua ]  [ Pulihkan ]  [ Hapus ]  [ Kosongkan… ]
┌ nama ─────────────┬ path asal ────────────────┬ dihapus ┬ ukuran ┬ ──┐
│ _cacache          │ ~/.npm/_cacache           │ 27/11   │ 7,0 G  │ … │
│ test_devices.avd  │ ~/.android/avd/…          │ 26/11   │ 5,1 G  │ … │
└───────────────────┴───────────────────────────┴─────────┴────────┴───┘
```

- Baris dengan `Restorable=false` menampilkan badge alasan; checkbox tetap aktif
  untuk **Hapus**, tetapi tombol **Pulihkan** melewati baris tersebut dan
  menampilkan jumlah yang dilewati.
- Aksi massal hanya untuk entry yang valid.
- “Kosongkan…” memakai modal konfirmasi ketik `KOSONGKAN` (ID) atau `EMPTY` (EN),
  dan memperingatkan bahwa ini menghapus isi trash semua aplikasi.
- Modal hasil `Clean` menampilkan tombol **Undo sesi ini** bila `res.batch` punya
  item restorable; memanggil `UndoSession(batch)`.
- Setelah restore/hapus: refresh, invalidasi cache, toast hasil parsial.

### 9.2 Preview

- Tombol **Lihat isi** di baris Bersihkan dan baris Trash.
- Modal: total, mtime, 20 entri teratas (nama, ukuran), badge “dipakai proses: …”
  bila `Blocked` terisi, tombol “Buka folder”.
- Loading state; tidak memblokir UI.

### 9.3 Pengaturan

- Tombol Riwayat → `#/history` (bukan membuka folder).
- Tetap sediakan “Buka folder audit” sebagai aksi sekunder.

### 9.4 Demo & i18n

- `demo.js` menambah data sesi/trash palsu supaya wireframe tetap jalan tanpa Wails.
- Semua string layar baru lewat `__t()`; tambah kamus ID/EN.
- `frontend_embed_test.go` otomatis memastikan `history.js`/`preview.js` ter-embed.

### 9.5 Kontrak binding

```go
func (a *App) TrashList() (trash.Report, error)
func (a *App) TrashRefresh() bool
func (a *App) TrashRestore(ids []string, dryRun bool) (trash.ActionResult, error)
func (a *App) TrashDelete(ids []string, confirm, dryRun bool) (trash.ActionResult, error)
func (a *App) TrashEmpty(confirm string, dryRun bool) (trash.ActionResult, error)
func (a *App) HistorySessions(limit int) (clean.SessionPage, error)
func (a *App) UndoSession(batch string, dryRun bool) (trash.ActionResult, error)
func (a *App) PreviewCleanup(id string) (preview.Result, error)
func (a *App) PreviewTrash(id string) (preview.Result, error)
```

Event: `trash:size` (payload batch + `done`/`total`, throttle ~100 ms), `trash:done`,
`trash:progress` (`done`, `total`, `name`), `trash:changed`.
Semua binding berat memakai `context` dari `App`/`cleanMu`; UI hanya mengirim ID.

## 10. CLI

| Flag | Sifat | Output |
|---|---|---|
| `--trash-list [--no-size] [--json]` | baca | `trash.Report` (ukuran dihitung; `--no-size` opsional) |
| `--history [--limit N] [--json]` | baca | `clean.SessionPage` (default 50) |
| `--undo BATCH [--dry-run] [--json]` | tulis (restoratif) | `trash.ActionResult` |
| `--trash-restore ID[,ID] [--dry-run] [--json]` | tulis | `trash.ActionResult` |
| `--trash-delete ID[,ID] [--yes] [--dry-run] [--json]` | tulis | `trash.ActionResult` (freed) |
| `--trash-empty [--yes] [--dry-run] [--json]` | tulis | `trash.ActionResult` (freed) |

- ID = `Entry.ID` dari `--trash-list`.
- Flag destruktif tanpa `--yes` → exit 2 + pesan, tanpa perubahan.
- `--dry-run` tidak butuh `--yes` (tidak mengubah apa pun).
- `--trash-empty` menerima konfirmasi `KOSONGKAN`/`EMPTY`; CLI memakai `--yes`.
- `--undo` batch tidak dikenal → exit 1 + pesan, bukan sukses kosong.
- Semua perintah JSON valid (`jq .`) dan tidak membuka GUI.
- `--dry-run` untuk undo/restore menampilkan rencana tanpa menyentuh file.

## 11. Keamanan & konsistensi

| Aturan | Implementasi |
|---|---|
| UI tidak mengirim path | ID di-resolve di Go dari cache/scan terakhir |
| Restore tidak menimpa | cek `Lstat` tujuan; konflik = skip + alasan |
| Tidak ada mkdir diam-diam | parent tujuan harus ada |
| Symlink tidak diikuti | `Lstat`; `EvalSymlinks` hanya pada parent root; restore symlink = symlink |
| Mount point di dalam trash | `Delete`/`Empty` memakai `removeAllOneFS` (device sama) dan skip entry berisi mount |
| Audit tidak dipercaya | path tujuan dari `.trashinfo`; `Path=` volume relatif di-join `MountPoint` dan divalidasi |
| Tujuan restore bukan protected | `rules.IsProtected` pada path lexical + parent hasil `EvalSymlinks` |
| Root trash tidak dihapus | hanya isi `files/` + `info/`; subdir symlink ditolak |
| Rotasi tidak memecah batch | `RotateAudit` di awal operasi |
| Mutasi serial | `cleanMu` dipakai Clean, restore, delete, empty, undo |
| Race dengan scan | invalidasi `scanItems` + cache analisis/direnv setelah mutasi |
| Item permanen/command | tidak pernah ditawarkan untuk undo |

## 12. Acceptance (gate M8)

1. `go test -tags desktop,production,webkit2_41 ./...` hijau; `gofmt`/`go vet` bersih.
2. `cd frontend/test && npm test` hijau (termasuk alur baru).
3. Fixture nyata (Go test `TestM8EndToEnd` + `scripts/verify-trash.sh`):
   - Go test: file 2 MiB di `HOME`/`XDG_*` sementara → `clean.Execute` mode trash
     (`CLEANLIE_NO_GIO=1`) → `trash.List` memuat entry dengan `originalPath`
     benar → `trash.Restore` mengembalikan file + info hilang → `ReadSessions`
     memuat sesi dengan `Batch`;
   - script CLI: seed trash + audit entry di `HOME` sementara → `--trash-list`,
     `--trash-restore`, `--undo`, `--trash-delete`, `--trash-empty` (tanpa GUI);
   - `--trash-empty --yes` membebaskan byte dan menyisakan direktori `files/`/`info/`.
4. Keamanan:
   - audit yang diubah manual (`TrashPath` ke `/etc/passwd`) tidak menghapus/
     memindahkan apa pun;
   - entry di luar root trash ditolak;
   - symlink di trash tidak diikuti;
   - restore ke path yang sudah ada → skip dengan alasan, file lama utuh.
5. Performa:
   - `--trash-list` 1.000 entry metadata < 200 ms (tanpa ukuran);
   - preview direktori besar < 1,5 s;
   - `--history --limit 50` pada log 10 MB < 100 ms;
   - clean 100 item dengan `gio` + resolve < 2 s (resolve tidak memanggil `List` penuh).
6. Tidak ada jalur hapus/restore baru yang melewati validasi root/protected.
7. UI: 0 console error, modal bisa ditutup Esc, resize 1280×800 & 1920×1080.
8. Rotasi di tengah batch tidak memecah sesi: `--undo <batch>` tetap menemukan
   semua item setelah `history.jsonl` berotasi.
9. `--undo <batch> --dry-run` tidak mengubah file dan tidak menulis audit.
10. `--trash-delete`/`--trash-empty` tanpa `--yes` exit 2 dan trash tidak berubah;
    dengan `--dry-run` tetap boleh tanpa `--yes`.

## 13. Risiko & mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| `gio` tidak memberi `TrashPath` | undo tidak jalan | `trash.Resolve()` setelah gio + fallback resolve saat undo |
| Trash per-volume tidak terbaca | item tidak terlihat | discovery `/proc/self/mounts`; root gagal masuk `Errors`, bukan crash |
| `Path=` volume relatif | restore ke path salah | join `MountPoint`, tolak escape; terima absolut hanya di dalam mount |
| Root volume symlink | baca/hapus di luar mount | `Lstat` wajib direktori nyata untuk root volume |
| Mount point di dalam trash | hapus filesystem lain | `removeAllOneFS`, skip entry yang device-nya beda |
| Basename mengandung `*`/`?` | resolve salah entry | `ReadDir` + `HasPrefix`, bukan `filepath.Glob` |
| Audit ditulis user | path palsu | validasi root + baca ulang `.trashinfo`; audit hanya petunjuk |
| Restore menimpa file baru | kehilangan data | tolak konflik, tampilkan alasan, jangan auto-rename |
| Trash besar (ribuan item) | list lambat | metadata dulu, ukuran streaming + cache 60 s |
| Log audit tumbuh | memori/IO | tail read + rotasi 8 MiB |
| Race clean vs restore | status kacau | satu `cleanMu` untuk semua mutasi |
| Symlink/junction di trash | hapus di luar root | `Lstat`, validasi prefix, tidak mengikuti symlink |
| Tanpa `gio`, clean lintas device | item volume gagal masuk trash | perilaku lama dipertahankan; M8 hanya mengelola trash volume yang sudah ada |
| Log lama tanpa batch | undo salah grup | fallback grouping `(Time, Mode)`; item tanpa trash path = skip; dua operasi dalam detik yang sama bisa tergabung (hanya log pra-0.8) |
| Sesi terpotong batas tail | sesi tidak tampil | buang sesi tertua + `Truncated=true`; `UndoSession` membaca batch penuh dari file, bukan dari `SessionPage` |
| Audit bulk 5.000 item | lambat | `AppendAuditBatch` sekali tulis |
| Rotasi di tengah batch | undo kehilangan item | rotasi sekali di awal operasi |
| Resolve per item mahal | clean melambat | targeted lookup `info/<basename>*.trashinfo`, bukan `List` penuh |
| `gio` tidak terpasang | restore gagal | fallback `os.Rename`; `EXDEV` = error jelas |
| Item command/permanent | user kecewa | badge “tidak bisa di-undo” sejak awal |

## 14. Keputusan yang dikunci

| # | Keputusan | Alasan |
|---|---|---|
| 1 | Satu layar **Riwayat** dengan tab Aktivitas + Trash | Menghindari nav ke-9; riwayat dan trash saling terkait |
| 2 | Restore menolak konflik (tanpa auto-rename) | Tidak mengejutkan dan tidak menimpa data |
| 3 | Kosongkan trash wajib ketik `KOSONGKAN`/`EMPTY` | Aksi massal permanen harus disengaja |
| 4 | Audit JSONL tetap append-only + rotasi 8 MiB | Sederhana, mudah di-debug, tidak butuh DB |
| 5 | ID trash = hash(root+name) | UI/CLI tidak pernah mengirim path |
| 6 | CLI destruktif wajib `--yes` | Mencegah salah ketik di terminal |
| 7 | Preview maks 20 entri / 1,5 s | Cukup untuk keputusan, tidak membekukan UI |
| 8 | Ukuran trash dihitung terpisah | List tetap cepat; UI bisa streaming |
| 9 | Rotasi audit di awal operasi | Satu batch tidak terpecah dua file |
| 10 | Protected memakai `rules.IsProtected` | Satu sumber kebenaran, tanpa siklus paket |

## 15. Estimasi

| Bagian | Est. |
|---|---|
| `internal/trash` + integrasi `clean` + test (T8.1–T8.4) | 0,85 d |
| audit batch/sesi/rotasi + test (T8.5) | 0,3 d |
| undo + binding + test (T8.6) | 0,4 d |
| preview + binding (T8.7) | 0,2 d |
| UI Riwayat + Trash + preview + i18n (T8.8–T8.9) | 0,7 d |
| CLI + script verifikasi + docs + rilis 0.8.0 (T8.10) | 0,25 d |
| **Total** | **~2,7 d** |

Task rinci ada di [`../tasks/cleanlie-m8-trust.md`](../tasks/cleanlie-m8-trust.md).
