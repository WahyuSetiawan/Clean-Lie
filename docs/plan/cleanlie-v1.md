# Cleanlie v1 — Plan (GUI cleaner untuk NixOS)

> Status: **M6 selesai** (settings, i18n, packaging) · v1 lengkap
> Target mesin: NixOS 26.11, Hyprland/Wayland, uid 1000 (`juragankoding`)

## 1. Ringkasan

Cleanlie adalah aplikasi desktop (window) untuk membersihkan disk di Linux/NixOS, mengambil
ide dari [CleanBro](https://github.com/jetorbitdev/cleanbro) (macOS) tapi disesuaikan dengan
kenyataan NixOS: Nix store 82G (≈34G dead paths), cache tool dev **24.8G**, Android SDK/AVD **28.2G**,
`~/.cache` 23G, `/var/log` 4.1G (journal 4G).

Tiga pilar:

1. **Bersihkan** — kategori junk yang aman & terukur (cache dev per-tool: npm/pnpm/gradle/cargo/uv/dart/Android, cache XDG, log & journal, trash, artefak build).
2. **Nix** — lihat isi store, generasi lama (system / user / home-manager), jalankan GC.
3. **Analisis** — disk usage tree, file terbesar, duplikat, file lama.

Keputusan yang sudah disepakati:

| Aspek | Pilihan |
|---|---|
| Interface | **GUI window** (bukan TUI/CLI-only) |
| Bahasa | **Go** |
| Hapus default | **Dry-run dulu**, eksekusi → **trash** (bisa restore), `--permanent` opsional |
| Scope v1 | Nix GC+generasi, cache dev per-tool (npm/pnpm/gradle/cargo/go/uv/dart/composer/playwright/cypress), Android SDK+AVD, cache XDG, log & journal, analisis disk |

## 2. Apa yang diambil dari CleanBro

| Bagian CleanBro | Dipakai? | Catatan |
|---|---|---|
| `main.go` scanner paralel (allocated blocks, hardlink dedupe, one-fs, top-N heap) | ✅ port langsung | `syscall.Stat_t` (Blocks/Dev/Ino/Nlink) identik di Linux |
| `Cleaner.swift` JunkScanner (root → anak = 1 item, ukur paralel) | ✅ pola | ganti daftar path macOS → Linux/NixOS |
| `isProtected()` + deepRemove + unlock | ✅ konsep | daftar path Linux sendiri; `unlock` via chmod, bukan `lchflags` |
| Trash macOS | ❌ | Linux pakai `gio trash` (freedesktop Trash spec), fallback manual |
| Uninstaller (app bundle macOS) | ❌ v1 | Linux tidak ada konsep app bundle; diganti "hapus flatpak/appimage/sisa config" nanti |
| Homebrew tool | ❌ | padanan Nix = modul Nix |
| Duplicate / Large / Stale / Disk tree | ✅ | dari `Duplicates.swift`, `AnalysisViews.swift`, `Scanner.swift` |
| Home dashboard + Other space | ✅ sebagian | padanan: bar per mount, breakdown `/nix/store`, cache, journal |
| SwiftUI | ❌ | diganti web UI |

## 3. Stack: Wails v2 (Go + web UI), bukan Fyne

**Status: DIKUNCI (2026-11-22).** Modul: `github.com/Juragan-Koding/cleanlie`, binary: `cleanlie`.

Dipilih **Wails v2**:

- Backend Go asli (scanner & rule engine bisa di-unit-test tanpa UI).
- UI = HTML/CSS/JS vanilla tanpa bundler → **tidak butuh Node saat build**, aset di `//go:embed`.
- Hasil akhir satu binary statis (~12–15 MB) + `desktop entry` → cocok dipaketkan jadi Nix derivation.
- Fidelity visual paling dekat dengan referensi (sidebar, kartu, tabel, dark/light).
- Window native GTK3 + WebKitGTK 4.1 (tersedia di nixpkgs).

Risiko utama: WebKitGTK di Wayland/Hyprland kadang perlu `WEBKIT_DISABLE_COMPOSITING_MODE=1`.
Mitigasi: set env itu di wrapper kalau terdeteksi render bermasalah; **fallback plan**: Fyne v2
(pure Go, tanpa WebKit) kalau Wails gagal total di mesin ini — UI jadi lebih sederhana, engine tetap sama.

Catatan: `go` belum terpasang → `nix shell nixpkgs#go nixpkgs#pkg-config nixpkgs#webkitgtk_4_1 nixpkgs#gtk3`.

**Alasan tidak memilih Fyne** (tercatat supaya tidak didiskusikan ulang): Fyne tidak punya tabel/treemap
siap pakai, jadi layar Bersihkan (ratusan item bergrup) harus digambar manual dan tetap terasa kaku.
**Alasan tidak memilih GTK4/gotk4**: CGO berat, binding tertinggal, kode UI jauh lebih banyak.
Fyne tetap disimpan sebagai **fallback darurat** (engine & layar tidak berubah, hanya lapisan view).

## 4. Arsitektur

```
cleanlie/
├── flake.nix                  # devShell + package + desktop item
├── go.mod  (module github.com/Juragan-Koding/cleanlie)
├── main.go                    # wails.Run + flag mode headless (--scan-json, --dry-run)
├── app.go                     # App struct: binding untuk UI + events
├── internal/
│   ├── fsx/                   # walker: allocated size, hardlink dedupe, one-fs, worker pool
│   │   ├── walk.go            # port main.go CleanBro
│   │   ├── usage.go           # Usage(path), List(dir), human()
│   │   └── walk_test.go
│   ├── rules/                 # katalog junk: definisi kategori + dry-run planner
│   │   ├── rules.go           # Rule{ID, Label, Roots, Match, Risk, Action}
│   │   ├── linux.go           # path Linux/XDG
│   │   ├── devtools.go        # cache tool dev: npm/pnpm/gradle/cargo/uv/dart/composer/go
│   │   ├── android.go         # AVD, system-images, NDK, build-tools, platforms
│   │   ├── nix.go             # path Nix + perintah GC (hanya baca di sini)
│   │   └── rules_test.go
│   ├── tools/                 # jalankan perintah resmi paket manager (dengan command preview)
│   │   ├── tools.go           # Detect(tool), Run(cmd), output parsing
│   │   └── tools_test.go
│   ├── clean/                 # eksekusi: trash, permanent, confirm gating
│   │   ├── trash.go           # gio trash + fallback .trashinfo
│   │   ├── guard.go           # isProtected(), mustExist(), running-process check
│   │   └── clean_test.go
│   ├── nix/                   # generasi, closure size, dry-run GC, jalankan GC (pkexec/sudo)
│   │   ├── generations.go
│   │   ├── store.go
│   │   └── nix_test.go
│   ├── analysis/              # disk tree, large files, duplicates, stale, dev artifacts
│   │   ├── tree.go  large.go  duplicates.go  stale.go  artifacts.go
│   │   └── *_test.go
│   ├── android/               # parse AVD/SDK, mtime pemakaian, saran hapus
│   │   ├── avd.go  sdk.go
│   │   └── android_test.go
│   └── privileged/            # wrapper pkexec untuk aksi root (journal vacuum, GC system)
│       └── exec.go
└── frontend/
    ├── index.html  app.js  style.css
    └── src/  (view per layar: home, cleanup, nix, tree, large, dupes, stale, settings)
```

Aturan dependency: `frontend` → `app.go` → `internal/*`. `internal/rules` **tidak boleh** meng-import
`internal/clean` (pemisahan perencanaan vs eksekusi, supaya dry-run dijamin tidak menyentuh disk).

## 5. Katalog aturan v1 (Linux/NixOS)

Format: **Kategori** → item = anak langsung dari tiap root (sama seperti CleanBro), ukuran = allocated blocks.

Semua angka di bawah **hasil ukur nyata di mesin ini** (`du -sk`, 2026-11-22).

### 5.1 Cache XDG & aplikasi  (~9G, di luar cache tool dev)
| Root | Ukuran | Risiko | Catatan |
|---|---|---|---|
| `~/.cache/opencode` | 3.0 G | aman | regenerate |
| `~/.cache/qmd` | 2.3 G | aman | index/embedding, regenerate |
| `~/.cache/phpactor` | 2.2 G | aman | index PHP, regenerate |
| `~/.cache/codebase-memory-mcp` | 1.0 G | aman | |
| `~/.cache/BraveSoftware` | 1.2 G | **skip kalau Brave jalan** | cek `/proc/*/comm` |
| `~/.cache/mozilla` | 0.7 G | skip kalau Firefox jalan | |
| `~/.cache/nix` | 0.9 G | aman | eval cache Nix |
| `~/.cache/appimage-run`, `~/.cache/Thumbnailer`, `~/.cache/fontconfig`, `~/.cache/mesa_shader_cache` | <0.5 G | aman | |
| `~/.local/share/Trash/files` | 2.2 M | aman | = "isi Trash" |

### 5.2 Cache tool dev  (**~24.8 G** — target utama v1; Android dipisah ke §5.3)

Ini bagian yang diminta: folder dev tool yang benar-benar ada di komputer ini. Ukuran terukur
(tanpa Android, tanpa cache aplikasi XDG):

| Tool | Path | Ukuran | Cara bersihkan yang dipakai | Risiko |
|---|---|---|---|---|
| **npm** | `~/.npm/_cacache` | **7.0 G** | `npm cache clean --force` | aman |
| | `~/.npm/_npx` | **2.6 G** | hapus (cache `npx`, regenerate) | aman |
| **Gradle** | `~/.gradle/caches/{9.3.1,7.5,...}` | **3.5 G** | hapus versi lama; sisakan versi yang dipakai proyek aktif (`gradle --version`) | aman — build berikutnya re-download |
| | `~/.gradle/wrapper/dists` | **1.0 G** | hapus dist lama | aman |
| **pnpm** | `~/.cache/pnpm` | **2.1 G** | `pnpm store prune` | aman |
| | `~/.local/share/pnpm/store` | **1.2 G** | `pnpm store prune` (bukan `rm -rf store`) | aman |
| **Playwright** | `~/.cache/ms-playwright` | **1.9 G** | hapus browser lama / `PLAYWRIGHT_BROWSERS_PATH=0` | sedang — `npx playwright install` ulang |
| **uv** | `~/.cache/uv` | **1.5 G** | `uv cache prune` / `uv cache clean` | aman |
| | `~/.local/share/uv` | **0.3 G** | `uv cache prune` | aman |
| **Dart/Flutter** | `~/.pub-cache/hosted` | **1.5 G** | `dart pub cache clean` | aman (re-download) |
| **Cypress** | `~/.cache/Cypress` | **0.8 G** | hapus versi lama, sisakan yang dipakai `package.json` | sedang |
| **Cargo** | `~/.cargo/registry` | **0.5 G** | hapus `registry/cache` + `registry/src` (bukan `bin/`) | aman |
| | `~/.cargo/git` | 16 K | hapus | aman |
| **Go** | `~/go/pkg/mod` | **0.4 G** | `go clean -modcache` | aman (go belum ada di PATH; modul lama) |
| | `~/.cache/go-build`, `~/.cache/gopls` | 20 M | `go clean -cache` / hapus | aman |
| **Composer** | `~/.cache/composer` | **0.2 G** | `composer clear-cache` | aman |
| **pip** | `~/.cache/pip` | 0.1 G | `pip cache purge` (pip tidak di PATH → hapus folder) | aman |
| **TypeScript/Prisma/bun/esbuild** | `~/.cache/{typescript,prisma,bun,...}` | <0.4 G | hapus | aman |
| **Maven** | `~/.m2/repository` | tidak ada di mesin ini | tampil hanya kalau ada | aman |
| **Yarn/Deno/Bun store** | `~/.cache/yarn`, `~/.deno`, `~/.bun` | tidak ada | deteksi otomatis | aman |

Aturan tambahan untuk kategori ini:

- **Deteksi dulu, sembunyikan kalau tidak ada.** Setiap entri punya `Detect` (`exec.LookPath` tool +
  `stat` path). Yang tidak ada tidak ditampilkan, jadi daftar tetap pendek dan relevan.
- **Pakai perintah resmi kalau ada** (`npm cache clean`, `pnpm store prune`, `uv cache prune`,
  `composer clear-cache`, `go clean -modcache`, `dart pub cache clean`), bukan `rm -rf`,
  supaya index/metadata tool tetap konsisten. Kalau tool tidak terpasang → fallback hapus folder.
- **Jangan hapus saat tool jalan.** Cek proses: `java`/`gradle`, `node`/`pnpm`/`npm`, `dart`, `go`,
  `cargo`, `adb`/`emulator`, `composer`. Item ditandai ⛔ dan di-skip, bukan dipaksa.
- **Gradle multi-versi**: tampilkan ukuran per folder versi + versi yang dipakai proyek aktif;
  default centang hanya versi yang **bukan** versi terbaru-terpakai.
- **`_npx` dan `_cacache` dipisah** supaya user bisa pilih: `_npx` cepat regenerate, `_cacache` 7G
  berisi semua tarball paket.

### 5.3 Android SDK & emulator (28 G — kategori sendiri)

Dipisah karena risikonya beda dengan cache biasa: bukan "regenerate", tapi "hilang & harus download
ulang / bikin AVD baru". Layar khusus **Android** dengan tiga blok:

| Blok | Isi | Aksi |
|---|---|---|
| AVD | daftar AVD (`~/.android/avd/*.ini`): nama, API level, ukuran, **terakhir dipakai** (mtime `*.avd/userdata-qemu.img`) | `avdmanager delete avd -n <name>`, tombol **Wipe data** per AVD |
| System images | `~/Android/Sdk/system-images/<api>/<tag>/<abi>` + ukuran | `sdkmanager --uninstall "..."`, tandai yang dipakai AVD aktif |
| NDK / build-tools / platforms | `~/Android/Sdk/{ndk,build-tools,platforms}` per versi | hapus versi lama; cek `local.properties`/`gradle` proyek untuk versi terpakai |

Default: **tidak ada yang dicentang**. Rekomendasi otomatis hanya untuk AVD dengan mtime > 90 hari dan
system image yang tidak direferensikan AVD mana pun.

### 5.4 Log & journal  (~4.1G)
| Target | Aksi | Risiko |
|---|---|---|
| Journal system | `journalctl --vacuum-size=<target>` (default 500M) via pkexec | aman, log lama hilang |
| Journal user | `journalctl --user --vacuum-size` (tanpa root) | aman |
| `/var/log/*.old`, `*.gz`, `*.xz` > 30 hari | trash/permanent | aman |
| `/var/tmp/*` > 30 hari | trash | aman |
| `~/.local/state/*/log`, core dumps (`coredumpctl`) | list + hapus | aman |

### 5.5 Artefak build di dalam proyek (analisis, bukan auto)
Scan `~/Projects`/`~` dengan kedalaman terbatas untuk direktori bernama:
`node_modules`, `target`, `dist`, `build`, `.next`, `.turbo`, `.venv`, `__pycache__`, `.pytest_cache`,
`.gradle`, `.dart_tool`, `vendor`.
Aturan aman: tampil sebagai daftar (path proyek + ukuran + mtime); **tidak pernah dicentang otomatis**;
peringatan kalau mtime < 7 hari (proyek sedang aktif). Berlaku juga untuk `~/.local/share/docker`? tidak — itu data.

### 5.6 Perintah paket manager (dipanggil, bukan `rm`)

`npm cache clean --force`, `pnpm store prune`, `uv cache prune`, `composer clear-cache`,
`go clean -modcache`, `dart pub cache clean`, `gradle --stop` (sebelum hapus cache Gradle),
`docker system prune` (konfirmasi + hanya kalau daemon hidup), `nix store gc`,
`flatpak uninstall --unused` (kalau ada).
Semua perintah ditampilkan lebih dulu (command preview: perintah + path + ukuran) sebelum dieksekusi.

### 5.7 Nix (modul sendiri)
| Fitur | Perintah / sumber data | Root? |
|---|---|---|
| Ukuran store | `nix path-info --json --all` → jumlah `narSize` (≈5 s; `du` 75 s) | tidak |
| Estimasi reclaimable (dead paths) | `nix-store --gc --print-dead` + `nix path-info --json` per 300 path (≈32 s, byte NAR); fallback `nix-collect-garbage --dry-run` (jumlah saja) | tidak (dry-run) |
| Generasi system | symlink `/nix/var/nix/profiles/system-<N>-link` + mtime; `nix path-info -S` batch untuk closure | baca: tidak, hapus: ya |
| Generasi user | `~/.local/state/nix/profiles/profile` + `nix-env -p <path> --list-generations` | tidak |
| Generasi home-manager | `~/.local/state/nix/profiles/home-manager` + `nix-env -p <path> --list-generations` | tidak |
| Hapus generasi lama | UI memilih nomor non-current → `nix-env -p <profile> --delete-generations <N...>` | system: root |
| GC store | `nix-collect-garbage -d` (root) / `nix store gc` (user roots saja) | ya |
| GC terjadwal | `nix.gc.automatic` di konfigurasi NixOS (aplikasi hanya menampilkan status) | — |

Aturan keras: GC dan hapus generasi **selalu** dua langkah — tampilkan dry-run + angka, user harus
mengetik `GC` di dialog konfirmasi. Tidak ada auto-GC dari aplikasi.

Catatan implementasi (2026-11-25): `nix-env` pada profile system gagal sebagai non-root
(`system.lock`), jadi tanggal system diambil dari mtime link. `nix-collect-garbage --dry-run`
tidak memberi ukuran byte; gate M3 membandingkan **jumlah path** dengan `--nix-dead` (624 == 624).
Store dan dead-path memakai NAR size, bukan allocated block. Detail: `implementation/m3-nix.md`.

## 6. Model keamanan

1. **Dry-run pertama.** `rules.Plan()` hanya membaca; UI menampilkan "akan membebaskan X".
   Eksekusi hanya lewat `clean.Execute(plan, mode)` dengan `plan` yang sudah ditampilkan.
2. **Whitelist root.** Cleanup hanya menerima item yang berada di dalam root katalog, dengan
   kedalaman ≤ `Rule.Depth` (default 1 = anak langsung). Root itu sendiri hanya boleh bila item
   bertanda `Whole` (mis. satu direktori `~/.gradle/caches`). UI tidak pernah mengirim path bebas.
3. **Protected list** (tidak pernah dihapus, walau dipilih):
   `/`, `/nix`, `/nix/store`, `/nix/var`, `/etc`, `/usr`, `/bin`, `/sbin`, `/boot`, `/var/lib`,
   `/home`, `~`, `~/Documents`, `~/Pictures`, `~/Videos`, `~/Music`, `~/Projects` (root-nya),
   `~/.config`, `~/.local/share`, `~/.ssh`, `~/.gnupg`, `~/.local/share/keyrings`, `~/.password-store`.
   Plus: path dengan komponen < 2, dan semua item di luar home kecuali root katalog terdaftar.
4. **Trash dulu.** `gio trash` (freedesktop, restore-able). Kalau gagal karena beda filesystem
   (`/nix`, `/var`), tampilkan dialog: *"tidak bisa ke trash, hapus permanen?"*.
5. **Proses berjalan.** Sebelum hapus cache browser/editor, cek `/proc/*/comm` (brave, firefox, code,
   node). Kalau hidup → item di-skip dengan alasan, bukan dipaksa.
6. **Verifikasi setelah hapus.** Ukur ulang path; `freed = before - after` (bukan asumsi), gagal
   dicatat per item dan ditampilkan di laporan hasil.
7. **Log audit.** `~/.local/state/cleanlie/history.jsonl`: waktu, kategori, path, ukuran, mode, hasil.
   Bisa dipakai untuk "undo hint" (path asal di trash).
8. **Root minimal — hybrid pkexec/sudo (dikunci).** Hanya 3 aksi yang butuh root: hapus generasi
   system, GC system, vacuum journal system. Alur:
   1. Saat start, deteksi polkit agent dengan memindai `/proc/*/cmdline` (`hyprpolkitagent`, atau
   `polkit` + `agent`).
   2. Agent ada → jalankan via `pkexec` dengan **path absolut** (env pkexec di-strip, PATH nix hilang).
   3. Agent tidak ada / pkexec gagal (exit 126/127) → tampilkan dialog berisi perintah `sudo` siap-copy
      + tombol **Copy**, dan tombol **Coba pkexec lagi**. Jangan pernah gagal senyap.
   4. Semua perintah root ditampilkan utuh sebelum dieksekusi (command preview).

## 7. Layar GUI

> **Wireframe interaktif (referensi visual resmi):** [`wireframe/index.html`](../wireframe/index.html)
> — buka dengan `xdg-open docs/wireframe/index.html`. Wireframe memakai angka ukur nyata mesin ini
> dan memetakan tiap layar ke task M0–M6 (lihat [`wireframe/README.md`](../wireframe/README.md)).
> Mockup ASCII di bawah adalah ringkasan; kalau ada perbedaan, wireframe yang jadi acuan.

Sidebar tetap (ala CleanBro):

```
┌───────────────┬──────────────────────────────────────────────────────────────┐
│ 🧹 Cleanlie   │  Dashboard                                                   │
│               │  ┌───────────┬───────────┬───────────┬───────────┐         │
│ ▸ Dashboard   │  │ Disk /    │ Nix store │ Cache dev │ Journal   │         │
│ ▸ Bersihkan   │  │ 104/139G  │ 82.0 G    │ 24.8 G    │ 4.0 G     │         │
│ ▸ Nix         │  │ ██████░80%│ 34 G bisa │ 10 G bisa │ target500M│         │
│ ▸ Android     │  └───────────┴───────────┴───────────┴───────────┘         │
│ ▸ Analisis    │  ┌──────────────────────────────────────────────────────┐   │
│    Disk usage │  │ Reclaim teratas: avd/ 14.2G · _cacache 7.0G · Sdk/  │   │
│    File besar │  │ 14.0G · gradle 3.5G · opencode 3.0G · npm/_npx 2.6G  │   │
│    Duplikat   │  └──────────────────────────────────────────────────────┘   │
│    File lama  │  [ Scan ulang ]  [ Bersihkan yang aman (12 G) ]              │
└───────────────┴──────────────────────────────────────────────────────────────┘
```

**Bersihkan** — master-detail, item bisa dicentang, dikelompokkan per tool:

```
┌ Kategori ──────────┬ Item (Dev · Node) ──────────────────────────────────┐
│ ▸ Dev · Node (10G) │ ☑ ~/.npm/_cacache       7.0 G  aman   npm cache clean│
│   Dev · JVM (4.5G) │ ☑ ~/.npm/_npx           2.6 G  aman                 │
│   Dev · Android    │ ☑ ~/.local/share/pnpm/store 1.2 G  pnpm store prune │
│     AVD (14.2G)    │ ☑ ~/.cache/pnpm         2.1 G  aman                 │
│   Dev · Python     │ ☐ ~/.cache/ms-playwright 1.9 G ⚠ download ulang    │
│   Dev · Dart (1.5G)│ ☐ ~/.cache/BraveSoftware 1.2 G ⛔ Brave sedang jalan│
│   Cache app (9G)   │                                                     │
│   Log & journal    │                                                     │
│   Artefak build    │                                                     │
│   Trash            │                                                     │
└────────────────────┴─────────────────────────────────────────────────────┘
  6 item dipilih · 14.8 G   [ Pindah ke Trash ]  [ Hapus permanen… ]
```

**Nix** — kartu + tabel generasi:

```
 Store /nix/store  82.0 G        Dead paths (dry-run) ~34.5 G
 ┌ Profile ──────────────┬ Gen ┬ Tanggal ──────────┬ Closure ┬ ──────────┐
 │ system                │ 51  │ 2026-11-22 10:12  │ 6.1 G   │ (aktif)  │
 │ system                │ 50  │ 2026-11-15 21:04  │ 5.9 G   │ [hapus]  │
 │ home-manager          │ 719 │ 2026-11-22 12:31  │ 2.2 G   │ (aktif)  │
 │ user (nix-env)        │ 12  │ ...               │ 1.1 G   │ [hapus]  │
 └───────────────────────┴─────┴───────────────────┴─────────┴──────────┘
 [ Hapus generasi > 30 hari ]  [ Jalankan GC (dry-run dulu) ]  [ Vacuum journal → 500M ]
```

**Android** — kategori berisiko tinggi, default tidak ada yang dicentang:

```
 Android  28.2 G  ·  emulator jalan: tidak
 ┌ AVD ──────────────┬ API ┬ Ukuran ┬ Terakhir dipakai ┬ ───────────────┐
 │ voi_api35         │ 35  │ 8.9 G  │ 2026-11-18       │ [Wipe] [Hapus] │
 │ test_devices      │ 33  │ 5.3 G  │ 2026-06-02       │ [Wipe] [Hapus] │
 └───────────────────┴─────┴────────┴──────────────────┴───────────────┘
 ┌ System images ──────────────────────────┬ Ukuran ┬ Dipakai AVD ┬ ────┐
 │ system-images;android-33;google_apis;x86_64 │ 3.5 G │ test_devices │    │
 │ system-images;android-35;google_apis;x86_64 │ 4.3 G │ voi_api35    │    │
 └────────────────────────────────────────┴────────┴──────────────┴────┘
 Saran: test_devices > 90 hari tidak dipakai (5.3 G)   [ Hapus yang disarankan ]
```

**Analisis disk** — tree dengan bar + breadcrumb, klik folder untuk masuk, tombol "buka di file manager".
**File besar** (≥100 MB), **Duplikat** (size → hash parsial 64 KB → SHA-256), **File lama** (>1 tahun).
**Artefak build** — daftar `node_modules`, `target`, `.venv`, `dist`, `build`, `__pycache__` per proyek
(path proyek + ukuran + mtime, urut terbesar).

**Pengaturan**: path yang dikecualikan, root tambahan untuk scan artefak, ukuran target journal, ambang file
besar/lama, trash vs permanent, **default centang hanya item risiko aman** (⚠/⛔ selalu off),
bahasa (ID/EN), tema (system/light/dark).

## 8. Kontrak Go ↔ UI (binding Wails)

```go
// app.go
func (a *App) ScanJunk() (JunkReport, error)              // baca saja
func (a *App) Clean(req CleanRequest) (CleanResult, error) // mode: trash|permanent, dryRun
func (a *App) NixStatus() (NixStatus, error)               // store size, dry-run GC, generasi
func (a *App) NixDeleteGenerations(req GenRequest) (NixStatus, error) // butuh konfirmasi teks
func (a *App) NixCollectGarbage(confirm string) (CleanResult, error)
func (a *App) VacuumJournal(target string) (CleanResult, error)
func (a *App) ScanTree(root string, depth int) (TreeNode, error)
func (a *App) ScanLarge(root string, minBytes int64) ([]FileItem, error)
func (a *App) ScanDuplicates(root string) ([]DupGroup, error)
func (a *App) ScanStale(root string, olderThanDays int) ([]FileItem, error)
func (a *App) ScanArtifacts(root string) ([]ProjectItem, error)
// Implementasi M5 menggabungkan empat fungsi di atas menjadi:
// AnalysisTree(root, depth), AnalysisOverview(root), AnalysisDuplicates(root),
// AnalysisRefresh(), AnalysisCancel(), AnalysisClean(req), DiskStatus().
func (a *App) ScanAndroid() (AndroidReport, error)         // AVD + system images + NDK, read-only
func (a *App) AndroidAction(req AndroidRequest) (CleanResult, error) // hapus/wipe AVD, uninstall image
func (a *App) OpenInFileManager(path string) error
func (a *App) Settings()/SaveSettings(...)
```

Event runtime: `scan:progress` (pass, path, done, total), `clean:progress`, `nix:progress`.
Setiap fungsi berat menerima `context.Context` lewat `a.ctx` supaya UI bisa Cancel.

Binding juga dipakai headless (untuk test/CI): `cleanlie --dry-run --json` mencetak `JunkReport`
tanpa membuka window. Ini yang dipakai di integration test.

## 9. Milestone

| # | Isi | Output | Est. |
|---|---|---|---|
| **M0** | Scaffold: `flake.nix` (devShell go+webkitgtk+gtk3), `wails init`-manual, window kosong, tema gelap, `desktop` entry | `nix develop` → `wails dev` jalan | 0.5 h |
| **M1** | `internal/fsx` (port scanner CleanBro: allocated size, hardlink dedupe, one-fs, worker pool, progress) + `internal/rules` katalog Linux/XDG **+ `devtools.go` (npm, pnpm, gradle, cargo, go, uv, pip, dart, composer, playwright, cypress, nix cache)** + `internal/tools` (deteksi tool + command preview) + `Plan()` dry-run + test | satu kategori per tool (ukuran cocok dengan `du`), tool tak terpasang tersembunyi | 1 d |
| **M2** | `internal/clean` (gio trash + fallback, guard, protected, process check, verify freed, history) + layar **Bersihkan** (grup per tool, ⚠/⛔ badge) + dialog konfirmasi + eksekusi via perintah resmi | `npm cache clean`, `pnpm store prune`, `gradle caches`, `uv cache prune` jalan & ukuran turun; restore dari trash OK | 1.5 d |
| **M3** | `internal/nix` (parse generasi, closure size, dry-run GC) + layar **Nix** + `privileged` (pkexec) untuk GC/hapus generasi system/vacuum journal | GC dry-run = angka nyata; hapus generasi jalan | 1 d |
| **M4** | `internal/android` (parse AVD/ini, system-images, NDK, mtime pemakaian) + layar **Android** + aksi hapus/wipe AVD & uninstall image | hapus `test_devices` membebaskan ~5 G, `avdmanager list avd` konsisten | 0.5 d |
| **M5** ✅ | `internal/analysis` (tree, large, dup, stale) + `artifacts.go` + 4 layar analisis + dashboard kartu & top-reclaim | semua layar terisi data nyata | 1 d |
| **M6** ✅ | Pengaturan + persistensi (default tanpa centang untuk item ⚠/⛔), laporan hasil, i18n ID/EN, tema, packaging flake (`package.cleanlie`), ikon, README | `nix build .#cleanlie` → binary + desktop item | 0.5 d |

Total ~6 hari kerja. Urutan wajib M1 → M2 sebelum M3/M4/M5 (engine & planner dipakai bersama).

## 10. Packaging Nix

- `flake.nix` menyediakan: `devShells.default`, `packages.cleanlie` (buildGoModule + `makeWrapper`
  untuk `GDK_BACKEND`/`WEBKIT_DISABLE_COMPOSITING_MODE`), `apps.default`.
- Runtime deps: `gtk3`, `webkitgtk_4_1`, `nix` (sudah ada), `glib` (`gio trash`), `polkit` (`pkexec`),
  `systemd` (`journalctl`), `xdg-utils`. Wrapper wajib menambahkan path absolut tool-tool ini ke PATH,
  karena proses `pkexec` kehilangan environment sesi.
- **Polkit**: Cleanlie tidak memaksa user memasang agent. README mencantumkan opsi
  `security.polkit.enable = true` + `hyprpolkitagent`/`polkit-gnome`; tanpa agent semua aksi root
  otomatis jatuh ke mode perintah-siap-copy.
- Opsi NixOS (dokumentasi di README, tidak dipaksa): tambahkan `cleanlie` ke `environment.systemPackages`.
- Catatan: jangan pernah menaruh `cleanlie` sebagai auto-GC; `nix.gc.automatic` tetap urusan konfigurasi.

## 11. Testing & verifikasi

**Unit (tanpa disk nyata / pakai `t.TempDir()`):**
- `fsx`: ukuran = `du -s --block-size=1` pada fixture (sparse file, hardlink ganda, symlink tidak diikuti).
- `rules`: setiap root di katalog punya `Risk` dan tidak mengandung path protected; `Plan()` idempoten.
- `clean`: guard menolak `/`, `~`, `~/.config`; trash gagal di fs lain → lapor, bukan hapus diam-diam.
- `nix`: parser `--list-generations` & output `--dry-run` (fixture string asli).

**Integration (manual + script):**
1. Fixture: buat `~/.cache/cleanlie-test/{a,b}` 100 MB, scan → 100 MB, trash → hilang & ada di `~/.local/share/Trash/files`.
2. Bandingkan `ck sum` scanner vs `du -sx` di `~/.cache` **dan** di tiap katalog dev (`~/.npm/_cacache`, `~/.gradle/caches`, `~/.cargo/registry`, `~/.pub-cache`) — toleransi < 1%.
3. Jalankan tiap aksi dev tool di sandbox (`HOME=$(mktemp -d)` + copy fixture): pastikan `npm cache clean`/
   `pnpm store prune`/`uv cache prune` terpanggil, dan `Plan()` tidak berubah sebelum eksekusi.
4. `avdmanager list avd` vs hasil parser Cleanlie (nama, path, API) → harus sama; uji `Wipe` pada AVD fixture.
5. `nix-collect-garbage --dry-run` → angka aplikasi harus sama persis dengan output CLI.
6. Hapus generasi home-manager lama → `nix-env -p ... --list-generations` berkurang, sistem masih boot-able
   (uji hanya generasi **lama**, bukan current).
7. UI: cek 0 console error (WebKit inspector), resize 1280×800 & 1920×1080, Wayland Hyprland.
8. Tidak ada proses `rm -rf` untuk tool yang punya perintah resmi; audit `history.jsonl` cocok dengan item terpilih.

## 12. Risiko & mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| WebKitGTK render jelek di Hyprland/Wayland | UI rusak | env `WEBKIT_DISABLE_COMPOSITING_MODE=1`; fallback Fyne (engine sama) |
| Salah hapus cache yang sedang dipakai (Brave/VS Code/Gradle daemon) | app crash / build rusak | cek `/proc`, `gradle --stop` + skip + alasan |
| Hapus AVD yang masih dipakai / salah system image | emulator hilang, harus recreate | default tidak dicentang, cek referensi AVD → system image, saran hanya mtime > 90 hari |
| `rm -rf` langsung ke store tool (pnpm/gradle) | metadata tool rusak | selalu pakai perintah resmi bila ada; `rm` hanya fallback |
| Trash tidak bisa untuk `/nix`, `/var` | gagal hapus | dialog konfirmasi permanen eksplisit |
| GC menghapus generasi yang masih dibutuhkan | tidak bisa rollback | selalu dry-run + ketik `GC`; jangan sertakan generasi current |
| `du`/scan lambat di 82G store | UI beku | worker pool + progress event + cancel; store pakai `du -s` (bukan walk Go) atau cache hasil |
| pkexec tanpa agent polkit di sesi Hyprland | aksi root gagal | deteksi & tampilkan instruksi; sediakan `sudo` fallback di terminal |
| Paket manager tidak ada (cargo/flatpak) | error | deteksi `exec.LookPath`, sembunyikan aksi |

## 13. Non-goals v1

- Uninstaller aplikasi (Linux tidak seragam; nanti: flatpak/appimage + sisa `~/.config`).
- Auto-clean terjadwal / daemon / tray.
- Hapus item di `/nix/store` secara manual (hanya lewat GC).
- Pemindaian filesystem eksternal (NTFS/exFAT) & restore dari trash.
- Windows/macOS build.

## 14. Keputusan (sudah dikunci 2026-11-22)

| # | Keputusan | Alasan singkat |
|---|---|---|
| 1 | **Wails v2** (Go + HTML/CSS/JS vanilla), Fyne hanya fallback darurat | tabel/treemap/dialog butuh UI bebas; tanpa Node; satu binary |
| 2 | Module `github.com/Juragan-Koding/cleanlie`, binary `cleanlie` | siap publish, tidak perlu rename import nanti |
| 3 | **Hybrid pkexec → fallback perintah `sudo` siap-copy** | Hyprland belum tentu punya polkit agent; tidak boleh gagal senyap |
| 4 | UI: GUI window, Go backend, dry-run → trash | sudah disepakati sebelumnya |
| 5 | Scope v1: Nix, cache dev per-tool, Android, log/journal, analisis disk | sesuai temuan ukur di mesin ini |

Tidak ada pertanyaan terbuka. Langkah berikutnya: task breakdown (`docs/tasks/cleanlie-v1.md`) lalu **M0**.
