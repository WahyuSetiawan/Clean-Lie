# UI smoke test (opsional)

Test ini menjalankan `dist/index.html` di [jsdom](https://github.com/jsdom/jsdom) dengan
binding Go tiruan. Tidak dipakai saat build (UI tetap statis tanpa Node); hanya untuk
regresi alur M2: scan → pilih → konfirmasi → hasil.

```sh
cd frontend/test
npm install
npm test
```

Yang dijaga:

- `window.go` (binding Wails) tidak boleh tertimpa fungsi navigasi global.
- hanya item `safe` yang otomatis dicentang; item `requiresRoot` terkunci.
- hapus permanen terkunci sampai checkbox konfirmasi dicentang.
- `Clean` menerima ID dari hasil scan, bukan path dari JS.
- hasil menampilkan `freed` dan memicu scan ulang.
