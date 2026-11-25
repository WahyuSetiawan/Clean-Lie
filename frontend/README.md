# frontend/

UI statis tanpa bundler: `dist/index.html` di-embed ke binary lewat `//go:embed all:frontend/dist`.

- **Sumber desain:** [`../docs/wireframe/index.html`](../docs/wireframe/index.html) — acuan visual v1.
- **Implementasi:** `dist/index.html` mulai sebagai salinan wireframe, lalu bertahap disambungkan ke binding Go
  (`window.go.main.App.*`). Di M0 hanya `Diagnostics()` yang benar-benar memanggil Go; sisanya masih data contoh.
- Tidak ada `npm install` / build step. Edit HTML/CSS/JS langsung, lalu `wails dev` (hot reload) atau `make build`.
- Binding Go **tidak** perlu file JS generated: Wails menyuntik `window.go` saat runtime.
- Jangan tambahkan framework dulu (Svelte/React) kecuali ada kebutuhan nyata — target v1 tetap satu file statis.

## Status per milestone

| Milestone | Isi UI |
|---|---|
| M0 | shell + navigasi + diagnosa runtime |
| M2 | layar Bersihkan tersambung ke scan & hapus |
| M3–M5 | Nix, Android, analisis, dashboard tersambung |
| M6 | settings persist, i18n, tema, empty/error state |
