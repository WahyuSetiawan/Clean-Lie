# M5 — Analisis + Dashboard (implementasi)

Status: **selesai** 2026-11-25 · versi **0.5.0** · branch `feat/m5-analysis`.

M5 menambahkan paket baca-saja `internal/analysis`, binding Wails, layar **Analisis**
(5 tab), dan mengisi **Dashboard** dengan angka nyata dari scan yang sudah ada.
Tidak ada jalur hapus baru: tombol trash memakai `clean.Execute` dan guard yang sama
dengan layar **Bersihkan**.

## Isi

| File | Peran |
|---|---|
| `internal/analysis/analysis.go` | default (depth 2, file besar 100 MiB, stale 365 hari, top 100, duplikat 1 MiB, artefak aktif 7 hari), tipe `Tree`, `Overview`, `File`, `Artifact`, `Duplicates`, helper allocated/umur/top-N |
| `internal/analysis/tree.go` | `ScanTree` + rekursi `treeDir`; depth terakhir memakai `fsx.UsageSeenSkip` |
| `internal/analysis/overview.go` | `ScanOverview` — **satu** `WalkDir` untuk file besar, file lama, dan artefak; folder artefak tidak dimasuki lagi |
| `internal/analysis/duplicates.go` | `FindDuplicates` — group ukuran → hash 64 KiB → SHA-256 penuh; hardlink di-dedupe |
| `app_analysis.go` | binding `AnalysisTree`, `AnalysisOverview`, `AnalysisDuplicates`, `AnalysisRefresh`, `AnalysisCancel`, `AnalysisClean`, `DiskStatus`; cache 60 detik + cache ID target |
| `app.go` | field cache analisis + `analysisCancel` |
| `cli.go` | `--analysis-tree`, `--analysis-overview`, `--analysis-dups` (JSON), `--depth` |
| `frontend/dist/index.html` | Dashboard kartu/diskmap/top-reclaim nyata; tab Analisis dinamis |
| `frontend/test/ui-smoke.mjs` | mock binding analisis + assert dashboard, drill-down, trash, duplikat, artefak |
| `internal/fsx/walk.go`, `usage.go` | `Options.Inodes` (inode set bersama), `Options.SkipNames`, `UsageSeenSkip` |

## Alur data

```
Dashboard
  ScanJunk (M2) ──────────────► kartu cache/log + top reclaim + tool dev
  NixStatus (M3, cache 2 mnt) ► kartu Nix store/journal/generasi
  DiskStatus (statfs) ────────► kartu Disk /
  AnalysisTree(home, 1) ──────► diskmap + ukuran home

Analisis
  tab Disk usage   → AnalysisTree(path, 1) per klik folder (cache 60 s per path)
  tab File besar   → AnalysisOverview(home)  ┐
  tab File lama    → (hasil yang sama)        ├ satu WalkDir
  tab Artefak      → (hasil yang sama)        ┘
  tab Duplikat     → AnalysisDuplicates(home) (baru saat tab dibuka)

Aksi trash di semua tab → AnalysisClean({ids}) → clean.Execute(guard M2) → audit
```

UI **tidak pernah mengirim path**: `AnalysisClean` hanya menerima ID yang dibuat saat
hasil analisis disimpan di `analysisTargets` (mis. `large:/home/u/file.iso`,
`artifact:/home/u/proyek/node_modules`, `dup:/home/u/a.bin`). ID asing ditolak.

## Keputusan implementasi

- **Satu paket, tiga file** (bukan `large.go`/`stale.go`/`artifacts.go` terpisah).
  File besar, file lama, dan artefak butuh traversal yang sama; satu `WalkDir` menghemat
  I/O dan memberi umur/mtime yang konsisten. Artefak yang ditemukan di-skip agar
  `node_modules` di dalam `node_modules` tidak dihitung dua kali.
- **Tree memakai inode set global.** Setiap subtree pada depth terakhir memakai
  `fsx.UsageSeenSkip` dengan `sync.Map` yang sama; hardlink yang tersebar di dua
  subtree tidak dihitung dua kali. Ini juga yang membuat total tree mendekati `du`.
- **`.git` dilewati juga di depth terakhir.** `fsx.Options.SkipNames` diperlukan karena
  `fsx.Usage` sebelumnya tidak tahu nama yang harus dilewati; tanpa ini `.git` di bawah
  folder depth-2 ikut terhitung (selisih ~1,5 GB di mesin target).
- **Duplikat tidak menghitung hardlink.** Kandidat dengan `(dev, ino)` yang sama dilewati;
  hardlink bukan ruang yang bisa dibebaskan.
- **Duplikat dibatasi 100 grup tampil**, tetapi `wasted` dan `groupCount` tetap total.
  Pencarian penuh di mesin target 51 dtk / 1.715 grup; mengirim semua grup ke UI tidak
  sepadan.
- **Dashboard memakai ulang `ScanJunk` + `NixStatus`**, tidak membuat jalur hitung baru.
  Tombol **Bersihkan yang aman** tetap memakai alur M2 (`quick-clean` → `openClean('trash')`).
- **Analisis hanya menawarkan trash**, bukan hapus permanen; gate teks `GC` tetap khusus Nix.
- **`AnalysisCancel`** tersedia di binding (context per pemindaian); UI belum menampilkan
  tombol batal karena tiap tab sudah punya status memuat dan cache 60 detik.

## Angka nyata (mesin target, 2026-11-25)

| Ukuran | Hasil |
|---|---|
| Tree `~` depth 2 | **179.182.256.128 B** (166,88 GiB) · 3.385.788 file · 5,18 s |
| Pembanding `du -s --block-size=1 --exclude=.git ~` | 179.184.680.960 B → selisih **0,0013%** |
| Tree `~/.cache` depth 2 | **24.009.011.200 B** (22,36 GiB) · 902.202 file · **1,70 s** (< 10 s) |
| Overview `~` | 2.194.118 file · 247.927 folder · 15,3–18,7 s · 406 error permission |
| File besar | 100 item (top-N); teratas `system-images/android-36/.../system.img` 4,11 GiB |
| File lama | 100 item; banyak metadata cargo ber-mtime 1970 |
| Artefak | 100 item; teratas `~/.gradle` 13,09 GiB, `~/.npm-global/lib/node_modules` 4,23 GiB |
| Duplikat | **1.715 grup** · **17.932.690.922 B** (16,70 GiB) · 6.315 kandidat · 11.834 hash · 51,3 s · 423 error |
| Dashboard | kartu disk/statfs, Nix NAR 65,6 GiB, cache dev dari grup `dev.*` (tanpa Android), log `sys.log`, top-5 reclaim |

Top children tree home: `.local` 55,7 GiB, `.cache` 22,4 GiB, `.config` 16,2 GiB,
`Android` 14,0 GiB, `.gradle` 13,1 GiB.

## Verifikasi

```sh
nix develop -c gofmt -l .
nix develop -c go vet -tags desktop,production,webkit2_41 ./...
nix develop -c go test -tags desktop,production,webkit2_41 ./...
nix develop -c make build
nix build .#cleanlie

./bin/cleanlie --analysis-tree --depth 2 | jq '.bytes,.files,.durationMs'
./bin/cleanlie --analysis-tree --path ~/.cache --depth 2 | jq '.durationMs'
./bin/cleanlie --analysis-overview | jq '.large|length,.stale|length,.artifacts|length'
./bin/cleanlie --analysis-dups | jq '.groupCount,.wasted,.groups|length'

cd frontend/test && npm test
```

Hasil: **9 paket Go hijau** (main, analysis, android, clean, fsx, nix, privileged, rules,
tools), `gofmt`/`go vet` bersih, `make build` + `nix build` versi 0.5.0, UI smoke hijau
("scan, gate permanen, trash, hasil, rescan, Nix, GC, Android, dashboard, analisis").

## Deviasi dari task

1. `large.go`, `stale.go`, `artifacts.go` digabung menjadi `overview.go` (satu traversal).
2. Tree memakai kedalaman 1 per klik, bukan memuat seluruh depth 2 sekaligus; drill-down
   tetap 3+ level tanpa reload.
3. Duplikat menampilkan 100 grup teratas; total grup/wasted tetap dilaporkan.
4. File besar tidak memasuki folder artefak (supaya tidak dobel dengan tab Artefak).
5. Dashboard memakai `ScanJunk`/`NixStatus` yang sudah ada, bukan endpoint dashboard baru.
6. `AnalysisCancel` ada di binding tetapi belum ada tombol UI.
7. Aksi analisis hanya mode trash (tanpa permanen).

## Gotcha

- `fsx.Usage` per subtree punya inode map sendiri → hardlink lintas subtree dobel. Pakai
  `UsageSeen`/`UsageSeenSkip` dengan map bersama.
- `.git` di bawah folder depth terakhir tetap dipindai oleh `fsx.Usage`; perlu `SkipNames`.
- `du --exclude=.git` adalah pembanding yang tepat untuk tree; `du` tanpa exclude lebih
  dekat ke total filesystem.
- UI smoke `human()` membulatkan `2.100.000` menjadi `2 MiB` (bukan `2.00 MiB`).
- `AnalysisRefresh` mengosongkan cache **dan** target ID; setelah trash, UI harus memuat
  ulang tab aktif.
- `m-ok` modal M2 punya listener simulasi; modal analisis meng-clone tombol sebelum
  memasang handler nyata.
- Cache key tree memakai `root|depth`; dashboard dan drill-down memakai depth berbeda.
