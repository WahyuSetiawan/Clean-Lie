# Screenshot harness

Menghasilkan gambar di [`docs/screenshots/`](../) secara deterministik dari UI
statis di `frontend/dist/`, tanpa menjalankan backend Go.

## Kenapa ada

UI Cleanlie butuh binding `window.go.main.App` untuk menampilkan data nyata.
Harness ini menggantinya dengan `fixtures.js` berisi data contoh, lalu menangkap
tiap layar lewat Chrome DevTools Protocol. Hasilnya:

- tidak memerlukan Linux/GTK/WebKit atau build Wails;
- tidak menampilkan path, nama proyek, atau ukuran dari mesin siapa pun;
- bisa di-regenerate kapan saja setelah UI berubah.

## Butuh

- Node.js 18+ (memakai `fetch` dan `WebSocket` bawaan)
- Chromium/Chrome/Brave — dicari di `/Applications`, lalu `PATH`, atau set
  `CHROMIUM=/path/to/browser`
- ImageMagick (`magick`) untuk menurunkan resolusi 2x → 1x

## Jalankan

```sh
node docs/screenshots/harness/shoot.mjs
```

Hasilnya menimpa `docs/screenshots/{dashboard,cleanup,nix,android,analysis,direnv,settings}.png`
(1440×900).

## Cara kerja

1. `frontend/dist/` disalin ke direktori sementara.
2. `fixtures.js` disuntikkan **sebelum** `ui/demo.js`, sehingga `window.go` sudah
   ada saat `ui/core.js` mengambil referensi bridge.
3. Folder disajikan lewat HTTP lokal, lalu browser headless membuka tiap layar
   (`#/dashboard`, `#/cleanup`, …).
4. Screenshot diambil pada device scale factor 2, lalu diperkecil 50% agar teks
   tetap tajam dan file kecil.

## Mengubah data contoh

Semua angka ada di `fixtures.js` (`report`, `nix`, `android`, `direnv`, `tree`,
`overview`, `dups`, `settings`, `diagnostics`). Path memakai `HOME = /home/wahyu`
supaya UI menampilkan `~/...`.

Kalau menambah method binding baru di `app*.go` dan dipakai UI, tambahkan juga
stub-nya di objek `api` pada `fixtures.js` — semua method dibungkus `Promise`
seperti Wails asli.
