# M4 — Android: AVD, SDK, dan aksi resmi

Status: **selesai 2026-11-25**. Gate M4 — hapus AVD benar-benar membebaskan ruang —
diverifikasi dengan fixture (folder + `.ini` hilang, `freed` terukur) dan inventaris nyata
mesin target. AVD asli `test_devices` tidak dihapus selama pengembangan.

## Yang dibangun

| File | Isi |
|---|---|
| `internal/android/android.go` | `ReadStatus`, parser AVD (`*.ini` + `config.ini`), parser paket SDK, deteksi emulator, saran >90 hari |
| `internal/android/actions.go` | `DeleteAVD`, `UninstallPackage`, `WipeAVD`, `Preview`, pencarian tool absolut |
| `app_android.go` | Binding Wails: status, refresh, hapus AVD, uninstall paket, wipe, preview; cache 2 menit |
| `frontend/dist/index.html` | Layar Android nyata: kartu, tabel AVD, system image, NDK/build-tools, pilih saran, gate konfirmasi, hasil, rescan |
| `cli.go` | `--android-status` (JSON) untuk verifikasi headless |

`internal/rules/android.go` yang sudah ada tetap menjadi bagian katalog dry-run/scan.
`internal/android` menangani detail AVD/SDK dan aksi; keduanya tidak saling mengimpor.

## Inventaris nyata (2026-11-25)

| Data | Nilai |
|---|---|
| AVD | `test_devices` — **5.381.066.752 byte** (5,01 GiB) |
| Terakhir dipakai | `2026-11-24 07:32` (dari `emu-launch-params.txt`), usia 1 hari → **tidak** disarankan |
| AVD home | `/home/juragankoding/.android/avd` |
| SDK | `/home/juragankoding/Android/Sdk` |
| Total paket SDK | **14.833.647.616 byte** (13,81 GiB) |
| system-images | android-35 **3.754.844.160** (154 hari, tidak dipakai, **disarankan**); android-36 **4.585.545.728** (dipakai `test_devices`) |
| NDK | `27.0.12077973` 2.076.188.672; `28.2.13676358` 2.315.939.840 |
| build-tools | 28.0.3, 35.0.0, 36.0.0 (masing-masing ±0,13–0,15 GiB) |
| platforms | android-33/34/35/36/37.0 (±0,14–0,17 GiB) |
| emulator | 894.242.816 byte |
| Emulator berjalan | tidak |
| Durasi `--android-status` | **137 ms** |

Satu-satunya saran otomatis pada mesin ini adalah
`system-images;android-35;google_apis;x86_64` (3,50 GiB). AVD tidak disarankan karena
baru dipakai kemarin; ini disengaja agar saran tidak menghapus emulator yang masih aktif
dipakai.

## Sumber data (dan kenapa)

| Data | Implementasi | Alasan |
|---|---|---|
| Daftar AVD | baca `~/.android/avd/*.ini` + `config.ini` | `avdmanager list avd` mengembalikan **kosong** di mesin ini meski AVD ada; parser tidak boleh bergantung pada CLI itu |
| Nama & path AVD | nama file `.ini` dan field `path=` | mengikuti format resmi Android |
| API / ABI / tag | `target=`, `image.sysdir.1=`, `abi.type=` | `config.ini` tidak selalu punya semua field; parser toleran |
| Ukuran AVD | `fsx.Usage` pada folder `.avd` | sama dengan ukuran yang dilihat user; termasuk snapshot dan userdata |
| Terakhir dipakai | mtime terbaru dari `emu-launch-params.txt`, `snapshots/`, `userdata-qemu.img*`, `cache.img*` | `userdata` bisa sangat lama sementara emulator baru dijalankan; `emu-launch-params.txt` lebih jujur untuk "terakhir dipakai" |
| Paket SDK | direktori `system-images`, `ndk`, `build-tools`, `platforms`, `emulator` | paket v1; `cmdline-tools`, `cmake`, `platform-tools` tidak dihitung di kartu SDK |
| Dipakai AVD | `image.sysdir.1` dicocokkan ke ID system image + fallback `api`/`abi` | system image yang dipakai tidak boleh disarankan |
| Emulator berjalan | scan `/proc/*/cmdline` untuk `qemu-system`, `emulator -avd`, `-avd` | tidak menambah proses `pgrep`, dan tetap bekerja saat `avdmanager` bermasalah |
| Tool absolut | `cmdline-tools/latest/bin`, versi `cmdline-tools/*/bin`, `tools/bin`, `emulator/emulator` | PATH sesi GUI sering tidak memuat SDK |

Paket yang **tidak** dihitung ke kartu SDK: `cmdline-tools` (148 MB), `cmake` (61 MB),
`platform-tools` (22 MB), `licenses`. Ini menjaga scope T4.2 tetap sesuai plan; angka
kartu SDK karena itu lebih kecil dari `du -sh ~/Android/Sdk`.

## Aksi dan guard

| Aksi | Perintah | Guard |
|---|---|---|
| Hapus AVD | `<sdk>/cmdline-tools/latest/bin/avdmanager delete avd -n <nama>` | nama harus ada di inventory; emulator tidak berjalan; path di dalam `AVDHome`; konfirmasi checkbox |
| Uninstall paket | `<sdk>/cmdline-tools/latest/bin/sdkmanager --sdk_root=<sdk> --uninstall "<id>"` | paket harus ada; `UsedBy` kosong; emulator tidak berjalan; path di dalam SDK; konfirmasi checkbox |
| Wipe AVD | `<sdk>/emulator/emulator -avd <nama> -wipe-data -no-window -no-audio -no-snapshot -no-boot-anim -quit-after-boot 180` | AVD tidak berjalan; emulator tidak berjalan; konfirmasi checkbox; timeout 6 menit |

- **UI tidak pernah mengirim path.** Hapus AVD mengirim nama; uninstall mengirim ID paket;
  path diresolusi ulang dari inventory di Go.
- `avdmanager` dipakai lebih dulu. Fallback `os.RemoveAll` folder `.avd` + `.ini` hanya
  jika `avdmanager` tidak ada atau bilang AVD tidak dikenal/invalid. Fallback tetap
  divalidasi: folder harus berakhiran `.avd` dan berada di dalam `AVDHome`.
- Semua perintah memakai **path absolut**; `sdkmanager` juga menerima `--sdk_root` eksplisit.
- `freed` dihitung `sebelum − sesudah` dari ukuran folder/ini yang sebenarnya.
  `Deleted` hanya `true` bila folder/ini benar-benar hilang.
- Result membawa `exitCode`, `stdout`, `stderr`, `preview`, `durationMs`, `freed`, `deleted`.
- Jika emulator hidup, tombol hapus/wipe **disabled** dan handler tetap menolak sebagai
  lapisan kedua.
- `sdkmanager` yang sukses tetapi folder masih ada dilaporkan gagal dengan alasan jelas,
  bukan klaim berhasil.

## Layar Android

- Kartu: total AVD, total SDK (dengan rincian system-images + NDK), status emulator.
- Tabel AVD: checkbox, nama, API, ABI, ukuran, tanggal terakhir dipakai, status
  (`jalan`, `saran hapus`, `N hari`), tombol **Wipe** dan **Hapus** per baris.
- Tabel system image: checkbox, package ID, ukuran, chip AVD yang memakainya atau
  `saran hapus` / `tidak dipakai AVD`.
- Tabel NDK/build-tools/platform: checkbox, package ID, ukuran, `cek proyek` (v1 tidak
  menebak pemakaian Gradle/NDK).
- **Default tidak ada yang dicentang.** `Pilih yang disarankan` hanya mencentang AVD
  `>90 hari` dan system image yang tidak dipakai AVD; NDK/build-tools tidak pernah
  disarankan otomatis.
- Tombol **Hapus terpilih** dan **Wipe AVD terpilih** terkunci sampai ada pilihan;
  keduanya memakai gate checkbox eksplisit yang berbeda dari gate ketik `GC`.
- Modal menampilkan preview perintah per item, total ukuran, dan peringatan data hilang.
- Hasil menampilkan `freed`, status per item, error/stdout/stderr, dan **Muat ulang**
  memaksa scan baru (cache 2 menit).
- Tanpa AVD/paket, tabel menampilkan empty state, bukan error.

## Verifikasi

```sh
nix develop -c gofmt -l .
nix develop -c go vet -tags desktop,production,webkit2_41 ./...
nix develop -c go test -tags desktop,production,webkit2_41 ./...
nix develop -c make build
./bin/cleanlie --android-status | jq .
cd frontend/test && npm test
```

Hasil: **8 paket Go hijau** (main, android, clean, fsx, nix, privileged, rules, tools);
`gofmt` bersih; `go vet` bersih; UI smoke jsdom hijau untuk alur
saran → gate → hapus AVD + uninstall → hasil → wipe → gate → hasil.

Unit test Android memakai fixture 2 AVD + SDK palsu:
- `ReadStatus` → nama, API, ABI, system image, ukuran, usia, saran; system image yang
  dipakai bertanda `UsedBy` dan tidak disarankan; NDK tidak pernah disarankan.
- `DeleteAVDFallback` → `avdmanager` bilang "does not exist" → folder + `.ini` hilang,
  `Deleted=true`, `freed>0`.
- `DeleteAVDOfficial` → perintah absolut `avdmanager delete avd -n test`.
- `DeleteAVDRefusesRunning` → proses `emulator -avd test` di `/proc` palsu menolak aksi.
- `UninstallPackage` → `--uninstall` + `--sdk_root`, env `ANDROID_HOME`; paket yang
  dipakai AVD ditolak.
- `WipeAVDCommand` → `-wipe-data` + `-quit-after-boot`.
- `PreviewUsesAbsoluteTool` → preview memakai path absolut.

**Tidak** ada test yang menghapus AVD/SDK asli. Verifikasi gate di mesin nyata memakai
inventaris + fixture, bukan `rm` pada `test_devices`.

## Deviasi dari plan

1. **Tidak memakai `avdmanager list avd`** sebagai sumber utama. Di mesin ini perintah itu
   mengembalikan daftar kosong meski `test_devices.avd` ada. Parser file adalah sumber
   kebenaran; `avdmanager` hanya untuk aksi.
2. **`lastUsed` = mtime terbaru** dari beberapa kandidat file, bukan hanya
   `userdata-qemu.img`. `test_devices` punya `userdata-qemu.img.qcow2` dari Agustus 2026
   tetapi `emu-launch-params.txt` dari 24 Nov 2026; memakai yang terbaru mencegah saran
   hapus yang salah.
3. **Kartu SDK tidak menjumlahkan `cmdline-tools`/`cmake`/`platform-tools`**, sesuai
   scope T4.2. Total `du -sh ~/Android/Sdk` akan lebih besar.
4. **Saran system image memakai mtime folder paket ≥90 hari dan `UsedBy` kosong.**
   build-tools/NDK/platform tidak disarankan otomatis karena pemakaiannya tidak bisa
   dipastikan dari AVD saja.
5. **Wipe dijalankan headless** (`-no-window`, `-no-audio`, `-no-snapshot`,
   `-quit-after-boot 180`) agar aksi dari GUI tidak meninggalkan emulator menggantung.
6. **Gate hapus memakai checkbox**, bukan ketik `GC`. Gate `GC` khusus Nix; menghapus AVD
   tidak mengubah store Nix.
7. **Fallback manual hanya untuk AVD**, bukan uninstall paket. Uninstall wajib lewat
   `sdkmanager` karena metadata SDK harus ikut diperbarui.

## Gotcha

- `avdmanager list avd` kosong di mesin ini. Jangan jadikan sumber data.
- `sdkmanager`/`avdmanager` tidak ada di PATH; harus dicari di dalam SDK dan dipanggil
  dengan path absolut.
- `sdkmanager --uninstall` membutuhkan `--sdk_root=<sdk>` agar tidak menebak SDK lain.
- ID paket memakai titik koma (`system-images;android-36;...`); jangan dipecah shell,
  kirim sebagai satu argumen.
- `-quit-after-boot` menerima jumlah detik, bukan boolean.
- AVD yang sedang berjalan tidak boleh dihapus/di-wipe; cek `/proc` sebelum aksi dan
  disable tombol di UI.
- `userdata-qemu.img` bisa berupa `.qcow2`; parser harus menerima keduanya.
- Folder `system-images/android-35/google_apis/x86_64` punya mtime sendiri; usia paket
  dihitung dari folder terdalam, bukan hanya folder API.
- Jangan mengandalkan `ageDays` untuk AVD yang `Error`; item error tidak disarankan.

## Yang sengaja belum dikerjakan

- Deteksi paket SDK yang dipakai Gradle/NDK per proyek (baca `local.properties`,
  `build.gradle`, `ndkVersion`).
- Hapus `cmdline-tools`, `cmake`, `platform-tools`, `licenses`.
- Integrasi `adb` untuk mematikan emulator dari aplikasi.
- Saran otomatis untuk NDK/build-tools/platform.
- Undo: hapus AVD/SDK bersifat permanen (tidak masuk trash).
