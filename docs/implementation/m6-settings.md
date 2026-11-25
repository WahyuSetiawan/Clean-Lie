# M6 — Settings, tema, i18n, dan packaging

Status: **selesai** · versi **0.6.0** · branch `feat/m6-settings` · 2026-11-25

M6 menutup v1: preferensi persisten, tema system/light/dark, i18n ID/EN untuk shell
dan layar utama, desktop entry + ikon, serta verifikasi build/paket.

## File

| File | Isi |
|---|---|
| `internal/config/config.go` | `Config`, `Default`, `Path`, `Normalize`, `Load`, `Save`; normalisasi path dan fallback default |
| `internal/config/config_test.go` | round-trip, file rusak, clamp nilai, dedupe path |
| `app_settings.go` | binding `Settings`, `SaveSettings`, `ResetSettings`; `excludedPath` |
| `app_settings_test.go` | persistensi lewat `NewApp`, reset, guard path |
| `app.go` | field config + muat config saat `NewApp` |
| `app_clean.go` | `PlanOptions.Excluded` + guard kedua saat `Clean` |
| `app_analysis.go` | ambang, exclusion, dan root tambahan diteruskan ke `analysis.Options` |
| `internal/rules/rules.go` | `PlanOptions.Excluded` + `underAny` |
| `internal/analysis/*` | `Options.Excluded` dan `Options.ExtraRoots` di tree/overview/duplikat |
| `frontend/dist/index.html` | layar Pengaturan nyata, blok M6, CSS fokus/reduced motion |
| `frontend/test/ui-smoke.mjs` | mock Settings/Save/Reset + assertion tema, bahasa, simpan |
| `packaging/cleanlie.desktop` | desktop entry (`StartupWMClass=cleanlie`) |
| `packaging/cleanlie.svg` | ikon sapu + disk |
| `flake.nix` | versi 0.6.0 + `postInstall` desktop/ikon |

## Config

Lokasi mengikuti XDG: `$XDG_CONFIG_HOME/cleanlie/config.json`, fallback
`~/.config/cleanlie/config.json`.

```json
{
  "theme": "system",
  "language": "id",
  "showRisky": true,
  "deleteMode": "trash",
  "autoSelectSafe": true,
  "journalTarget": "500M",
  "largeBytes": 104857600,
  "staleDays": 365,
  "artifactRoots": [],
  "excluded": []
}
```

- File hilang → default tanpa warning.
- File rusak / nilai tidak dikenal → default + warning yang tampil di layar Pengaturan.
- `Normalize` membatasi ambang (≥1 MB, ≤100 GB; stale 1–3650 hari), menormalkan `~`,
  membuang path relatif/duplikat, dan mengurutkan path.
- `Save` menulis file sementara lalu `rename` (atomik), permission `0644`.
- `SaveSettings` membersihkan cache analisis karena ambang/exclusion/root berubah.

## Alur pemakaian setting

1. App dibuat → `config.Load()`.
2. UI membuka Pengaturan → `Settings()` mengembalikan config + path + warning.
3. User mengubah form → tombol **Simpan** memanggil `SaveSettings`.
4. `SaveSettings` menulis config, memperbarui state Go, lalu `AnalysisRefresh()`.
5. Scan katalog memakai `Excluded`; tree/overview/duplikat memakai `Excluded`,
   `LargeBytes`, `StaleDays`, dan `ExtraRoots`.
6. `Clean` memeriksa ulang `excludedPath` sebelum eksekusi. Item lama yang masih ada di
   cache scan tidak bisa menghapus path yang baru dikecualikan.

## Tema dan aksesibilitas

- Mode `system` memakai `prefers-color-scheme`; perubahan sistem langsung diterapkan.
- Tombol tema di topbar tetap ada untuk ganti cepat; perubahan ditandai belum disimpan.
- `:focus-visible` outline, `aria-checked` pada toggle, `role="switch"`, `aria-label`
  pada input penting, dan `prefers-reduced-motion` untuk mematikan animasi.
- Warna mengikuti token CSS yang sudah ada; teks utama memakai `--text` di kedua tema.

## i18n

- Tabel string berada di blok M6 `frontend/dist/index.html` (bukan file `src/i18n.js`
  terpisah, karena frontend sengaja satu file tanpa build Node).
- `t(key)` + atribut `data-i18n`, `data-i18n-placeholder`, `data-i18n-title`.
- Cakupan: sidebar, judul/subjudul halaman, Pengaturan, Dashboard, tab/header Analisis.
- Ganti bahasa langsung tanpa reload; pilihan disimpan lewat tombol Simpan.
- **Batas:** beberapa label dinamis dari layar M2–M4 (mis. teks status hasil scan,
  saran Android) masih berbahasa Indonesia. Ini dicatat sebagai pekerjaan bertahap,
  bukan diklaim sudah 100% i18n.

## Packaging

`nix build .#cleanlie` sekarang menghasilkan:

```
result/bin/cleanlie
result/share/applications/cleanlie.desktop
result/share/icons/hicolor/scalable/apps/cleanlie.svg
```

Desktop entry memakai `Exec=cleanlie`, `Icon=cleanlie`, `Categories=System;Utility;`,
dan `StartupWMClass=cleanlie`. Ikon SVG memakai palet accent Cleanlie.

## Verifikasi (2026-11-25)

| Cek | Hasil |
|---|---|
| `gofmt -l .` / `go vet` | bersih |
| `go test` | **10 paket hijau** (main, analysis, android, clean, config, fsx, nix, privileged, rules, tools) |
| `make build` | `cleanlie 0.6.0` |
| `nix build .#cleanlie` | sukses; versi 0.6.0 |
| `nix build .#cleanlie --rebuild` | output konsisten |
| Desktop entry | `desktop-file-validate` hanya memberi hint kategori ganda |
| UI smoke jsdom | scan → gate → trash → Nix/GC → Android → dashboard → analisis → **settings/tema/i18n** hijau |
| Config round-trip | `TestSettingsSaveAndReload` + `TestLoadMissingUsesDefaults` hijau |
| Exclusion | rule, tree, overview, dan guard `Clean` punya test |
| `--dry-run --json` | 151 item / 9 grup / 94.953.920.921 B / 1.438.991 file |
| `--nix-dead` | 716 path — sama dengan `nix-collect-garbage --dry-run` (716) |
| `--android-status` | 1 AVD / SDK 13,81 GiB / 13 paket / 1 saran |
| `--analysis-tree --depth 1` | 179.182.145.536 B / 3.387.028 file / 4,86 s |
| `nix run .#cleanlie` | window hidup 10 s tanpa crash (timeout terkontrol; MESA warning diketahui) |

## Deviasi dari task

1. i18n tidak memakai `frontend/src/i18n.js`; string table tetap inline karena UI satu file
   dan tidak ada build Node.
2. Cakupan i18n baru shell/settings/dashboard/analysis; label dinamis M2–M4 menyusul.
3. Screenshot 6 layar tidak dibuat ulang (headless browser di mesin ini tidak stabil);
   penggantinya smoke test jsdom yang menjalankan alur nyata dengan binding tiruan.
4. `ShowRisky` default `true` agar perilaku lama (semua item terlihat) tidak berubah;
   item ⚠/⛔ tetap tidak pernah otomatis dicentang.
5. Desktop entry memakai `StartupWMClass=cleanlie`; asosiasi ikon bergantung WM class
   yang dilaporkan WebKitGTK di compositor.

## Gotcha

- `os.UserConfigDir()` mengikuti `XDG_CONFIG_HOME`; test wajib `t.Setenv` sebelum `NewApp`.
- Config tidak boleh menyimpan path relatif; `Normalize` membuangnya agar guard prefix aman.
- Jangan panggil `AnalysisRefresh` saat masih memegang `cfgMu`; `SaveSettings` melepas lock
  lebih dulu.
- `data-i18n` mengganti `textContent`; jangan pasang pada elemen yang punya child markup
  penting (mis. ikon SVG).
- `window.__getSettings` dipakai kode M2 untuk auto-select/visibility; jangan hapus saat
  merapikan blok M6.
- `packaging/` harus ikut `git add` sebelum `nix build`, karena flake hanya melihat file
  yang di-track.
