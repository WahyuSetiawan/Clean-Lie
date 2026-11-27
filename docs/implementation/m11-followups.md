# Follow-up M11 — aktivasi, volume, i18n Go, Cachix (implementasi)

Status: **selesai** · versi **0.12.0** · 2026-11-27

## Ringkas

Empat follow-up setelah M11: instance kedua memfokuskan GUI lama lewat D-Bus,
dashboard bisa memilih volume lokal (bukan hanya `/`), pesan Go untuk notifikasi
dan timer mengikuti bahasa, dan CI memakai Cachix bila token tersedia.

## File yang berubah

| File | Isi |
|---|---|
| `internal/singleinstance/focus.go` | `Activate`, `ServeFocus`, nama D-Bus `org.cleanlie.Cleanlie` |
| `internal/singleinstance/focus_test.go` | round-trip Focus (skip bila tidak ada session bus) |
| `app_focus.go` | `focusWindow`: show + unminimise + always-on-top sesaat |
| `main.go` | instance kedua memanggil `Activate`; server fokus saat GUI pertama jalan |
| `internal/fsx/mounts.go` | baca `/proc/self/mounts`, filter fs nyata, dedupe device |
| `internal/fsx/mounts_test.go` | parse/unescape, filter, root + health |
| `app_analysis.go` | binding `DiskVolumes()` |
| `frontend/dist/index.html` | select `dash-disk-select` |
| `frontend/dist/ui/analysis.js` | render volume, ganti volume tanpa scan ulang |
| `app_lang.go` | helper `msg(id,en)` dari config bahasa |
| `app_notify.go` | judul/body notifikasi ID/EN |
| `app_schedule.go` | detail timer ID/EN |
| `frontend/dist/ui/settings.js` | re-fetch status timer saat bahasa berubah |
| `.github/workflows/test.yml` | Cachix opsional di job go + build |
| `frontend/test/ui-smoke.mjs` | mock dua volume + assertion health non-root |

## Alur

### Aktivasi instance kedua

```
instance kedua
  └─ Acquire() → ErrRunning
       └─ Activate() → session bus org.cleanlie.Cleanlie.Focus
            ├─ sukses → exit 0
            └─ gagal  → stderr + exit 1

instance pertama
  └─ ServeFocus(app.focusWindow)
       └─ WindowUnminimise + WindowShow + AlwaysOnTop(250 ms)
```

Fallback tetap aman: bila session bus tidak ada, perilaku lama (pesan + exit 1)
dipertahankan.

### Volume dashboard

```
/proc/self/mounts
  → filter ext4/btrfs/xfs/f2fs/vfat/exfat/ntfs/zfs/bcachefs
  → statfs + dedupe st.Dev
  → []fsx.Mount{path, fstype, device, total, free, used, health}
  → select dashboard → renderDisk(mount)
```

Root `/` selalu di urutan pertama. Pseudo-fs, overlay, tmpfs, dan squashfs
diabaikan. Binding `DiskVolumes()` gagal → UI kembali ke `DiskStatus()`.

### i18n pesan Go

`app.msg(id,en)` membaca `config.Language`. Dipakai untuk:

- judul/body notifikasi pembersihan, analisis, direnv, GC, hapus generasi,
  vacuum journal;
- detail status timer (`timer aktif`, `unit ditulis`, error enable, dst.).

Pesan error teknis dari paket internal tetap apa adanya; UI tetap memberi prefix
yang sudah diterjemahkan.

### CI Cachix

Job `go` dan `build` menjalankan `cachix/cachix-action` hanya bila
`secrets.CACHIX_AUTH_TOKEN` tersedia. PR dari fork tanpa secret tetap lolos
karena langkah Cachix di-skip, bukan digagalkan.

## Keputusan implementasi

1. **D-Bus, bukan socket ad-hoc.** Session bus sudah tersedia di desktop Linux;
   tidak perlu daemon tambahan dan tidak membuka port.
2. **Nama D-Bus tetap.** `org.cleanlie.Cleanlie` + path `/org/cleanlie/Cleanlie`
   agar instance kedua tidak perlu membaca state tambahan.
3. **Kegagalan D-Bus non-fatal.** GUI pertama tetap jalan tanpa aktivasi; instance
   kedua kembali ke pesan stderr + exit 1.
4. **Fokus best-effort.** Tidak ada API `WindowFocus` di Wails v2; show +
   unminimise + always-on-top sesaat adalah kompromi lintas X11/Wayland.
5. **Hanya fs nyata.** Mount pseudo/container tidak ditampilkan agar tidak
   membingungkan dan tidak memicu statfs pada path aneh.
6. **Dedupe device.** Bind mount dari device yang sama tidak muncul dua kali.
7. **Root selalu pertama.** Select default tetap `/`; pengguna bisa memilih
   volume lain tanpa mengubah config.
8. **i18n Go terbatas pada pesan produk.** Error teknis tidak diterjemahkan agar
   tetap bisa dicari di log/issue.
9. **Cachix opsional.** Workflow tidak boleh gagal hanya karena secret tidak
   tersedia di PR/fork.
10. **Tidak ada perubahan format audit/trend/report.**

## Verifikasi

```sh
nix develop --command go test -tags desktop,production,webkit2_41 ./...  # 17 paket
nix develop --command go vet -tags desktop,production,webkit2_41 ./...
nix develop --command gofmt -l .                                        # kosong
cd frontend/test && npm test                                            # UI smoke OK
nix run nixpkgs#yamllint -- .github/workflows/test.yml                  # bersih
nix build .#cleanlie                                                    # 0.12.0
```

Uji manual:

```sh
./result/bin/cleanlie &   # GUI pertama, mendaftarkan nama D-Bus
./result/bin/cleanlie     # exit 0, window lama tampil/fokus
```

Dashboard: pilih `/mnt/data` di select volume → chip health berubah sesuai
kapasitas volume tersebut.

## Batasan

- Fokus window bergantung pada compositor; Wayland bisa mengabaikan permintaan
  always-on-top/focus.
- Volume yang tidak ada di `/proc/self/mounts` (mis. FUSE pengguna) tidak tampil.
- Pesan error dari `internal/clean`/`internal/schedule` masih berbahasa Indonesia
  atau mengikuti error OS.
- Cachix hanya aktif bila secret repository tersedia.
