# M3 — Nix: generasi, dead paths, dan aksi root

Status: **selesai 2026-11-25**. Gate M3 — angka dry-run GC harus sama dengan CLI —
lolos: **624 path** dari `--nix-dead` dan **624 path** dari
`nix-collect-garbage --dry-run` pada saat yang sama.

## Yang dibangun

| File | Isi |
|---|---|
| `internal/privileged/privileged.go` | `Execute`, `DetectAgent`, `SudoCommand`, `Preview`; jalur root → pkexec → sudo siap-copy |
| `internal/nix/nix.go` | `ReadStatus`, `DeadPaths`, profile/generation discovery, closure, journal, store |
| `internal/nix/parse.go` | Parser `nix path-info -S`, `nix path-info --json`, dead paths, ukuran journal, hitungan dry-run |
| `internal/nix/actions.go` | `DeleteGenerations`, `CollectGarbage`, `VacuumJournal`, preview perintah |
| `app_nix.go` | Binding Wails: status/dead/preview/aksi, cache 2 menit, invalidasi setelah aksi sukses |
| `frontend/dist/index.html` | Layar Nix nyata: kartu, tabel generasi, gate `GC`, vacuum journal, fallback sudo |
| `cli.go` | `--nix-status` dan `--nix-dead` (JSON) untuk verifikasi headless |

`internal/nix` hanya membaca; semua eksekusi berada di `actions.go` dan lewat
`internal/privileged`. `internal/rules` tetap tidak mengimpor `internal/nix`
maupun `internal/clean`, jadi pemindaian katalog tidak pernah menyentuh store.

## Sumber data (dan kenapa bukan yang di plan)

| Data | Implementasi | Alasan |
|---|---|---|
| Ukuran store | `nix path-info --json --all`, jumlah `narSize` | `du -sh /nix/store` = **75 s**; path-info = **5,4 s** |
| Dead paths | `nix-store --gc --print-dead` lalu `nix path-info --json` per 300 path | `nix-collect-garbage --dry-run` hanya memberi **jumlah**, bukan byte; print-dead + path-info = **31,6 s** |
| Generasi system | symlink `system-<N>-link` + **mtime** | `nix-env -p /nix/var/nix/profiles/system --list-generations` gagal sebagai user (`Permission denied` pada `system.lock`) |
| Generasi user/home-manager | `nix-env -p <path> --list-generations` | tanggal asli; fallback mtime bila perintah gagal |
| Closure per generasi | satu `nix path-info -S` per batch 300 path unik | 129 generasi → 53 closure unik → **±1,2 s**; bukan 129 panggilan `-Sh` |
| Journal | `journalctl --disk-usage` | output `4.0G`; parser menerima `4.0G`, `500M`, koma desimal |
| Free/total disk | `fsx.Statfs("/")` | tidak menambah dependency |

Store adalah **NAR size**, bukan allocated block. Angka ini sengaja dipakai untuk
konsistensi dengan `nix path-info`, dan tidak diklaim sama dengan `du` (yang
menghitung block + metadata). Untuk UI, selisihnya kecil dan arahnya konservatif.

Profil yang dibaca:

| Nama UI | Path |
|---|---|
| `system` | `/nix/var/nix/profiles/system` |
| `home-manager` | `~/.local/state/nix/profiles/home-manager` |
| `user (nix-env)` | `~/.local/state/nix/profiles/profile` |

`~/.nix-profile` bukan profile generasi; ia menunjuk ke salah satu link di atas.
Karena itu UI memakai path `profile` yang sebenarnya.

## Angka nyata mesin target (2026-11-25)

```
storeBytes      70.425.698.432   (≈65,6 GiB NAR, 47.208 path)
generasi        129              (system 5, home-manager 50, user 74)
suggested       100              (umur ≥ 30 hari, bukan current)
journalBytes    4.187.593.113    (≈3,9 GiB)
dead paths      624 path / 9.720.894.784 byte (≈9,0 GiB NAR), 31,6 s
agent           false            → GCCommand = sudo /run/current-system/sw/bin/nix-collect-garbage -d
status duration 7,5 s
```

`--nix-dead` dan `nix-collect-garbage --dry-run` dijalankan berurutan pada store
yang sama dan sama-sama menghasilkan **624**. Itulah gate M3.

## Alur aksi root

```
UI (klik GC / hapus generasi / vacuum)
  │  hanya mengirim nama profile + nomor generasi, bukan path
  ▼
internal/nix.DeleteGenerations / CollectGarbage / VacuumJournal
  │  validasi: confirm == "GC", generasi ada, bukan current, target journal regex
  ▼
internal/privileged.Execute
  ├─ euid == 0 ...................... jalankan langsung
  ├─ polkit agent terdeteksi ........ pkexec <binary absolut> <args>
  │    └─ exit 126/127 ............. NeedsSudo + SudoCommand
  └─ tidak ada agent / pkexec hilang  NeedsSudo + SudoCommand (tidak dieksekusi)
  ▼
ActionResult { command, preview, sudoCommand, needsSudo, used, exitCode, stdout, stderr, error }
  ▼
UI: modal hasil + tombol Salin perintah; tidak pernah menampilkan "berhasil" bila NeedsSudo
```

Deteksi agent dilakukan dengan membaca `/proc/<pid>/cmdline` (bukan `pgrep`),
mencari `hyprpolkitagent` atau proses yang mengandung `polkit` **dan** `agent`.
Mesin target tidak punya agent polkit; karena itu semua aksi root berakhir di
dialog `sudo` siap-copy. Ini perilaku yang diinginkan: tidak ada kegagalan senyap.

Path absolut wajib karena `pkexec` membersihkan environment; `nix-env` dan
`nix-collect-garbage` harus di-resolve dengan `exec.LookPath` sebelum dipanggil.
`SudoCommand` memakai shell-quoting dari `tools.Preview`, sehingga path dengan
spasi tetap aman untuk disalin.

## Guard di UI

| Guard | Implementasi |
|---|---|
| Konfirmasi ketik `GC` | tombol OK disabled sampai `input.value.trim() === "GC"`; dicek lagi di Go |
| Generasi current | tidak dirender tombol Hapus; `DeleteGenerations` menolak nomor current |
| Nomor generasi asing | harus ada di profile; kalau tidak → error, tidak ada eksekusi |
| Profile asing | hanya `system`, `home-manager`, `user`; path tidak diterima dari UI |
| Journal injection | regex `^\d{1,4}[MG]$`; UI hanya menawarkan 500 MB / 1 GB / 2 GB |
| Double action | tombol langsung disabled saat proses; `NixDeleteGenerations`/GC invalidate cache setelah sukses |
| Dead paths masih dihitung | tombol GC disabled sampai `NixDeadPaths` selesai; klik terlalu awal hanya memberi toast |
| Hasil butuh sudo | modal menampilkan `sudoCommand`, tombol Salin; `deleted` tidak diisi |

## UI

Layar `#view-nix` memakai wireframe yang sudah ada, tetapi datanya nyata:

- **Kartu store** — NAR size, jumlah path, sisa/total disk.
- **Kartu dead paths** — hasil `--print-dead`, jumlah path, durasi, sumber.
- **Kartu generasi** — total, ringkasan per profile, tanggal terlama.
- **Tabel** — profile, nomor, tanggal, closure, status (`current` / `saran hapus` /
  `lama N hari` / `link hilang`), aksi Hapus per baris.
- **Hapus generasi > 30 hari** — mengelompokkan saran per profile, menampilkan
  preview perintah, lalu satu gate `GC` untuk semuanya.
- **Jalankan GC** — ringkasan store/dead + peringatan tidak bisa di-undo + gate `GC`.
- **Vacuum journal** — ukuran sekarang, target, preview, fallback sudo.
- **Muat ulang** — membuang cache UI dan server; status ±7 s, dead paths ±32 s.
  Selama dead paths dihitung, tombol GC tetap disabled.

`NixStatus` dan `NixDeadPaths` di-cache 2 menit di server. Tanpa cache, setiap
kembali ke layar Nix akan memicu 7 s + 32 s. Setelah aksi sukses cache dibuang,
sehingga angka tidak berbohong.

## Verifikasi

```sh
nix develop -c go vet -tags desktop,production,webkit2_41 ./...
nix develop -c go test -tags desktop,production,webkit2_41 ./...
nix build .#cleanlie

# backend nyata
./bin/cleanlie --nix-status | jq '{storeBytes,storePaths,journalBytes,agent,gcCommand}'
./bin/cleanlie --nix-dead   | jq .
nix-collect-garbage --dry-run | tail -1      # harus sama dengan count di atas

# UI (jsdom, opsional)
cd frontend/test && npm test
```

Hasil:

- `go vet` bersih; **7 paket** hijau (main, clean, fsx, nix, privileged, rules, tools).
- `--nix-status` → 70.425.698.432 byte, 47.208 path, 129 generasi, agent `false`,
  `gcCommand` sudo absolut, `errors` kosong.
- `--nix-dead` → **624** path / 9.720.894.784 byte.
- `nix-collect-garbage --dry-run` → **624**.
- UI smoke jsdom: scan → gate permanen → trash → hasil → rescan → layar Nix →
  tabel 2 generasi → gate `GC` terkunci/terbuka → hasil GC. Hijau.
- Test nyata `nix-env -p <profile-sementara> --delete-generations 1` pada profile
  di `/tmp/opencode/nixprof` berhasil (bukan profile user/system), membuktikan
  sintaks perintah yang dipakai.

Test yang di-gate:

- `NIX_TEST_REAL=1 go test ./internal/nix -run TestRealStatus` — status nyata.
- Test privileged memakai runner palsu; tidak pernah menjalankan pkexec/sudo.

## Deviasi dari plan/task

1. **T3.1** — system generation tidak lewat `nix-env` (lock ditolak non-root);
   tanggal dari mtime link. Jumlah dan nomor generasi tetap dari symlink, jadi
   acceptance "jumlah == CLI" tetap bisa dibandingkan sebagai root.
2. **T3.2** — closure diambil dengan satu `nix path-info -S` per batch, bukan
   `-Sh` per generasi. Ukuran sama; jauh lebih cepat.
3. **T3.3** — `nix-collect-garbage --dry-run` hanya memberi jumlah path, tanpa
   byte. Ukuran memakai `nix-store --gc --print-dead` + `nix path-info --json`.
   Gate diganti menjadi **jumlah path print-dead == jumlah dry-run**.
4. **T3.4** — `internal/privileged` satu file (+ test), bukan `detect.go`/`exec.go`
   terpisah; deteksi lewat `/proc`, bukan `pgrep` (tidak menambah proses).
5. **T3.5** — tombol per baris dan tombol "hapus > 30 hari" sama-sama memakai
   gate `GC`; wireframe hanya menunjukkan satu gate untuk aksi massal.

## Sengaja belum

- Auto-GC / GC terjadwal — plan melarang; aplikasi hanya menampilkan.
- `nix store gc` per-user roots — GC penuh lewat root sudah menangani.
- Diff generasi (`nvd`), pinning, rollback, atau `nixos-rebuild` — di luar v1.
- Hapus store path individual — hanya lewat GC; menghindari kerusakan closure.
- Progress bar real-time untuk `--print-dead` — perintah tidak memberi progress;
  UI menampilkan durasi + status "menghitung dead paths" dan mengunci GC.
- Integrasi polkit agent buatan sendiri — memakai agent sistem bila ada.

## Gotcha

- `nix path-info --json --all` mengembalikan **object map** (`{"/nix/store/...": {...}}`),
  bukan array. `--json-format 1` saja bukan pengganti. Parser menerima keduanya.
- `nix-env` pada profile system membuka lock dan gagal sebagai user; jangan
  retry berulang, langsung fallback ke mtime.
- `nix-store --gc --print-dead` tidak memberi progress dan butuh ±30 s; jangan
  panggil dari handler UI tanpa goroutine/cache (Wails binding berjalan async,
  tetapi cache tetap perlu agar pindah layar tidak mengulang).
- `nix path-info` untuk path yang sudah hilang mengembalikan error; jumlah path
  tetap dihitung dari print-dead, byte hanya dari path yang berhasil diukur.
- `os.Geteuid() == 0` harus mengalahkan `Options.IsRoot`; kalau tidak, aplikasi
  yang dijalankan sebagai root tetap meminta pkexec.
- UI tidak boleh mengirim path. Semua aksi memakai nama profile + nomor; path
  profile di-resolve di Go dari daftar tertutup.
