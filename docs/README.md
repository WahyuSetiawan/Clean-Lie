# Cleanlie — Dokumentasi

Cleaner GUI untuk NixOS (Go + Wails v2). Status: **M6 selesai** — engine scan, dry-run CLI, layar Bersihkan, eksekusi trash/permanen,
audit log, layar Nix (generasi, dead paths, GC, vacuum journal, fallback pkexec/sudo),
layar Android (AVD/SDK, saran >90 hari, hapus/wipe/uninstall), layar **Analisis**
(tree, file besar, duplikat, file lama, artefak) dengan **Dashboard** nyata, **Pengaturan**
persisten, tema system/light/dark, i18n ID/EN, serta desktop entry + ikon sudah jalan.

| Dokumen | Isi |
|---|---|
| [`../README.md`](../README.md) | Cara build/jalan (`nix develop`, `make dev`, `nix build`) |
| [`plan/cleanlie-v1.md`](plan/cleanlie-v1.md) | Plan lengkap: keputusan, arsitektur, katalog aturan (Nix, cache dev, Android, log, analisis), keamanan, layar, milestone, risiko |
| [`tasks/cleanlie-v1.md`](tasks/cleanlie-v1.md) | Task breakdown M0–M6 (37 task) + acceptance criteria + definisi selesai |
| [`wireframe/index.html`](wireframe/index.html) | **Wireframe interaktif** — acuan visual UI v1 (data ukur nyata, semua aksi simulasi) |
| [`wireframe/README.md`](wireframe/README.md) | Cara buka wireframe, pemetaan layar → task, aturan visual |
| [`implementation/m1-engine.md`](implementation/m1-engine.md) | Catatan implementasi engine: walker, katalog rule, CLI dry-run |
| [`implementation/m2-clean.md`](implementation/m2-clean.md) | Guard, trash freedesktop, deteksi proses, eksekutor, audit, wiring UI |
| [`implementation/m3-nix.md`](implementation/m3-nix.md) | Parser generasi, dead paths, privileged pkexec/sudo, layar Nix, gate `GC` |
| [`implementation/m4-android.md`](implementation/m4-android.md) | Parser AVD/SDK, aksi avdmanager/sdkmanager/wipe, guard, layar Android |
| [`implementation/m5-analysis.md`](implementation/m5-analysis.md) | Tree disk usage, overview file besar/lama/artefak, duplikat, binding/guard, dashboard |
| [`implementation/m6-settings.md`](implementation/m6-settings.md) | Config persisten, tema, i18n, packaging desktop/ikon, verifikasi akhir |
| [`../CHANGELOG.md`](../CHANGELOG.md) | Riwayat perubahan per milestone (M0–M6) |

## Buka wireframe

```sh
xdg-open docs/wireframe/index.html
```

## Urutan kerja

M0 scaffold → M1 engine (`fsx` + `rules` + dry-run) → M2 hapus + layar Bersihkan →
M3 Nix (generasi, GC, aksi root) → M4 Android (AVD/SDK) → M5 analisis + dashboard →
M6 settings, i18n, packaging & verifikasi.

## Fakta mesin target (2026-11-22)

`/nix/store` 82 G · `~/.cache` 23 G · cache dev 24.8 G · Android 28.2 G · journal 4 G ·
`/home` sisa 8.5 G dari 192 G.
