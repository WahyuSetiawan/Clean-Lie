# Cleanlie M11 — Polish

> Addendum `cleanlie-v1.md`. Status: **selesai** (0.11.0).
> Prasyarat: M10 selesai (v0.10.0). Target rilis: **0.11.0**.

## 1. Latar belakang

Fitur utama sudah lengkap, tetapi ada empat hal yang membuat pemakaian
sehari-hari belum rapi:

- Dua instance GUI bisa dibuka bersamaan dan saling menimpa state/aksi.
- Label dinamis (status, ringkasan, hint) masih hardcoded bahasa Indonesia
  sehingga mode English belum konsisten.
- CI hanya build; tidak ada test/lint yang menjaga regresi.
- Kartu disk menampilkan persentase, tetapi tidak memberi sinyal “sehat /
  tipis / kritis” yang bisa dibaca cepat.

## 2. Tujuan

1. **Single instance**: hanya satu GUI Cleanlie; instance kedua keluar dengan
   pesan jelas tanpa merusak state.
2. **i18n label dinamis**: status/hint/ringkasan yang dibuat JS mengikuti
   bahasa aktif (ID/EN), bukan hanya label statis `data-i18n`.
3. **CI**: workflow test + vet + gofmt + UI smoke + `nix build`.
4. **Disk health**: klasifikasi kapasitas (`ok`/`warn`/`critical`) + chip di
   dashboard.

## 3. Non-tujuan

- SMART/`smartctl` atau kesehatan hardware; M11 hanya kapasitas filesystem.
- Menerjemahkan label katalog aturan dari Go (nama teknis seperti `_cacache`
  tetap); fokus pada teks UI dinamis.
- Auto-focus instance lama lewat D-Bus.
- Perubahan format audit/trend/report.

## 4. Single instance

- `internal/singleinstance`: lock `flock` non-blocking di
  `<state>/cleanlie/cleanlie.lock` (0600), simpan PID.
- `Acquire()` → `*Lock`; `ErrRunning` bila sudah dipegang proses lain.
- GUI memanggil sebelum `wails.Run`; CLI headless **tidak** memakai lock agar
  `--insight`, `--report`, dan `--trash-*` tetap bisa dipakai saat GUI terbuka.
- Lock dilepas saat proses keluar (`defer`); file lock boleh tetap ada.
- Instance kedua: pesan ke stderr, exit 1.

## 5. Disk health

- `fsx.ClassifyHealth(total, free)`:
  - `unknown` bila total ≤ 0;
  - `critical` bila free < 5% **atau** < 2 GiB;
  - `warn` bila free < 12% **atau** < 10 GiB;
  - `ok` selain itu.
- `fsx.Volume` mendapat field `health`; `Statfs` mengisinya.
- Dashboard: chip `Sehat / Tipis / Kritis / Tidak diketahui`, hint
  `N% terpakai · sisa X`, warna chip mengikuti level.
- Tes klasifikasi batas + UI smoke chip.

## 6. i18n label dinamis

- Helper `window.__tf(key, vars)` (fallback ke ID) di `core.js`; semua modul
  memakai `window.__t`/`__tf` saat runtime.
- Tambah key untuk:
  - dashboard: disk health, hint, nix/journal, cache/log, sparkline kosong,
    rentang tren, item/trash, ekspor;
  - analisis: ringkasan tree/duplikat/artefak, loading/empty state;
  - Riwayat/preview: status item, jumlah dipilih, kosong;
  - Nix/Android/direnv: status dinamis utama.
- `applyLanguage()` memanggil refresh ringan untuk layar aktif agar teks yang
  sudah dirender ikut berganti bahasa.
- Label teknis (nama rule, path, unit) tidak diterjemahkan.

## 7. CI

`.github/workflows/test.yml`:

- **go**: `nix develop --command go test -tags desktop,production,webkit2_41 ./...`,
  `go vet`, cek `gofmt -l` kosong.
- **ui**: Node 20, `npm ci && npm test` di `frontend/test`.
- **build**: `nix build .#cleanlie --print-build-logs` (butuh go + ui).
- Trigger `push`, `pull_request`, `workflow_dispatch`.

## 8. Acceptance

- Dua `Acquire` berurutan: kedua `ErrRunning`; setelah `Release`, acquire sukses.
- `ClassifyHealth` tabel batas hijau; `DiskStatus().Health` terisi.
- UI smoke: chip disk punya teks; ganti bahasa → hint/status dinamis berubah.
- Workflow YAML valid secara sintaks dan perintahnya sama dengan yang dijalankan
  lokal.
- `go test`, `go vet`, `gofmt`, UI smoke, `nix build` 0.11.0 hijau.
