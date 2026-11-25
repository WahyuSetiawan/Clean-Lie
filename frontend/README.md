# frontend/

UI statis tanpa bundler: seluruh `dist/` di-embed ke binary lewat `//go:embed all:frontend/dist`.

- **Sumber desain:** [`../docs/wireframe/index.html`](../docs/wireframe/index.html) — acuan visual v1.
- **Implementasi:** `dist/index.html` mulai sebagai salinan wireframe, lalu bertahap disambungkan ke binding Go
  (`window.go.main.App.*`). Sejak M5 semua layar utama memakai data nyata; blok M6 menambah config,
  tema, dan i18n.
- Tidak ada `npm install` / build step. Edit HTML/CSS/JS langsung, lalu `wails dev` (hot reload) atau `make build`.
- Binding Go **tidak** perlu file JS generated: Wails menyuntik `window.go` saat runtime.
- Jangan tambahkan framework dulu (Svelte/React) kecuali ada kebutuhan nyata. UI tetap file statis, hanya
  dipecah per modul di `dist/ui/`.

## Layout `dist/`

```
index.html        # shell markup saja (sidebar, layar, modal, toast)
ui/app.css        # seluruh CSS (tema, komponen, layout)
ui/demo.js        # data demo wireframe + render tabel + routing + tema + modal + toast
ui/diagnostics.js # badge diagnosa runtime (M0)
ui/core.js        # bridge Wails, helper DOM (byId/esc/human/rebind/setOk), state scan + home
ui/cleanup.js     # layar Bersihkan (M2): scan, pilih, gate, eksekusi, hasil
ui/nix.js         # layar Nix (M3): store, generasi, dead paths, GC, vacuum
ui/android.js     # layar Android (M4): AVD, SDK, saran, hapus/uninstall/wipe
ui/screens.js     # boot layar M2–M4: preload data saat navigasi
ui/analysis.js    # Dashboard + Analisis (M5)
ui/settings.js    # Pengaturan, tema, i18n ID/EN (M6)
```

Urutan `<script>` di `index.html` adalah urutan dependensi (semua script klasik, bukan ES module):
`demo.js` mendefinisikan global yang dipakai modul lain (`toast`, `navigate`, `selected`, `ITEMS`, `MB`,
`setTheme`, `backdrop`); `core.js` wajib sebelum `cleanup/nix/android`; `screens.js` wajib setelah ketiganya
karena memanggil `ensureScan/ensureNix/ensureAndroid`. API lintas file lewat `window.cleanlie`
(`ensureScan`, `report`, `ensureNix`, `ensureAndroid`, `home`, `shortPath`, `esc`, `human`).
Jangan mengubah urutan tanpa mengecek pemakaian lintas file.
`frontend/dist/assets/` sengaja tetap di-ignore `.gitignore` (calon output bundler) — jangan taruh sumber di sana.
Aset baru wajib `git add`: `nix build` hanya menyalin file yang di-track. `go test .` menjalankan
`frontend_embed_test.go` yang gagal bila `index.html` merujuk aset yang tidak ikut ter-embed.

## Status per milestone

| Milestone | Isi UI |
|---|---|
| M0 | shell + navigasi + diagnosa runtime |
| M2 | layar Bersihkan tersambung ke scan & hapus |
| M3 | layar Nix: store, generasi, dead paths, GC, vacuum, fallback sudo |
| M4 | layar Android: AVD, SDK, saran >90 hari, hapus/uninstall/wipe |
| M5 | layar Analisis (tree, file besar, duplikat, file lama, artefak) + Dashboard nyata |
| M6 | Pengaturan nyata (simpan/reset, exclusion, ambang), tema system/light/dark, i18n ID/EN untuk shell/settings/dashboard/analysis |
