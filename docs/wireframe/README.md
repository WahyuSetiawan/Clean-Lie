# Wireframe UI — Cleanlie v1

Referensi visual untuk implementasi (M0–M6). **Bukan aplikasi**: semua tombol hanya simulasi,
file ini tidak membaca atau menghapus apa pun dari mesin. Implementasi nyata ada di
[`../../frontend/dist/index.html`](../../frontend/dist/index.html) (shell M0, sudah tersambung ke binding Go).

## Cara buka

```sh
# paling gampang
xdg-open docs/wireframe/index.html

# atau lewat server lokal (kalau mau inspect/devtools rapi)
python3 -m http.server 8080 -d docs/wireframe
# → http://localhost:8080
```

Data di wireframe memakai **angka hasil ukur nyata** di mesin ini (22 Nov 2026):
`/nix/store` 82.0 G, `~/.cache` 23 G, cache dev 24.8 G, Android 28.2 G, AVD 14.2 G, `_cacache` 7.0 G,
journal 4.0 G.

## Layar → task implementasi

| Layar | Isi wireframe | Task |
|---|---|---|
| **Dashboard** | 4 kartu (disk, nix, cache dev, journal), map `/home`, top-5 reclaim, status polkit, tombol bersihkan cepat | T5.7 |
| **Bersihkan** | Kategori bergrup (per tool + sistem), tabel item dengan checkbox/risiko/catatan, select-all-safe, action bar total terpilih | T2.5, T2.6, T2.7, T2.8 |
| **Nix** | Kartu store/dead paths/generasi, tabel generasi + closure, tombol GC & hapus generasi, panel journal | T3.1–T3.5 |
| **Android** | Kartu AVD/SDK/emulator, tabel AVD + saran `>90 hari`, system images, NDK/build-tools | T4.1–T4.4 |
| **Analisis** | Tab Disk usage (bar + peta), File besar, Duplikat, File lama, Artefak build | T5.1–T5.6 |
| **Pengaturan** | Tema/bahasa, mode hapus, target journal, ambang analisis, root artefak, pengecualian, status polkit, riwayat | T6.1–T6.3 |

Angka pada layar Nix di wireframe adalah contoh desain. Aplikasi mengisi kartu dan tabel
dengan data nyata (`--nix-status` / `--nix-dead`); nilai di wireframe tidak dipakai sebagai
klaim ukuran.

## Yang bisa dicoba di wireframe

- Klik sidebar → pindah layar (routing `#/dashboard`, `#/cleanup`, …).
- **Bersihkan**: ganti kategori kiri, centang item, lihat total di action bar berubah.
  Item ⛔ terkunci (disabled), item ⚠ tetap bisa dicentang tapi tidak otomatis.
- **Pilih semua yang aman** → hanya item `risk=safe` yang tercentang.
- **GC** → modal dry-run dengan angka, tombol baru aktif setelah mengetik `GC`.
- **Android** → tombol saran mencentang AVD yang > 90 hari.
- **Analisis** → pindah tab.
- Tombol tema (kanan atas) → cek dark/light.

## Aturan visual yang harus diikuti implementasi

1. **Warna risiko konsisten**: aman = hijau (`--accent`), hati-hati = amber (`--warn`),
   berisiko = merah (`--danger`), terkunci = abu (`--dim`).
2. **Ukuran selalu tabular** (`font-variant-numeric: tabular-nums`) supaya kolom tidak goyang saat scan.
3. **Path selalu monospace**, home dipendekkan jadi `~`.
4. **Action bar hanya muncul di layar Bersihkan**, tidak menutupi konten (padding bawah 110px).
5. **Konfirmasi destruktif**: modal gelap + ringkasan (jumlah item, ukuran, mode) + command preview.
   GC wajib ketik `GC`; hapus permanen wajib checkbox sadar-risiko.
6. **Badge "wireframe"** di top bar hanya ada di mockup, tidak di aplikasi jadi.
7. Layout minimum 1280×800; sidebar mengecil jadi rail ikon di bawah 1080 px (sudah dicontohkan).

## Belum ada di wireframe (sengaja)

- Empty state / skeleton loading / progress scan — desainnya menyusul di T2.6.
- Dialog error pkexec & fallback perintah `sudo` siap-copy — T3.4.
- Laporan hasil setelah hapus (freed aktual, item gagal) — T2.8.
