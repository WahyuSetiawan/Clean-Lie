# M1 — Engine scan & katalog rule (selesai 2026-11-25)

Status: ✅ · PR: [#2](https://github.com/WahyuSetiawan/Clean-Lie/pull/2)

## Hasil

```
$ cleanlie --dry-run
Cleanlie — dry run (tidak ada yang dihapus)
Total: 88.4 GiB · 1.435.650 file · 3,5 s

[dev.node]   Node (npm/pnpm)      16.2 GiB
[dev.jvm]    JVM (Gradle)         13.1 GiB
[dev.python] Python (uv/pip)       2.0 GiB
[dev.dart]   Dart / Flutter        1.5 GiB
[dev.rustgo] Rust / Go             0.9 GiB
[dev.android] Android SDK/AVD     18.8 GiB
[app]        Cache aplikasi       15.4 GiB
[sys.log]    Log & journal        20.7 GiB
[trash]      Trash                 2.2 MiB
```

Temuan terbesar yang tidak ada di ukuran awal:

| Item | Ukuran | Catatan |
|---|---|---|
| `/var/tmp/avd` | 17 GiB | salinan AVD di `/var/tmp`, bukan `~/.android/avd` |
| `~/.npm/_cacache` | 7.2 GiB | |
| `~/.android/avd/test_devices.avd` | 5.1 GiB | AVD lama |
| `~/.gradle/caches/9.5.0` | 4.8 GiB | |

## Yang dibuat

```
internal/fsx/walk.go        walker paralel: Lstat, st_blocks*512, hardlink dedupe, one-fs,
                            context cancel, progress, top-N heap
internal/fsx/usage.go       Usage, List, Human, Statfs, Exists, IsDir
internal/rules/rules.go     Rule/Item/Group/Report, Plan (read-only), dedupe prioritas,
                            dropAncestors, enumerasi depth, order grup
internal/rules/linux.go     cache aplikasi, journal (probe), log rotasi, /var/tmp, trash
internal/rules/devtools.go  npm/pnpm/gradle/maven/uv/pip/dart/cargo/go/gopls/playwright/cypress
internal/rules/android.go   AVD (+parser .ini), system-images, ndk, platforms, build-tools, emulator
internal/rules/nix.go       konstanta store (eksekusi GC di M3)
internal/rules/protected.go daftar path terlarang (pertahanan terakhir M2)
internal/tools/tools.go     Available, Preview, Run(ctx, cmd, timeout) + stdout/stderr
cli.go                      --dry-run, --json, --scan-only --path, --group, --workers, --version
scripts/verify-sizes.sh     bandingkan item > 100 MiB vs du -sk
```

## Keputusan

1. **Ukuran = blok terpakai**, bukan ukuran logis. `st_blocks * 512` membuat hasil sama
   dengan `du` (termasuk blok direktori). File sparse tidak dilebih-lebihkan.
2. **Hardlink dihitung sekali** per `(Dev, Ino)` dalam satu walk.
3. **`dropAncestors`**: kalau dua rule menghasilkan path yang saling bersarang, item induk
   dibuang supaya tidak dobel. Contoh: `~/.cache/ms-playwright` (generik) vs
   `chromium-1248` (rule Playwright).
4. **Prioritas rule** menyelesaikan path duplikat: rule dev eksplisit (20) mengalahkan
   generik `~/.cache/*` (-10).
5. **Journal bukan `du`**: `journalctl --disk-usage` (fallback walk). Item journal
   `fallbackDelete=false` karena hanya boleh lewat vacuum.
6. **Perintah resmi** hanya ditampilkan kalau binary-nya ada di PATH; kalau tidak,
   catatan item berubah menjadi "tool tidak ada di PATH, hapus folder manual".
7. **CLI read-only**: `internal/rules` tidak mengimpor `internal/clean` (belum ada),
   dan tidak ada `os.Remove*` di paket ini.

## Verifikasi

```sh
nix develop -c make build
./bin/cleanlie --dry-run --json | jq .        # JSON valid
./scripts/verify-sizes.sh ./bin/cleanlie      # 15 item, toleransi 2%
nix develop -c go test -tags desktop,production,webkit2_41 ./...
```

Hasil `verify-sizes.sh`: 14 item delta **0,00%**, journal **0,99%** (pembulatan output
`journalctl`), semuanya `ok`.

## Gotcha

- `journalctl --disk-usage` mencetak `4.0G`, bukan `4.0GiB`; parser harus menerima
  suffix tanpa `iB` dan koma desimal.
- Test `HOME` harus di-`t.Setenv` **sebelum** memanggil `DefaultRules`, karena path rule
  dihitung saat fungsi dipanggil.
- `/var/log` adalah root rule (enumerasi anak), jadi tidak boleh dianggap target hapus.
  `IsProtected` dipakai untuk target, bukan untuk root rule.
- Test paket `main` butuh build tag Wails (`webkit2_41`) karena `main.go` mengimpor Wails.
- `du` dan walker bisa beda beberapa KB untuk symlink pendek; test memakai toleransi 8 KiB.

## Sisa untuk M2

- Deteksi proses berjalan (`pgrep`) sebelum item browser/emulator dihapus.
- Eksekusi `tools.Run` + trash + ukur ulang `freed`.
- Audit log dan konfirmasi ketik `GC` untuk item Nix.
