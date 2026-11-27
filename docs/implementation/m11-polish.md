# M11 — Polish (implementasi)

Status: **selesai** · versi **0.11.0** · 2026-11-27

## Ringkas

M11 merapikan pemakaian harian: satu GUI saja, sinyal kesehatan kapasitas disk,
teks dinamis yang ikut bahasa ID/EN, dan CI yang menjaga test/lint/format/build.

## File yang berubah

| File | Isi |
|---|---|
| `internal/singleinstance/singleinstance.go` | lock `flock` non-blocking + PID |
| `internal/singleinstance/singleinstance_test.go` | acquire ganda, release lalu acquire |
| `main.go` | acquire setelah CLI, sebelum `wails.Run`; exit 1 bila sudah jalan |
| `internal/fsx/health.go` | `Health`, `ClassifyHealth` |
| `internal/fsx/health_test.go` | tabel batas ok/warn/critical/unknown |
| `internal/fsx/usage.go` | `Volume.Health` + `Statfs` mengisi |
| `frontend/dist/index.html` | chip `dash-disk-health` |
| `frontend/dist/ui/core.js` | helper `window.__tf(key, vars, fallback)` |
| `frontend/dist/ui/analysis.js` | chip/hint disk, i18n dashboard/analisis, re-render saat bahasa berubah |
| `frontend/dist/ui/history.js` | i18n status/aksi/jumlah + re-render |
| `frontend/dist/ui/nix.js`, `preview.js` | i18n status/loading/modal utama |
| `frontend/dist/ui/settings.js` | key ID/EN baru, dispatch `cleanlie:lang`, timer re-render |
| `.github/workflows/test.yml` | job `go`, `ui`, `build` |
| `frontend/test/ui-smoke.mjs` | mock `health`, assertion chip + teks EN dinamis |

## Alur

### Single instance

```
main ──runCLI(handled?)──> exit
  │
  v
singleinstance.Acquire ──> flock(LOCK_EX|LOCK_NB) pada <state>/cleanlie.lock
  │                                   │
  │ EWOULDBLOCK                       │ sukses
  v                                   v
stderr + exit 1                    wails.Run (defer Release)
```

CLI headless tidak mengambil lock supaya `--insight`, `--report`, `--trash-*`, dan
`--auto-clean` tetap bisa dipakai saat GUI terbuka.

### Disk health

```
syscall.Statfs("/") ──> Volume{total,free,used,health}
                              │
                              v
                    renderDisk: chip + hint
```

Ambang: `critical` <5% atau <2 GiB; `warn` <12% atau <10 GiB; `unknown` bila total
≤ 0/free negatif.

### i18n dinamis

```
settings.applyLanguage(lang)
  ├─ data-i18n statis
  ├─ renderSchedule(cache)
  └─ dispatch 'cleanlie:lang'
        ├─ analysis: disk, insight, report, tree, overview, dups
        ├─ history: sesi + trash
        └─ nix: status + dead paths
```

`__tf` mengganti placeholder `{n}`, `{size}`, `{days}`, dst. Label teknis (nama
rule, path, unit, profile) tetap apa adanya.

## Keputusan implementasi

1. **Lock hanya untuk GUI.** Mode CLI tidak boleh gagal karena GUI terbuka.
2. **File lock tidak dihapus saat release.** Inode tetap stabil; lock berikutnya
   memakai file yang sama.
3. **Lock failure non-fatal.** Jika state dir tidak bisa ditulis, GUI tetap jalan
   dengan warning di log (lebih baik daripada menolak start).
4. **Health dihitung di Go.** UI tidak menduplikasi ambang; fallback JS hanya untuk
   backend lama/mock.
5. **Helper `__tf` di `core.js`.** Semua modul sudah memuat core lebih dulu;
   fallback ke teks ID bila `__t` belum siap.
6. **Re-render ringan, bukan scan ulang.** Listener memakai data cache (`lastVolume`,
   `lastReport`, `dups`, `nixStatus`, sesi/trash) sehingga ganti bahasa tidak
   memicu I/O baru.
7. **Label teknis tidak diterjemahkan.** Nama aturan, path, profile, dan unit
   systemd tetap.
8. **CI memakai perintah lokal.** `nix develop --command ...` agar versi Go sama;
   UI smoke memakai Node 20 + `npm ci`.
9. **Build setelah test.** Job `build` bergantung pada `go` dan `ui`.
10. **Tidak ada SMART.** Indikator hanya kapasitas filesystem, bukan kesehatan
    hardware.

## Verifikasi

```sh
nix develop --command go test -tags desktop,production,webkit2_41 ./...  # 17 paket
nix develop --command go vet -tags desktop,production,webkit2_41 ./...
nix develop --command gofmt -l .                                        # kosong
cd frontend/test && npm test                                            # UI smoke OK
nix build .#cleanlie                                                    # 0.11.0
```

Uji manual single instance:

```sh
./result/bin/cleanlie &        # GUI pertama
./result/bin/cleanlie          # stderr: cleanlie: aplikasi sudah berjalan; exit 1
```

## Batasan

- Instance kedua tidak mem-fokuskan window lama (butuh D-Bus/aktivasi desktop).
- i18n dinamis mencakup teks utama dashboard/analisis/Riwayat/Nix/preview; pesan
  error dari Go dan label katalog tetap berbahasa Indonesia/teknis.
- Health hanya melihat filesystem root `/`; volume lain belum punya kartu sendiri.
- CI `build` mengunduh ulang dependensi Nix di runner (belum memakai Cachix).
