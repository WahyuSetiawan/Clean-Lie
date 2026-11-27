# M10 — Otomasi & Cakupan (implementasi)

Status: **selesai** · versi **0.10.0** · 2026-11-27

## Ringkas

M10 menutup tiga celah: folder cache yang tidak ada di katalog, hardlink yang
tampak seperti duplikat biasa, dan pembersihan yang hanya bisa dijalankan manual.

## File yang berubah

| File | Isi |
|---|---|
| `internal/rules/custom.go` | `CustomRule`, `CustomRules`, validasi + group `custom` |
| `internal/config/config.go` | `customRules` + normalisasi (maks 32, protected, risk, depth) |
| `custom_rules.go` | mapping `config.CustomRule` → `rules.Rule` |
| `internal/analysis/{analysis,duplicates}.go` | `hardlinks`/`hardlinkBytes` |
| `internal/schedule/schedule.go` | unit user systemd + install/remove/status |
| `app_schedule.go` | binding `ScheduleStatus/Install/Remove` |
| `cli.go` | `--auto-clean`, `--schedule-*`, `--interval` |
| `frontend/dist/index.html`, `ui/settings.js` | textarea aturan kustom + kartu Otomasi |
| `frontend/dist/ui/analysis.js` | info hardlink di ringkasan duplikat |
| `frontend/test/ui-smoke.mjs` | mock schedule + assertion aturan/timer |

## Alur data

```
config.json ──customRules──> config.Normalize (protected, max 32)
                                   │
                                   v
                         customRules() ──> rules.Plan ──> ScanJunk / --auto-clean
                                                               │
                                          --auto-clean: safe + !root + group app/dev/custom
                                                               v
                                                   clean.Execute(mode=trash)
                                                               │
                                                               v
                                                   audit batch auto-<timestamp>
```

Timer:

```
GUI/CLI ──> schedule.Install ──> ~/.config/systemd/user/cleanlie-clean.{service,timer}
                                        │
                                        v
                              systemctl --user enable --now
                                        │
                                        v
                     cleanlie --auto-clean --yes --json (Nice 10, idle IO)
```

## Keputusan implementasi

1. **Config memvalidasi, rules memetakan.** `config.Normalize` menolak path
   protected dan membatasi 32 aturan supaya UI tidak menyimpan config berbahaya;
   `rules.CustomRules` mengulang validasi sebagai pertahanan kedua (mis. config
   lama/manual).
2. **Aturan kustom bukan `safe` default.** Default `warn`; pengguna harus memilih
   `safe` secara eksplisit agar ikut auto-clean.
3. **`whole` eksplisit.** Folder default memakai anak depth 1; file tunggal harus
   memakai `whole` agar tidak menghasilkan item kosong.
4. **Hardlink dihitung saat inode sama dilewati.** `wasted` tetap hanya salinan
   nyata; `hardlinkBytes` adalah ukuran logical yang tidak bisa dibebaskan.
5. **Auto-clean dibatasi group.** Hanya `app`, `dev.*`, `custom`; `/var/tmp` dan
   log sistem tidak pernah ikut walaupun risk-nya `safe`.
6. **`--yes` wajib.** Tanpa `--yes` exit 2; `--dry-run` tetap boleh tanpa `--yes`.
7. **Selalu trash.** Tidak ada flag permanen di `--auto-clean`; `clean.Execute`
   juga memaksa non-permanent ke trash.
8. **Timer user, bukan root.** Unit ditulis ke `$XDG_CONFIG_HOME/systemd/user`;
   `systemctl --user`; `NoNewPrivileges=true`, `PrivateTmp=true`.
9. **Systemctl opsional.** `CLEANLIE_NO_SYSTEMCTL=1` hanya menulis unit (dipakai
   test); kegagalan systemctl tidak menghapus unit, hanya dicatat di `detail`.
10. **`%` di-escape.** Path binary dengan `%` menjadi `%%` agar tidak dibaca
    systemd sebagai specifier.

## Verifikasi

```sh
go test -tags desktop,production,webkit2_41 ./...   # 16 paket
go vet ./...
gofmt -l .
cd frontend/test && npm test                       # UI smoke
nix build .#cleanlie                               # 0.10.0
./result/bin/cleanlie --auto-clean --dry-run --json
./result/bin/cleanlie --schedule-install --dry-run
./result/bin/cleanlie --schedule-status
```

## Batasan

- Reflink tidak dideteksi; hanya hardlink.
- Timer user berhenti saat logout kecuali `loginctl enable-linger` diaktifkan
  (butuh root; tidak dilakukan otomatis).
- Aturan kustom tidak punya preview khusus; validasi muncul sebagai warning di
  Pengaturan setelah disimpan.
- Auto-clean tidak menjalankan `gio trash` bila `CLEANLIE_NO_GIO=1`; fallback
  manual memindahkan ke `~/.local/share/Trash`.
