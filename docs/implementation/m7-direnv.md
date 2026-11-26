# M7 — direnv (implementasi)

Status: **selesai** 2026-11-26 · versi **0.7.0** · branch `feat/m7-direnv`.

M7 menambahkan pemindai folder `.direnv` per proyek. Pengguna menambahkan satu
atau lebih folder proyek, lalu Cleanlie mencari `.direnv` di dalamnya (tanpa
memasuki `.git`, `node_modules`, dan artefak build lain), mengukur ukuran
allocated, dan menawarkan pemindahan ke trash. Tidak ada jalur hapus baru:
aksi memakai `clean.Execute` + guard yang sama dengan layar Bersihkan/Analisis.

## Kenapa

`.direnv` dibuat oleh direnv; dengan nix-direnv isinya profil devshell yang
menunjuk ke `/nix/store`. Folder ini bisa besar dan sering tertinggal setelah
proyek tidak dipakai. Menghapusnya aman: direnv/nix-direnv membuat ulang saat
proyek dibuka lagi (mungkin perlu `direnv allow`). Cache global
`~/.cache/direnv` dan `~/.cache/nix-direnv` juga diukur.

## Isi

| File | Peran |
|---|---|
| `internal/direnv/direnv.go` | `Scan` read-only: walk per root, deteksi `.direnv`, ukur `fsx.Usage`, cache global, sort, total |
| `app_direnv.go` | binding `DirenvPickRoot`, `DirenvScan`, `DirenvCancel`, `DirenvRefresh`, `DirenvAddRoot`, `DirenvRemoveRoot`, `DirenvClean`; cache 60 detik + target ID |
| `internal/config/config.go` | field `direnvRoots` (dinormalisasi seperti `artifactRoots`) |
| `app.go` | state scan direnv (`direnvMu`, report, key, target) |
| `cli.go` | `--direnv-scan` (JSON; `--path DIR` atau root dari config) |
| `frontend/dist/index.html` | nav + layar **direnv** |
| `frontend/dist/ui/direnv.js` | pilih folder (dialog native) / tambah / hapus, scan/cancel, pilih aman, trash, hasil |
| `frontend/test/ui-smoke.mjs` | mock binding direnv + alur tambah→scan→pilih→trash |
| `internal/direnv/direnv_test.go`, `app_direnv_test.go` | fixture `.direnv`, skip artefak, symlink tidak diikuti, exclusion, round-trip config |

## Alur data

```
Pengguna memilih folder lewat dialog native (DirenvPickRoot) atau mengetik path
  → DirenvAddRoot (Settings.DirenvRoots, ~/.config/cleanlie/config.json)
  → DirenvScan
      walk tiap root:
        .direnv ditemukan → fsx.Usage (allocated, symlink tidak diikuti) → Entry
        .git/node_modules/target/... → SkipDir
      + ~/.cache/direnv, ~/.cache/nix-direnv (kind=cache)
      → Report{entries, bytes, files, missing}
  → DirenvClean({ids}) → clean.Execute(guard M2) → trash + audit
```

UI hanya mengirim ID (`direnv:<path>`), bukan path. ID asing ditolak.
Target disimpan saat scan: root = folder proyek, depth = 1, whole = true.
Untuk cache global, root = path cache itu sendiri.

## Keputusan implementasi

- **Walk sendiri, bukan `analysis.ScanOverview`.** Kebutuhan utamanya menemukan
  folder bernama `.direnv` dan berhenti di situ; memakai walker analisis akan
  mengukur isi `.direnv` sebagai bagian dari total proyek dan tidak memberi
  daftar per-proyek.
- **`.direnv` tidak dimasuki.** Setelah ditemukan, `SkipDir` dipakai supaya
  profil di dalamnya tidak dianggap proyek lain dan tidak dihitung dua kali.
- **Symlink tidak diikuti.** `fsx.Usage` memakai `Lstat` + walk tanpa mengikuti
  symlink. Ini penting untuk nix-direnv: profil menunjuk ke `/nix/store`, dan
  mengikuti symlink akan melaporkan ukuran store sebagai ukuran proyek.
- **Cache global dipisah (`kind=cache`).** `~/.cache/nix-direnv` sebagian besar
  symlink; UI memberi catatan bahwa ruang nyata dibebaskan oleh GC Nix, bukan
  dengan menghapus cache-nya.
- **Root dibatasi 32** (`direnv.MaxRoots`) agar walk tidak liar; root yang sama
  dideduplikasi dan dinormalisasi (`~`, absolut, clean).
- **Cache 60 detik** dan `DirenvRefresh` dipanggil dari `SaveSettings`, jadi
  perubahan folder/exclusion langsung invalidasi hasil.
- **Auto-scan saat layar dibuka** hanya bila sudah ada root dan belum ada hasil.
  Scan bisa dibatalkan (`DirenvCancel`).

## Verifikasi

- `go test ./internal/... .` — 11 paket hijau (termasuk `internal/direnv`).
- `node frontend/test/ui-smoke.mjs` — tambah folder → scan → pilih aman →
  trash → hasil → rescan → hapus folder hijau.
- `cleanlie --direnv-scan --path <dir>` mengembalikan JSON `Report`.
- `nix build .#cleanlie` — versi 0.7.0.

## Batasan

- `.direnv` di dalam folder yang masuk daftar skip (`node_modules`, `target`,
  `.git`, dst.) tidak ditemukan. Ini disengaja agar walk cepat.
- Root yang tidak ada dicatat di `missing`, bukan error; UI menampilkannya.
- Folder aktif (< 7 hari) tidak ikut "Pilih aman", tetapi tetap bisa dipilih
  manual.
- Penghapusan `.direnv` tidak memanggil `direnv` (tidak ada perintah resmi untuk
  membersihkan satu proyek); guard trash + audit tetap dipakai.
