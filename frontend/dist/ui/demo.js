/* ---------------- data (angka hasil ukur nyata) ---------------- */
const MB = 1024;
const fmt = m => m >= 1024 ? (m/1024).toFixed(1)+' GB' : Math.round(m)+' MB';

const ITEMS = [
  // node
  {g:'node', p:'~/.npm/_cacache', mb:7168, risk:'safe', note:'npm cache clean --force'},
  {g:'node', p:'~/.npm/_npx', mb:2662, risk:'safe', note:'cache npx, regenerate'},
  {g:'node', p:'~/.cache/pnpm', mb:2150, risk:'safe', note:'pnpm store prune'},
  {g:'node', p:'~/.local/share/pnpm/store', mb:1229, risk:'safe', note:'pnpm store prune (jangan rm)'},
  {g:'node', p:'~/.cache/ms-playwright', mb:1946, risk:'warn', note:'browser test harus diunduh ulang'},
  {g:'node', p:'~/.cache/Cypress', mb:819, risk:'warn', note:'versi Cypress harus diunduh ulang'},
  // jvm
  {g:'jvm', p:'~/.gradle/caches', mb:3584, risk:'safe', note:'sisakan versi yang dipakai proyek'},
  {g:'jvm', p:'~/.gradle/wrapper/dists', mb:1024, risk:'safe', note:'dist Gradle lama, re-download'},
  // android
  {g:'android', p:'~/.android/avd', mb:14541, risk:'danger', note:'2 AVD — data & snapshot hilang'},
  {g:'android', p:'~/Android/Sdk/system-images', mb:7987, risk:'danger', note:'sdkmanager --uninstall'},
  {g:'android', p:'~/Android/Sdk/ndk', mb:4198, risk:'danger', note:'build native gagal sampai re-install'},
  {g:'android', p:'~/Android/Sdk/emulator', mb:819, risk:'danger', note:'emulator binary, unduh ulang'},
  {g:'android', p:'~/Android/Sdk/platforms', mb:717, risk:'danger', note:'platform API, dipakai compileSdk'},
  {g:'android', p:'~/Android/Sdk/build-tools', mb:410, risk:'warn', note:'versi lama bisa dihapus'},
  // python
  {g:'python', p:'~/.cache/uv', mb:1536, risk:'safe', note:'uv cache prune'},
  {g:'python', p:'~/.local/share/uv', mb:307, risk:'safe', note:'uv cache prune'},
  {g:'python', p:'~/.cache/pip', mb:102, risk:'safe', note:'pip cache purge'},
  // dart
  {g:'dart', p:'~/.pub-cache/hosted', mb:1536, risk:'safe', note:'dart pub cache clean'},
  // rustgo
  {g:'rustgo', p:'~/.cargo/registry', mb:512, risk:'safe', note:'hapus cache + src, bukan bin'},
  {g:'rustgo', p:'~/go/pkg/mod', mb:410, risk:'safe', note:'go clean -modcache'},
  {g:'rustgo', p:'~/.cache/gopls', mb:20, risk:'safe', note:'language server cache'},
  // app
  {g:'app', p:'~/.cache/opencode', mb:3072, risk:'safe', note:'regenerate otomatis'},
  {g:'app', p:'~/.cache/qmd', mb:2355, risk:'safe', note:'index embedding'},
  {g:'app', p:'~/.cache/phpactor', mb:2253, risk:'safe', note:'index PHP'},
  {g:'app', p:'~/.cache/codebase-memory-mcp', mb:1024, risk:'safe', note:'index kode'},
  {g:'app', p:'~/.cache/nix', mb:905, risk:'safe', note:'eval cache Nix'},
  {g:'app', p:'~/.cache/BraveSoftware', mb:1229, risk:'blocked', note:'Brave sedang berjalan'},
  {g:'app', p:'~/.cache/mozilla', mb:734, risk:'warn', note:'tutup Firefox sebelum hapus'},
  // log
  {g:'log', p:'/var/log/journal', mb:4096, risk:'warn', note:'journalctl --vacuum-size=500M'},
  {g:'log', p:'/var/log (rotasi >30h)', mb:120, risk:'safe', note:'hanya *.old/*.gz'},
  {g:'log', p:'/var/tmp (>30 hari)', mb:60, risk:'safe', note:'file milik user'},
  // build
  {g:'build', p:'~/Projects/*/node_modules', mb:6200, risk:'warn', note:'12 proyek, 9 tidak aktif'},
  {g:'build', p:'~/Projects/*/target', mb:3100, risk:'warn', note:'Rust build cache'},
  {g:'build', p:'~/Projects/*/.venv', mb:900, risk:'warn', note:'venv dibuat ulang'},
  {g:'build', p:'~/Projects/*/dist', mb:700, risk:'warn', note:'hasil build'},
  {g:'build', p:'~/Projects/*/__pycache__', mb:400, risk:'safe', note:'bytecode Python'},
  {g:'build', p:'~/Projects/*/.next', mb:300, risk:'warn', note:'cache Next.js'},
  // trash
  {g:'trash', p:'~/.local/share/Trash/files', mb:2.2, risk:'safe', note:'kosongkan trash'},
];

const TITLES = {
  node:['Node (npm/pnpm)','Perintah resmi dijalankan lebih dulu (npm cache clean, pnpm store prune). Hapus folder hanya fallback.'],
  jvm:['JVM (gradle)','Gradle caches per versi. Versi yang dipakai proyek aktif tidak disarankan dihapus.'],
  android:['Android SDK/AVD','Default tidak dicentang. Hapus = data emulator hilang & system image harus diunduh ulang.'],
  python:['Python (uv/pip)','uv cache prune / pip cache purge. Aman, paket diunduh ulang saat dibutuhkan.'],
  dart:['Dart / Flutter','dart pub cache clean. Paket diunduh ulang saat pub get.'],
  rustgo:['Rust / Go','Hapus registry & modcache; bin/ dan toolchain tidak disentuh.'],
  app:['Cache aplikasi','Cache yang dibuat ulang otomatis. Browser di-skip kalau prosesnya sedang jalan.'],
  log:['Log & journal','Journal di-vacuum ke target; rotasi lama dihapus. Butuh root untuk journal system.'],
  build:['Artefak build','Daftar per proyek. Proyek dengan mtime < 7 hari ditandai aktif dan tidak dicentang.'],
  trash:['Trash','Isi trash. Setelah dikosongkan tidak bisa di-restore.'],
};
const RISK = {safe:['aman','safe'], warn:['hati-hati','warn'], danger:['berisiko','danger'], blocked:['terkunci','locked']};

let activeCat = 'node';
const selected = new Set();

function renderCleanup(){
  const showRisky = !window.__getSettings || window.__getSettings().showRisky;
  const rows = ITEMS.filter(i => i.g === activeCat && (showRisky || i.risk === 'safe'));
  const [title, note] = TITLES[activeCat];
  document.getElementById('cat-title').textContent = title;
  document.getElementById('cat-sub').textContent = rows.length + ' item · ' + fmt(rows.reduce((a,b)=>a+b.mb,0));
  document.getElementById('cat-note').textContent = note;

  document.getElementById('rows').innerHTML = rows.map((it,idx) => {
    const [label, cls] = RISK[it.risk];
    const id = activeCat + '-' + idx;
    const checked = selected.has(it.p) ? 'checked' : '';
    const disabled = it.risk === 'blocked' ? 'disabled' : '';
    return `<tr>
      <td><input type="checkbox" class="check" data-path="${it.p}" data-mb="${it.mb}" id="${id}" ${checked} ${disabled}></td>
      <td class="path mono">${it.p}</td>
      <td class="r num">${fmt(it.mb)}</td>
      <td><span class="chip ${cls}">${label}</span></td>
      <td class="small muted">${it.note}</td>
    </tr>`;
  }).join('');

  document.querySelectorAll('#rows .check').forEach(cb => {
    cb.addEventListener('change', () => {
      if (cb.checked) selected.add(cb.dataset.path); else selected.delete(cb.dataset.path);
      updateTotals();
    });
  });
  updateTotals();
}

function updateTotals(){
  let mb = 0;
  for (const it of ITEMS) if (selected.has(it.p)) mb += it.mb;
  const n = selected.size;
  document.getElementById('sel-count').textContent = n;
  document.getElementById('sel-size').textContent = fmt(mb);
  document.getElementById('actionbar').hidden = (current !== 'cleanup');
}

/* ---------------- routing ---------------- */
const META = {
  dashboard:['Dashboard','Ringkasan disk dan reclaimable space'],
  cleanup:['Bersihkan','Pilih item per kategori · default hanya item aman'],
  nix:['Nix','Store, generasi, dan garbage collection'],
  android:['Android','AVD, system image, dan NDK'],
  analysis:['Analisis','Disk usage, file besar, duplikat, file lama, artefak build'],
  settings:['Pengaturan','Preferensi pembersihan dan tampilan'],
};
let current = 'dashboard';

function navigate(view){
  current = view;
  document.querySelectorAll('.view').forEach(v => v.classList.toggle('active', v.id === 'view-' + view));
  document.querySelectorAll('#nav button').forEach(b => b.classList.toggle('active', b.dataset.view === view));
  const [t,s] = META[view] || ['Cleanlie',''];
  const tr = k => (window.__t ? window.__t(k) : null);
  document.getElementById('page-title').textContent = tr('page.' + view + '.title') || t;
  document.getElementById('page-sub').textContent = tr('page.' + view + '.sub') || s;
  document.getElementById('actionbar').hidden = view !== 'cleanup';
  if (view === 'cleanup') updateTotals();
  if (location.hash !== '#/'+view) history.replaceState(null,'','#/'+view);
}

document.querySelectorAll('#nav button').forEach(b => b.addEventListener('click', () => navigate(b.dataset.view)));
document.querySelectorAll('.view button[data-cat]').forEach(b => b.addEventListener('click', () => {
  activeCat = b.dataset.cat;
  document.querySelectorAll('.view button[data-cat]').forEach(x => x.classList.toggle('active', x === b));
  renderCleanup();
}));

/* select all safe */
document.getElementById('select-all').addEventListener('click', () => {
  ITEMS.filter(i => i.g === activeCat && i.risk === 'safe').forEach(i => selected.add(i.p));
  renderCleanup();
});
document.getElementById('clear-sel').addEventListener('click', () => { selected.clear(); renderCleanup(); });

/* analysis tabs */
document.querySelectorAll('#tabs button').forEach(b => b.addEventListener('click', () => {
  document.querySelectorAll('#tabs button').forEach(x => x.classList.toggle('active', x === b));
  document.querySelectorAll('.tabpane').forEach(p => { p.hidden = p.id !== 'tab-' + b.dataset.tab; });
}));

/* android suggestion ditangani oleh blok M4 di bawah */

/* toggles + theme */
const themeSelect = document.getElementById('theme-select');
const systemDark = window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)') : null;
function resolveTheme(mode){ return mode === 'system' ? (systemDark && systemDark.matches ? 'dark' : 'light') : mode; }
function setTheme(mode){ document.documentElement.dataset.themeMode = mode; document.documentElement.dataset.theme = resolveTheme(mode); if (themeSelect) themeSelect.value = mode; }
document.getElementById('theme-btn').addEventListener('click', () => {
  const next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  setTheme(next);
  if (window.__themeChanged) window.__themeChanged(next);
});
if (themeSelect) themeSelect.addEventListener('change', e => {
  setTheme(e.target.value);
  if (window.__themeChanged) window.__themeChanged(e.target.value);
});
if (systemDark && systemDark.addEventListener) systemDark.addEventListener('change', () => {
  if (document.documentElement.dataset.themeMode === 'system') setTheme('system');
});
setTheme(document.documentElement.dataset.themeMode || 'system');

/* ---------------- modal ---------------- */
const backdrop = document.getElementById('backdrop');
function openModal(kind){
  const clean = kind === 'confirm-clean';
  const gc = kind === 'confirm-gc';
  document.getElementById('m-title').textContent = clean ? `Pindahkan ${selected.size || 6} item ke Trash?` : 'Jalankan garbage collection?';
  document.getElementById('m-sub').textContent = clean
    ? 'Item masuk ke trash freedesktop dan bisa di-restore. Tidak ada file yang dihapus permanen.'
    : 'Dry-run dulu, lalu ketik GC untuk mengonfirmasi. Tindakan ini tidak bisa dibatalkan.';

  let mb = 0; for (const it of ITEMS) if (selected.has(it.p)) mb += it.mb;
  document.getElementById('m-body').innerHTML = clean ? `
    <div class="row" style="gap:8px;margin-bottom:10px">
      <span class="pill">${selected.size || 6} item</span>
      <span class="pill"><b>${fmt(mb || 15400)}</b> dibebaskan</span>
      <span class="pill">mode: <b>trash</b></span>
    </div>
    <div class="cmdbox">nix-store --gc --dry-run
# dead paths: 34.5 GB</div>
    <div class="tiny dim" style="margin-top:8px">Perintah di atas hanya contoh preview. Setelah eksekusi, freed diukur ulang (before − after), bukan diasumsikan.</div>
  ` : `
    <div class="row" style="gap:8px;margin-bottom:10px">
      <span class="pill">store: <b>82.0 G</b></span>
      <span class="pill">dead: <b>34.5 G</b></span>
      <span class="pill">estimasi sisa: <b>47.5 G</b></span>
    </div>
    <div class="cmdbox">$ nix-collect-garbage --dry-run
finding garbage collector roots...
34.5 GiB will be freed

$ sudo nix-collect-garbage -d   # setelah konfirmasi</div>
    <div class="field" style="border:0;padding:12px 0 0">
      <div class="label"><b>Ketik GC untuk melanjutkan</b><span>Generasi current tidak akan disentuh</span></div>
      <div class="ctrl"><input type="text" id="gc-input" placeholder="GC" autocomplete="off" style="width:90px"></div>
    </div>
  `;
  const ok = document.getElementById('m-ok');
  ok.textContent = clean ? 'Ya, pindahkan' : 'Jalankan GC';
  ok.disabled = gc;
  if (gc){
    const inp = document.getElementById('gc-input');
    inp.addEventListener('input', () => { ok.disabled = inp.value.trim() !== 'GC'; });
    setTimeout(() => inp.focus(), 30);
  }
  backdrop.classList.add('open');
}
document.querySelectorAll('[data-open]').forEach(b => b.addEventListener('click', () => openModal(b.dataset.open)));
document.getElementById('m-cancel').addEventListener('click', () => backdrop.classList.remove('open'));
backdrop.addEventListener('click', e => { if (e.target === backdrop) backdrop.classList.remove('open'); });
document.addEventListener('keydown', e => { if (e.key === 'Escape') backdrop.classList.remove('open'); });
document.getElementById('m-ok').addEventListener('click', () => {
  backdrop.classList.remove('open');
  toast('Simulasi wireframe — tidak ada file yang dihapus.');
});

/* ---------------- toast ---------------- */
let toastTimer;
function toast(msg){
  const t = document.getElementById('toast');
  t.textContent = msg; t.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove('show'), 2600);
}

/* ---------------- boot ---------------- */
// ukuran kategori + total item aman dihitung dari dataset (tidak hardcode)
document.querySelectorAll('.cat').forEach(btn => {
  const g = btn.dataset.cat;
  const mb = ITEMS.filter(i => i.g === g).reduce((a,b) => a + b.mb, 0);
  const el = btn.querySelector('.sz');
  if (el) el.textContent = mb >= 1024 ? (mb/1024).toFixed(1) + ' G' : Math.round(mb) + ' M';
});
const safeMb = ITEMS.filter(i => i.risk === 'safe').reduce((a,b) => a + b.mb, 0);
const qs = document.getElementById('quick-size');
if (qs) qs.textContent = (safeMb/1024).toFixed(1) + ' GB';

// hasil scan default: item aman di kategori pertama sudah dicentang
ITEMS.filter(i => i.g === 'node' && i.risk === 'safe').forEach(i => selected.add(i.p));
renderCleanup();
const initial = (location.hash || '#/dashboard').slice(2);
navigate(META[initial] ? initial : 'dashboard');
