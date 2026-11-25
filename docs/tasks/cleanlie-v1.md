# Cleanlie v1 — Task Breakdown

> Turunan dari `docs/plan/cleanlie-v1.md`. Satu task = satu commit yang bisa dites.
> Module: `github.com/Juragan-Koding/cleanlie` · UI: Wails v2 · Binary: `cleanlie`
>
> **Wireframe UI (acuan visual):** [`../wireframe/index.html`](../wireframe/index.html) —
> `xdg-open docs/wireframe/index.html`. Pemetaan layar → task ada di `../wireframe/README.md`.
> Kalau wireframe dan mockup ASCII di plan berbeda, **wireframe yang jadi acuan**.

## Status

| Milestone | Status | Bukti |
|---|---|---|
| **M0** scaffold | ✅ 2026-11-25 | `nix develop -c make build` → `bin/cleanlie` 8.8 MB; `nix build .#cleanlie` sukses; window jalan 6 s tanpa crash; binding `Diagnostics()` tampil di layar Pengaturan |
| M1 engine | ✅ 2026-11-25 | `--dry-run --json` 150 item/9 grup; `scripts/verify-sizes.sh` 15 item terbesar delta ≤ 1%; 4 paket test hijau |
| M2 hapus + Bersihkan | ✅ 2026-11-25 | `internal/clean` + layar Bersihkan nyata; UI smoke (jsdom) scan→gate→trash→hasil hijau; restore trash nyata lolos; 5 paket Go hijau |
| M3 Nix | ✅ 2026-11-25 | `--nix-status` 129 generasi/47.208 path/70,4 GB NAR; `--nix-dead` **624 path** == `nix-collect-garbage --dry-run` **624**; 7 paket Go hijau; UI smoke gate `GC` hijau |
| M4 Android | ✅ 2026-11-25 | `--android-status` 1 AVD/5,01 GiB + SDK 13,81 GiB/13 paket; 1 saran system image; 8 paket Go hijau; UI smoke saran→gate→hapus/uninstall→wipe hijau |
| M5–M6 | ⏳ | — |

## Aturan main

- Urutan **M0 → M1 → M2** wajib. M3/M4/M5 boleh paralel setelah M2 (semua memakai `fsx` + `rules`).
- Setiap task punya **acceptance** yang bisa dijalankan lewat perintah, bukan "kelihatan benar".
- `internal/rules` tidak boleh import `internal/clean` (dry-run dijamin read-only).
- Semua test jalan tanpa GUI: `go test ./internal/...` + `cleanlie --dry-run --json`.
- Commit kecil, prefix: `feat(m1):`, `fix(m2):`, `chore(m0):`, `docs:`.

## Ringkasan

| Milestone | Isi | Task | Est. |
|---|---|---|---|
| M0 | Scaffold Wails + flake + tema | T0.1–T0.4 | 0.5 h |
| M1 | Engine scan + katalog rule + dry-run | T1.1–T1.9 | 1 d |
| M2 | Eksekusi bersih + layar Bersihkan | T2.1–T2.8 | 1.5 d |
| M3 | Nix: generasi, GC, layar Nix | T3.1–T3.5 | 1 d |
| M4 | Android: AVD/SDK + layar | T4.1–T4.4 | 0.5 d |
| M5 | Analisis + dashboard | T5.1–T5.7 | 1 d |
| M6 | Settings, i18n, packaging, verifikasi | T6.1–T6.7 | 0.5 d |
| | | **37 task** | **~6 d** |

---

## M0 — Scaffold (gate: `wails dev` tampil di Hyprland)

### T0.1 — `flake.nix`: devShell + package skeleton
- `devShells.default`: go, gopkg, pkg-config, gtk3, webkitgtk_4_1, glib, nodejs (opsional), wails CLI via `go install`.
- `packages.cleanlie` placeholder `buildGoModule` + `makeWrapper`.
- **Acceptance:** `nix develop -c go version` jalan; `nix flake check` tanpa error.
- **Files:** `flake.nix`, `flake.lock`

### T0.2 — Struktur Go + Wails
- `go.mod` (`module github.com/Juragan-Koding/cleanlie`), `main.go`, `app.go`, `wails.json`.
- `frontend/index.html` + `app.js` + `style.css` minimal (satu layar "Cleanlie").
- **Acceptance:** `wails dev` membuka window; `go build ./...` sukses; binary < 20 MB.
- **Files:** `go.mod`, `main.go`, `app.go`, `wails.json`, `frontend/*`

### T0.3 — Shell UI: sidebar, routing, tema
- Sidebar 6 item (Dashboard, Bersihkan, Nix, Android, Analisis, Pengaturan), router hash `#/dashboard`.
- Token warna dark/light di CSS variables, font stack sistem.
- **Acceptance:** klik sidebar ganti layar tanpa reload; resize 1280×800 & 1920×1080 tidak pecah.

### T0.4 — CI lokal + smoke script
- `scripts/dev.sh` (nix develop + wails dev), `make test`/`make lint`, `.gitignore`.
- **Acceptance:** `make test` hijau di repo kosong; `make build` menghasilkan `bin/cleanlie`.
- **Gate M0:** window jalan di Wayland/Hyprland; kalau render rusak → coba `WEBKIT_DISABLE_COMPOSITING_MODE=1` sebelum lanjut.

---

## M1 — Engine (gate: angka cocok dengan `du`)

### T1.1 — `internal/fsx`: walker paralel
- Port `main.go` CleanBro: `Lstat`, `Blocks*512`, hardlink dedupe (`Dev,Ino`), `oneFS`, worker pool, `context` cancel, progress callback.
- **Acceptance:** fixture `t.TempDir()` (sparse file, hardlink ganda, symlink) == `du -s --block-size=1`; hardlink dihitung sekali.
- **Files:** `internal/fsx/walk.go`, `walk_test.go`

### T1.2 — `internal/fsx`: usage, list, human, statfs
- `Usage(path) int64` (dengan cancel), `List(dir)`, `Human(int64)`, `Statfs` untuk dashboard.
- **Acceptance:** `Usage` pada 3 path nyata selisih < 1% vs `du -sk`; unit test untuk `Human`.

### T1.3 — `internal/rules`: tipe inti + planner
- `Rule{ID, Group, Label, Roots, Match, Risk, Action, Detect}`; `Risk` = `safe|warn|danger|blocked`.
- `Plan(ctx) []Item` read-only, item = anak langsung root, `Match` filter.
- **Acceptance:** `Plan` idempoten (2x run hasil sama); test memastikan tidak ada `os.Remove*` di paket ini.

### T1.4 — Katalog Linux/XDG
- `linux.go`: `~/.cache/*` app (opencode, qmd, phpactor, nix, appimage-run, thumbnailer), browser cache, trash, `/var/tmp`, `/var/log` old.
- **Acceptance:** setiap rule punya `Risk`; test guard: tidak ada root di protected list.

### T1.5 — Katalog tool dev
- `devtools.go`: npm (`_cacache`, `_npx`), pnpm (cache + store), gradle (caches per versi + wrapper), cargo, go, uv, pip, dart/pub, composer, playwright, cypress, prisma/typescript/bun/esbuild, maven/yarn/deno (detect-only).
- Tiap entri: `Detect` (LookPath + stat), `Risk`, `Cmd` (perintah resmi) / fallback path.
- **Acceptance:** tool yang tidak terpasang tidak muncul; test fixture `HOME` palsu untuk detect.
- **Files:** `internal/rules/devtools.go`, `devtools_test.go`

### T1.6 — `internal/tools`: eksekusi perintah dengan preview
- `Available(name)`, `Preview(cmd)`, `Run(ctx, cmd, progress)` menangkap stdout/stderr, timeout.
- **Acceptance:** `npm cache clean --force` dan `pnpm store prune` terpanggil di `HOME` sandbox; output tersimpan di hasil.

### T1.7 — `internal/rules`: Android & Nix definitions (read-only)
- `android.go`: AVD (`~/.android/avd/*.ini`), system-images, ndk/build-tools/platforms per versi.
- `nix.go`: path store, profile generations paths (read-only; eksekusi di M3).
- **Acceptance:** parser `*.ini` (fixture) menghasilkan nama/path/API; tidak ada perintah tulis di sini.

### T1.8 — Headless CLI
- `cleanlie --dry-run --json` → `JunkReport` (grup, item, ukuran, risk, command).
- `cleanlie --scan-only --path X` untuk uji scanner.
- **Acceptance:** JSON valid (`jq .`), total per grup == jumlah item, exit 0 tanpa GUI.

### T1.9 — Cross-check vs `du`
- `scripts/verify-sizes.sh`: bandingkan total per rule vs `du -sk`, toleransi 1%.
- **Acceptance:** laporan hijau untuk 10 rule terbesar (`_cacache`, gradle, pnpm, opencode, qmd, phpactor, uv, pub-cache, playwright, cargo).

### Catatan implementasi M1 (2026-11-25)

- **Hasil:** `--dry-run` menemukan 9 grup / 150 item / 88.4 GiB dalam 3,5 s. Temuan terbesar di luar dugaan: `/var/tmp/avd` **17 GiB** (sisa emulator), disusul `_cacache` 7,2 GiB dan `test_devices.avd` 5,1 GiB.
- **Verifikasi:** `scripts/verify-sizes.sh` membandingkan 15 item > 100 MiB; delta 0,00% untuk 14 item, journal 0,99% (perbedaan pembulatan `journalctl`).
- **Test:** `go test -tags desktop,production,webkit2_41 ./...` → 4 paket hijau (fsx, rules, tools, main/CLI).
- **Keputusan teknis:**
  - `Plan` membuang item yang merupakan **induk** dari item lain (`dropAncestors`) agar `~/.cache/ms-playwright` tidak dobel dengan `chromium-1248`.
  - Rule dengan `Priority` lebih tinggi menang saat path sama; ini yang membuat rule dev eksplisit mengalahkan rule generik `~/.cache/*`.
  - Journal memakai probe `journalctl --disk-usage` (fallback walk), bukan `du`.
  - `Rule.Probe != nil` menandai item tanpa path yang bisa di-trash; `fallbackDelete=false`.
- **Deviasi dari rencana:** T1.3 memakai `Plan(ctx, []Rule, opts)` (bukan `Plan(ctx)`) supaya CLI bisa memindai satu direktori dan M4 bisa memfilter katalog. `Roots` menjadi `Path` + `Whole`/`Depth`.
- **Belum ada:** deteksi proses berjalan (M2), eksekusi perintah (M2), cache browser dipisah per profile (M2).

---

## M2 — Bersihkan (gate: hapus nyata + restore dari trash)

### T2.1 — `internal/clean`: trash freedesktop
- `gio trash -- <path>`; fallback manual `~/.local/share/Trash/{files,info}`; path di fs lain → error terstruktur.
- **Acceptance:** fixture 100 MB pindah ke trash, ada `.trashinfo` valid, bisa `gio trash --restore`.
- **Files:** `internal/clean/trash.go`, `trash_test.go`

### T2.2 — `internal/clean`: guard & protected
- `isProtected` (daftar plan §6.3), depth < 2, whitelist root (item di dalam root, kedalaman ≤ `Rule.Depth`).
- **Acceptance:** test menolak `/`, `~`, `~/.config`, `~/.ssh`, item di luar root; mengizinkan `~/.cache/opencode`.

### T2.3 — `internal/clean`: eksekutor + verifikasi freed
- `Execute(ctx, plan, mode)` dengan mode `trash|permanent`, ukur `before/after`, hasil per item (`removed/failed/kept`).
- **Acceptance:** freed == `before-after` (bukan asumsi); item gagal tidak menghentikan sisanya.

### T2.4 — Proses berjalan + audit log
- Cek `/proc/*/comm` (brave, firefox, code, java, node, dart, adb, emulator, cargo, composer) → skip + alasan.
- `~/.local/state/cleanlie/history.jsonl` (JSON per baris, append-only).
- **Acceptance:** dengan proses palsu (`sleep` dinamai), item ter-skip; history bisa dibaca ulang.

### T2.5 — UI: halaman Bersihkan (master-detail)
- Kategori bergrup di kiri (Dev · Node/JVM/Python/Dart, Cache app, Log, Artefak, Trash) + badge ukuran.
- Tabel item kanan: checkbox, path, ukuran, badge `aman/⚠/⛔`, tombol "buka folder".
- Default centang: hanya `safe`; `warn/danger` off.
- **Acceptance:** total terpilih update real-time; filter kategori benar; ⛔ tidak bisa dicentang.

### T2.6 — Scan progress + cancel
- Event `scan:progress` (grup, path, done, total), tombol Cancel, skeleton state.
- **Acceptance:** scan `~/.cache` bisa dibatalkan < 1 s; UI tidak beku.

### T2.7 — Dialog konfirmasi + command preview
- Ringkasan: N item, X GB, mode (Trash/Permanen), daftar perintah yang akan dijalankan.
- Permanen wajib checkbox "saya mengerti tidak bisa di-undo".
- **Acceptance:** batal → tidak ada perubahan; `Plan` tidak berubah antar dialog dan eksekusi.

### T2.8 — Laporan hasil
- Setelah eksekusi: freed aktual, item gagal + alasan, tombol "Buka trash", "Salin log".
- **Acceptance:** freed di UI == `before-after` di log; gagal tampil, bukan silent.

### Catatan implementasi M2 (2026-11-25)

- **Hasil:** layar Bersihkan memakai scan nyata (bukan data demo), default hanya item `safe`,
  gate checkbox untuk hapus permanen, hasil menampilkan `freed` terukur + alasan gagal/skip.
- **Guard:** path absolut → di dalam root katalog → kedalaman ≤ `Item.Depth` → bukan protected →
  proses tidak berjalan. UI hanya mengirim ID; path diambil dari cache scan terakhir.
- **Trash:** `gio trash` dulu, fallback manual `~/.local/share/Trash` + `.trashinfo`; beda
  filesystem → `ErrCrossDevice` dan UI menawarkan permanen. Restore lewat gio terverifikasi.
- **Freed:** `before` dari item, `after` diukur ulang; perintah sukses tapi ukuran tetap → `kept`.
- **Audit:** `~/.local/state/cleanlie/history.jsonl` (0600, append-only), reader toleran baris korup.
- **Deviasi plan §6.2:** whitelist memakai kedalaman relatif root (bukan hanya `Dir(path)==root`)
  karena system image Android 3 level. `TestDefaultCatalogValidates` memvalidasi 151 item nyata.
- **Bug laten M0 yang ditemukan smoke test:** fungsi navigasi global `go()` menimpa `window.go`
  (binding Wails) sehingga diagnostik/integrasi selalu jatuh ke mode demo. Fungsi diganti
  `navigate()`; test jsdom menjaga agar binding tidak tertimpa.
- **Test baru:** `internal/clean` (guard, trash, executor, audit, proses), `app_clean_test.go`,
  `frontend/test/ui-smoke.mjs` (jsdom, opsional, tanpa Node saat build).
- **Belum:** aksi root (M3), cache browser per profil & saran AVD (M4).

---

## M3 — Nix (gate: GC dry-run == CLI)

### T3.1 — Parser generasi
- `nix-env -p <profile> --list-generations` untuk user dan home-manager; system memakai symlink `system-<N>-link` + mtime karena `nix-env` gagal membuka `system.lock` sebagai non-root.
- **Acceptance:** jumlah generasi == output CLI (dibandingkan sebagai root); fixture string asli di test.

### T3.2 — Ukuran closure per generasi
- `nix path-info -S` per batch 300 path unik (bukan `-Sh` per generasi); fallback kosong bila gagal.
- **Acceptance:** angka == `nix path-info -Sh` manual; generasi yang link-nya hilang ditandai `linkMissing`.

### T3.3 — Dry-run GC
- `nix-store --gc --print-dead` → daftar path; ukuran dari `nix path-info --json` per 300 path. Fallback `nix-collect-garbage --dry-run` (jumlah saja).
- **Acceptance:** jumlah path == `nix-collect-garbage --dry-run` persis; tidak ada perubahan store setelah dry-run.

### T3.4 — `internal/privileged`: hybrid pkexec
- Deteksi agent dengan memindai `/proc/*/cmdline` (`hyprpolkitagent`, atau `polkit` + `agent`), `pkexec` dengan path absolut, fallback perintah `sudo` siap-copy + tombol Copy.
- **Acceptance:** tanpa agent → UI menampilkan perintah, exit code & stderr terlihat; dengan agent → prompt muncul.
- **Files:** `internal/privileged/privileged.go` (+ test)

### T3.5 — Layar Nix
- Kartu store size, dead paths (dry-run), tabel generasi (profile/gen/tanggal/closure/aksi), tombol GC & hapus generasi.
- Konfirmasi hapus generasi: ketik `GC`; tombol disabled sampai cocok.
- **Acceptance:** hapus generasi home-manager lama (`>30d`) mengurangi list; generasi current tidak pernah bisa dipilih.

### Catatan implementasi M3 (2026-11-25)

- Package baru: `internal/nix` (`nix.go`, `parse.go`, `actions.go`), `internal/privileged`; binding `app_nix.go`; CLI `--nix-status` / `--nix-dead`.
- Angka nyata: store 70.425.698.432 byte NAR / 47.208 path / 129 generasi; dead 624 path / 9.720.894.784 byte; journal 4.187.593.113 byte; status 7,5 s; dead 31,6 s.
- Gate M3 lolos: `--nix-dead` **624** == `nix-collect-garbage --dry-run` **624**.
- Tidak ada agent polkit di mesin target → semua aksi root berhenti di `NeedsSudo` + `sudoCommand`; UI menampilkan tombol Salin, bukan klaim berhasil.
- Deviasi: tanggal system dari mtime; closure batch `-S`; ukuran dead dari NAR path-info; deteksi agent lewat `/proc`; satu file privileged.
- Detail: [`../implementation/m3-nix.md`](../implementation/m3-nix.md).

---

## M4 — Android (gate: hapus AVD benar-benar membebaskan ruang)

### T4.1 — Parser AVD
- Baca `~/.android/avd/*.ini` + `config.ini` → nama, API, ABI, ukuran (`userdata-qemu.img`), mtime "terakhir dipakai".
- **Acceptance:** hasil == `avdmanager list avd`; test fixture 2 AVD.

### T4.2 — Parser SDK
- `~/Android/Sdk/{system-images,ndk,build-tools,platforms,emulator}` per versi + ukuran; tandai yang direferensikan AVD.
- **Acceptance:** system image yang dipakai AVD ditandai `dipakai` dan tidak disarankan hapus.

### T4.3 — Aksi Android
- `avdmanager delete avd -n <name>`, `sdkmanager --uninstall "<pkg>"`, wipe (`emulator -avd X -wipe-data` saat mati).
- Cek emulator tidak jalan sebelum aksi.
- **Acceptance:** di fixture, hapus AVD menghilangkan folder + `.ini`; sdkmanager dipanggil dengan path absolut.

### T4.4 — Layar Android
- Tabel AVD, tabel system-image/NDK, saran otomatis (`>90 hari` & tidak dipakai), default tanpa centang.
- **Acceptance:** "Hapus yang disarankan" hanya menyentuh item yang disarankan; konfirmasi eksplisit.

### Catatan implementasi M4 (2026-11-25)

- **AVD:** parser membaca `~/.android/avd/*.ini` + `config.ini`; **tidak** memakai
  `avdmanager list avd` karena di mesin target perintah itu mengembalikan daftar kosong
  meski `test_devices.avd` ada. `lastUsed` = mtime terbaru dari `emu-launch-params.txt`,
  `snapshots/`, `userdata-qemu.img*`, `cache.img*`. Saran hanya bila usia >90 hari,
  tidak berjalan, dan tidak error.
- **SDK:** paket `system-images`, `ndk`, `build-tools`, `platforms`, `emulator`; system
  image yang dipakai AVD ditandai `UsedBy` dan tidak disarankan. `cmdline-tools`, `cmake`,
  `platform-tools` belum dihitung ke kartu SDK (di luar scope T4.2).
- **Aksi:** `avdmanager delete avd -n`, `sdkmanager --sdk_root=... --uninstall`,
  `emulator -avd ... -wipe-data -no-window -no-snapshot -quit-after-boot 180`.
  Semua path tool absolut. Emulator hidup memblokir semua aksi. Fallback hapus folder +
  `.ini` hanya untuk AVD yang tidak dikenali `avdmanager`, dan tetap divalidasi di
  dalam `AVDHome`.
- **Nyata:** AVD `test_devices` 5.381.066.752 byte (usia 1 hari → tidak disarankan);
  SDK 14.833.647.616 byte; saran `system-images;android-35;google_apis;x86_64`
  3.754.844.160 byte (154 hari, tidak dipakai); `--android-status` 137 ms.
- **Gate:** fixture membuktikan hapus AVD menghilangkan folder + `.ini` dan `freed>0`;
  AVD asli tidak dihapus saat pengembangan. UI default tanpa centang; gate checkbox
  eksplisit untuk hapus dan wipe.
- Detail: [`../implementation/m4-android.md`](../implementation/m4-android.md).

---

## M5 — Analisis + Dashboard

### T5.1 — `analysis/tree.go`
- Tree ukuran berkedalaman N (reuse `fsx`), cache hasil 60 s, cancel.
- **Acceptance:** tree `~` depth 2 == output `cleanbro -d 2` (port pembanding); < 10 s untuk `~/.cache`.

### T5.2 — `analysis/large.go`
- File ≥ ambang (default 100 MB), top-N, sortable.
- **Acceptance:** tidak ada hasil duplikat; ukuran allocated (bukan logical).

### T5.3 — `analysis/duplicates.go`
- Group by size → hash 64 KB pertama → SHA-256 penuh; hanya file > 1 MB.
- **Acceptance:** fixture 3 file identik + 1 mirip → 1 grup benar; tidak menghash file unik.

### T5.4 — `analysis/stale.go`
- mtime > 365 hari, exclude protected & `.git`, top-N.
- **Acceptance:** fixture mtime diatur `os.Chtimes`; hasil sesuai.

### T5.5 — `analysis/artifacts.go`
- Cari `node_modules`, `target`, `dist`, `build`, `.next`, `.venv`, `__pycache__`, `.gradle`, `.dart_tool`, `vendor`; path proyek + ukuran + mtime.
- **Acceptance:** `node_modules` di dalam `node_modules` tidak dihitung dua kali; proyek mtime < 7 hari diberi tanda.

### T5.6 — Layar Analisis
- Tab: Disk usage (bar + breadcrumb + drill-down), File besar, Duplikat, File lama, Artefak build.
- Aksi: buka folder, pindah ke trash (lewat guard yang sama).
- **Acceptance:** drill-down 3 level tanpa reload; buka folder memakai `xdg-open`.

### T5.7 — Dashboard
- Kartu: Disk /, Nix store, Cache dev, Journal; top-5 reclaim; tombol "Scan ulang" & "Bersihkan yang aman".
- **Acceptance:** angka kartu == hasil `Plan`/`NixStatus`; tombol memakai alur M2 (bukan jalur hapus baru).

---

## M6 — Polish & packaging

### T6.1 — Settings + persistensi
- `~/.config/cleanlie/config.json`: excluded paths, scan roots, target journal, ambang, mode hapus, bahasa, tema.
- **Acceptance:** restart mempertahankan setting; config invalid → fallback default + warning.

### T6.2 — i18n ID/EN
- Tabel string `frontend/src/i18n.js`; semua label lewat `t()`.
- **Acceptance:** tidak ada string hardcoded di view; ganti bahasa tanpa reload.

### T6.3 — Tema & aksesibilitas
- System/light/dark, fokus keyboard, `aria-label` pada checkbox/tombol, kontras ≥ 4.5:1.
- **Acceptance:** navigasi Tab penuh di layar Bersihkan; badge terbaca di kedua tema.

### T6.4 — `nix build .#cleanlie`
- `buildGoModule` + `makeWrapper` (PATH absolut: nix, journalctl, pkexec, gio, xdg-open; `GDK_BACKEND`, `WEBKIT_DISABLE_COMPOSITING_MODE`).
- **Acceptance:** `nix run .#cleanlie` membuka window; `nix build` reproducible (2 build hash sama).

### T6.5 — Desktop entry + ikon
- `share/applications/cleanlie.desktop`, ikon SVG (sapu, tema CleanBro), `StartupWMClass`.
- **Acceptance:** muncul di launcher Hyprland; window ter-asosiasi ke ikon.

### T6.6 — README + dokumentasi pemakaian
- Instalasi (flake), izin, daftar kategori + risiko, cara restore dari trash, catatan polkit, batasan (GC butuh konfirmasi).
- **Acceptance:** orang lain bisa install dari README tanpa bertanya.

### T6.7 — Verifikasi akhir (checklist plan §11)
- Unit test semua paket, cross-check `du` 10 rule, GC dry-run == CLI, hapus generasi lama, hapus/wipe AVD fixture, screenshot 6 layar.
- **Acceptance:** semua checklist §11 hijau; tidak ada item ⛔ yang bisa dieksekusi.

---

## Definisi selesai (v1)

1. `nix run .#cleanlie` membuka window di Hyprland tanpa env manual.
2. Bisa scan + dry-run + hapus (trash) untuk semua kategori `safe`; item `warn/danger` butuh konfirmasi.
3. Nix: lihat generasi + closure, dry-run GC, hapus generasi lama dengan konfirmasi ketik `GC`.
4. Android: lihat AVD/SDK, hapus/wipe dengan default aman.
5. Analisis: tree, file besar, duplikat, file lama, artefak build.
6. Semua angka ukuran diverifikasi terhadap `du`/`nix path-info` (toleransi 1%).
7. `history.jsonl` mencatat setiap aksi; tidak ada jalur hapus yang melewati guard.
