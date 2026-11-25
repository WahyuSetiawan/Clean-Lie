# frontend/

UI statis tanpa bundler: `dist/index.html` di-embed ke binary lewat `//go:embed all:frontend/dist`.

- **Sumber desain:** [`../docs/wireframe/index.html`](../docs/wireframe/index.html) — acuan visual v1.
- **Implementasi:** `dist/index.html` mulai sebagai salinan wireframe, lalu bertahap disambungkan ke binding Go
  (`window.go.main.App.*`). Sejak M5 semua layar utama memakai data nyata; blok M6 menambah config,
  tema, dan i18n.
- Tidak ada `npm install` / build step. Edit HTML/CSS/JS langsung, lalu `wails dev` (hot reload) atau `make build`.
- Binding Go **tidak** perlu file JS generated: Wails menyuntik `window.go` saat runtime.
- Jangan tambahkan framework dulu (Svelte/React) kecuali ada kebutuhan nyata — target v1 tetap satu file statis.

## Status per milestone

| Milestone | Isi UI |
|---|---|
| M0 | shell + navigasi + diagnosa runtime |
| M2 | layar Bersihkan tersambung ke scan & hapus |
| M3 | layar Nix: store, generasi, dead paths, GC, vacuum, fallback sudo |
| M4 | layar Android: AVD, SDK, saran >90 hari, hapus/uninstall/wipe |
| M5 | layar Analisis (tree, file besar, duplikat, file lama, artefak) + Dashboard nyata |
| M6 | Pengaturan nyata (simpan/reset, exclusion, ambang), tema system/light/dark, i18n ID/EN untuk shell/settings/dashboard/analysis |
