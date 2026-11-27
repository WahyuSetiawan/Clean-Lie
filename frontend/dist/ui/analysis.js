/* ================= M5: Analisis + Dashboard ================= */
(() => {
  const bridge = window.go && window.go.main && window.go.main.App;
  if (!bridge) return;

  const byId = id => document.getElementById(id);
  const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const human = b => b >= 1073741824 ? (b / 1073741824).toFixed(2) + ' GiB'
    : b >= 1048576 ? (b / 1048576).toFixed(0) + ' MiB'
    : b >= 1024 ? (b / 1024).toFixed(0) + ' KiB' : b + ' B';
  const errMsg = e => typeof e === 'string' ? e : (e && e.message) || String(e);

  let home = '';
  let dashBusy = false;
  let tree = null, overview = null, dups = null;
  let treePath = '';
  let treeSeq = 0;
  const loaded = { tree: false, overview: false, dups: false };
  const sel = new Set();
  let activeTab = 'tree';

  async function ensureHome() {
    if (home) return home;
    if (window.cleanlie && window.cleanlie.home()) {
      home = window.cleanlie.home();
      return home;
    }
    try {
      const d = await bridge.Diagnostics();
      home = d.home || '';
    } catch (e) { home = ''; }
    return home;
  }

  const short = p => home && p && p.startsWith(home + '/') ? '~' + p.slice(home.length) : (p || '');

  /* ---------------- dashboard ---------------- */
  function setDash(id, text) { const el = byId(id); if (el) el.textContent = text; }

  function renderDisk(vol) {
    const used = vol.used || 0, total = vol.total || 1, pct = Math.min(100, Math.round(used / total * 100));
    setDash('dash-disk-used', human(used));
    setDash('dash-disk-total', human(total));
    setDash('dash-disk-hint', pct + '% terpakai · sisa ' + human(vol.free || 0));
    const bar = byId('dash-disk-bar'); if (bar) bar.style.width = pct + '%';
  }

  function renderNix(st) {
    setDash('dash-nix-size', human(st.storeBytes || 0));
    setDash('dash-nix-dead', (st.generations || []).length + ' generasi · ' + (st.profiles || []).length + ' profil');
    setDash('dash-nix-gens', 'journal ' + human(st.journalBytes || 0) + ' · ' + (st.storePaths || 0).toLocaleString('id-ID') + ' path');
  }

  function renderSpark(points) {
    const svg = byId('dash-spark');
    if (!svg) return;
    if (!points || points.length < 2) {
      svg.innerHTML = '<text x="50" y="21" text-anchor="middle" font-size="8" fill="#64748b">belum ada data tren</text>';
      return;
    }
    const vals = points.map(p => Number(p.free) || 0);
    const min = Math.min(...vals), max = Math.max(...vals);
    const span = Math.max(max - min, 1);
    const pts = vals.map((v, i) => {
      const x = (i / (vals.length - 1)) * 100;
      const y = 34 - ((v - min) / span) * 30;
      return x.toFixed(2) + ',' + y.toFixed(2);
    }).join(' ');
    svg.innerHTML = '<polyline points="' + pts + '" fill="none" stroke="var(--accent)" stroke-width="1.6" vector-effect="non-scaling-stroke"/>';
  }

  function renderInsight(rep) {
    if (!rep) return;
    const w = rep.week || {};
    setDash('dash-week-freed', human(w.freedBytes || 0));
    setDash('dash-week-moved', human(w.movedBytes || 0) + ' · ' + (w.removed || 0) + ' item');
    setDash('dash-week-sessions', String(w.sessions || 0));
    setDash('dash-week-trash', (w.trashEntries || 0) + ' · ' + human(w.trashBytes || 0));
    const trend = rep.trend || [];
    setDash('dash-week-range', trend.length ? trend[0].date + ' → ' + trend[trend.length - 1].date : 'belum ada data');
    renderSpark(trend);
  }

  async function doExport() {
    if (!bridge || !bridge.ExportReport) { toast('Ekspor hanya tersedia di aplikasi.'); return; }
    const sel = byId('dash-export-format');
    const btn = byId('dash-export');
    const format = sel ? sel.value : 'md';
    if (btn) { btn.disabled = true; btn.textContent = 'Mengekspor…'; }
    try {
      const path = await bridge.ExportReport(format);
      setDash('dash-export-path', path || '');
      toast('Laporan diekspor: ' + path);
    } catch (e) {
      toast('Ekspor gagal: ' + errMsg(e));
    } finally {
      if (btn) { btn.disabled = false; btn.textContent = 'Ekspor'; }
    }
  }

  function renderHome(tr) {
    setDash('dash-home-size', human(tr.bytes) + ' · ' + (tr.files || 0).toLocaleString('id-ID') + ' file');
    const map = byId('dash-map'), legend = byId('dash-legend');
    if (!map || !legend) return;
    const kids = (tr.root && tr.root.children || []).filter(n => n.bytes > 0).slice(0, 5);
    const total = Math.max(tr.bytes, 1);
    const colors = ['var(--accent)', 'var(--accent-2)', 'var(--warn)', 'var(--danger)', '#64748b'];
    let html = '', lg = '', used = 0;
    kids.forEach((n, i) => {
      used += n.bytes;
      const pct = Math.max(3, Math.round(n.bytes / total * 100));
      html += '<i title="' + esc(short(n.path)) + ' · ' + human(n.bytes) + '" style="width:' + pct + '%;background:' + colors[i] + '"><span>' + esc(n.name) + '</span></i>';
      lg += '<span><i class="dot" style="background:' + colors[i] + '"></i>' + esc(n.name) + ' <b>' + human(n.bytes) + '</b></span>';
    });
    const rest = Math.max(0, total - used);
    if (rest > 0) {
      const pct = Math.max(3, Math.round(rest / total * 100));
      html += '<i title="lain-lain · ' + human(rest) + '" style="width:' + pct + '%;background:#64748b"><span>lain</span></i>';
      lg += '<span><i class="dot" style="background:#64748b"></i>Lain-lain <b>' + human(rest) + '</b></span>';
    }
    map.innerHTML = html || '<span class="tiny dim">tidak ada data</span>';
    legend.innerHTML = lg;
  }

  function renderReport(r) {
    if (!r) return;
    let cache = 0, log = 0, tools = [];
    for (const g of r.groups || []) {
      if (g.id === 'sys.log') log += g.bytes || 0;
      if (g.id.startsWith('dev.') && g.id !== 'dev.android') {
        cache += g.bytes || 0;
        tools.push(g);
      }
    }
    setDash('dash-cache-size', human(cache));
    setDash('dash-log-size', human(log));
    const cacheHint = tools.slice(0, 3).map(g => g.id.split('.').pop() + ' ' + human(g.bytes)).join(' · ');
    setDash('dash-cache-hint', cacheHint || 'tidak ada cache terdeteksi');
    setDash('dash-log-hint', 'journal + /var/log · target 500 MB');
    const bar = byId('dash-cache-bar');
    if (bar) bar.style.width = Math.min(100, Math.round(cache / Math.max(r.bytes, 1) * 100)) + '%';
    setDash('dash-tools', tools.length + ' tool dev terdeteksi');
    setDash('dash-tools-detail', tools.slice(0, 6).map(g => g.id.split('.').pop()).join(' · ') || '—');

    const items = [];
    for (const g of r.groups || []) for (const it of g.items || []) {
      if (it.requiresRoot) continue;
      items.push({ it, g });
    }
    items.sort((a, b) => b.it.bytes - a.it.bytes);
    const top = items.slice(0, 5);
    const body = byId('dash-reclaim');
    if (!body) return;
    if (!top.length) {
      body.innerHTML = '<tr><td colspan="4" class="tiny dim" style="padding:14px">Tidak ada item.</td></tr>';
      return;
    }
    body.innerHTML = top.map(({ it, g }) => {
      const pct = Math.max(3, Math.round(it.bytes / Math.max(top[0].it.bytes, 1) * 100));
      const chip = it.risk === 'danger' ? 'danger' : it.risk === 'warn' ? 'warn' : 'safe';
      return '<tr><td style="width:1%"><span class="chip ' + chip + '">' + esc(g.id.split('.').pop()) + '</span></td>'
        + '<td class="path mono" title="' + esc(short(it.path)) + '">' + esc(short(it.path)) + '</td>'
        + '<td class="r num"><b>' + human(it.bytes) + '</b></td>'
        + '<td class="r" style="width:130px"><div class="sizebar"><i style="width:' + pct + '%"></i></div></td></tr>';
    }).join('');
  }

  async function refreshDashboard(force) {
    if (dashBusy) return;
    dashBusy = true;
    const btn = byId('dash-refresh');
    if (btn) { btn.disabled = true; btn.textContent = 'Memindai…'; }
    try {
      await ensureHome();
      const volP = bridge.DiskStatus().then(renderDisk).catch(() => {});
      const nixP = bridge.NixStatus().then(renderNix).catch(() => {});
      const treeP = bridge.AnalysisTree(home, 1).then(renderHome).catch(() => {});
      const insightP = (bridge.Insight ? bridge.Insight() : Promise.resolve(null)).then(renderInsight).catch(() => {});
      const r = await window.cleanlie.ensureScan(!!force);
      renderReport(r || window.cleanlie.report());
      await Promise.allSettled([volP, nixP, treeP, insightP]);
    } catch (e) {
      toast('Dashboard gagal: ' + errMsg(e));
    } finally {
      dashBusy = false;
      if (btn) { btn.disabled = false; btn.textContent = 'Scan ulang'; }
    }
  }

  /* ---------------- analysis: tree ---------------- */
  function loadingBody(id, cols, text) {
    const el = byId(id);
    if (el) el.innerHTML = '<tr><td colspan="' + cols + '" class="tiny dim" style="padding:18px 16px">' + esc(text) + '</td></tr>';
  }

  async function loadTree(path) {
    const seq = ++treeSeq;
    treePath = path;
    loadingBody('an-tree-body', 3, 'Memuat disk usage…');
    const map = byId('an-map'); if (map) map.innerHTML = '<div class="small dim">Memuat…</div>';
    try {
      const tr = await bridge.AnalysisTree(path, 1);
      if (seq !== treeSeq) return;
      tree = tr;
      treePath = tr.root.path;
      loaded.tree = true;
      renderTree();
    } catch (e) {
      if (seq !== treeSeq) return;
      loadingBody('an-tree-body', 3, 'Gagal: ' + errMsg(e));
    }
  }

  function renderTree() {
    if (!tree) return;
    const tr = tree, total = Math.max(tr.bytes, 1);
    const kids = tr.root.children || [];
    const body = byId('an-tree-body');
    if (body) {
      body.innerHTML = kids.length ? kids.map(n => {
        const pct = Math.max(2, Math.round(n.bytes / total * 100));
        const icon = n.dir ? '▸ ' : '';
        return '<tr' + (n.dir ? ' class="anrow" data-an-path="' + esc(n.path) + '"' : '') + '>'
          + '<td class="path mono" title="' + esc(n.path) + '">' + icon + esc(n.name) + (n.dir ? '/' : '') + '</td>'
          + '<td class="r num">' + human(n.bytes) + '</td>'
          + '<td><div class="sizebar"><i style="width:' + pct + '%"></i></div></td></tr>';
      }).join('') : '<tr><td colspan="3" class="tiny dim" style="padding:18px 16px">Folder kosong.</td></tr>';
    }
    const crumb = byId('an-crumb');
    if (crumb) {
      const rel = home && treePath.startsWith(home + '/') ? treePath.slice(home.length + 1) : '';
      const parts = rel ? rel.split('/') : [];
      let acc = home || treePath;
      let html = '<button data-an-path="' + esc(home || treePath) + '">~</button>';
      for (const p of parts) {
        acc = acc + '/' + p;
        html += '<span class="sl">/</span><button data-an-path="' + esc(acc) + '">' + esc(p) + '</button>';
      }
      html += '<span class="sl">·</span><span class="dim">' + human(tr.bytes) + ' · ' + (tr.files || 0).toLocaleString('id-ID') + ' file</span>';
      crumb.innerHTML = html;
    }
    const map = byId('an-map');
    if (map) {
      const colors = ['a', 'b', 'c', ''];
      map.innerHTML = kids.filter(n => n.dir).slice(0, 4).map((n, i) =>
        '<div class="tm ' + colors[i] + '"' + (n.dir ? ' data-an-path="' + esc(n.path) + '" style="cursor:pointer"' : '') + '>'
        + '<b>' + esc(n.name) + '</b><span>' + human(n.bytes) + ' · ' + Math.round(n.bytes / total * 100) + '%</span></div>'
      ).join('') || '<div class="small dim">Tidak ada subfolder.</div>';
    }
  }

  /* ---------------- analysis: overview ---------------- */
  async function loadOverview() {
    loadingBody('an-large-body', 5, 'Memindai file besar…');
    loadingBody('an-stale-body', 4, 'Memindai file lama…');
    loadingBody('an-art-body', 5, 'Memindai artefak build…');
    try {
      overview = await bridge.AnalysisOverview(home);
      loaded.overview = true;
      renderLarge();
      renderStale();
      renderArtifacts();
    } catch (e) {
      loadingBody('an-large-body', 5, 'Gagal: ' + errMsg(e));
    }
  }

  const cb = id => '<input type="checkbox" class="check" data-an-id="' + esc(id) + '"' + (sel.has(id) ? ' checked' : '') + '>';

  function renderLarge() {
    if (!overview) return;
    const min = Number(byId('an-large-size') && byId('an-large-size').value) || 104857600;
    setDash('an-large-threshold', human(min));
    const rows = (overview.large || []).filter(f => f.bytes >= min);
    const body = byId('an-large-body');
    if (body) body.innerHTML = rows.length ? rows.map(f => {
      const id = 'large:' + f.path;
      return '<tr><td>' + cb(id) + '</td>'
        + '<td class="path mono" title="' + esc(f.path) + '">' + esc(short(f.path)) + '</td>'
        + '<td class="r num">' + human(f.bytes) + '</td>'
        + '<td class="tiny dim">' + esc(f.modTime) + '</td>'
        + '<td class="r"><button class="btn ghost" data-an-open="' + esc(f.path) + '">Buka folder</button></td></tr>';
    }).join('') : '<tr><td colspan="5" class="tiny dim" style="padding:18px 16px">Tidak ada file ≥ ' + human(min) + '.</td></tr>';
    updateSel();
  }

  function renderStale() {
    if (!overview) return;
    const rows = overview.stale || [];
    setDash('an-stale-summary', 'Tidak diubah > 365 hari · ' + rows.length + ' file');
    const body = byId('an-stale-body');
    if (body) body.innerHTML = rows.length ? rows.map(f => {
      const id = 'stale:' + f.path;
      return '<tr><td>' + cb(id) + '</td>'
        + '<td class="path mono" title="' + esc(f.path) + '">' + esc(short(f.path)) + '</td>'
        + '<td class="r num">' + human(f.bytes) + '</td>'
        + '<td class="tiny dim">' + esc(f.modTime) + ' · ' + f.ageDays + ' hari</td></tr>';
    }).join('') : '<tr><td colspan="4" class="tiny dim" style="padding:18px 16px">Tidak ada file lama.</td></tr>';
    updateSel();
  }

  function renderArtifacts() {
    if (!overview) return;
    const rows = overview.artifacts || [];
    const body = byId('an-art-body');
    if (body) body.innerHTML = rows.length ? rows.map(a => {
      const id = 'artifact:' + a.path;
      const dis = a.active || a.protected;
      const chip = a.protected ? '<span class="chip danger">protected</span>'
        : a.active ? '<span class="chip warn">aktif</span>' : '<span class="chip safe">aman</span>';
      return '<tr><td><input type="checkbox" class="check" data-an-id="' + esc(id) + '"' + (sel.has(id) ? ' checked' : '') + (dis ? ' disabled' : '') + '></td>'
        + '<td class="path mono" title="' + esc(a.project) + '">' + esc(short(a.project)) + '</td>'
        + '<td><span class="mono small">' + esc(a.name) + '</span> ' + chip + '</td>'
        + '<td class="r num">' + human(a.bytes) + '</td>'
        + '<td class="tiny dim">' + esc(a.modTime) + ' · ' + a.ageDays + ' hari</td></tr>';
    }).join('') : '<tr><td colspan="5" class="tiny dim" style="padding:18px 16px">Tidak ada artefak.</td></tr>';
    updateSel();
  }

  /* ---------------- analysis: duplicates ---------------- */
  async function loadDups() {
    const el = byId('an-dup-body');
    if (el) el.innerHTML = '<div class="small dim">Membaca isi file untuk mencari duplikat…</div>';
    try {
      dups = await bridge.AnalysisDuplicates(home);
      loaded.dups = true;
      renderDups();
    } catch (e) {
      if (el) el.innerHTML = '<div class="small" style="color:var(--danger)">Gagal: ' + esc(errMsg(e)) + '</div>';
    }
  }

  function renderDups() {
    if (!dups) return;
    const count = dups.groupCount || (dups.groups || []).length;
    const shown = (dups.groups || []).length;
    byId('an-dup-summary').textContent = count + ' grup · ' + human(dups.wasted || 0) + ' bisa dibebaskan'
      + (dups.hardlinks ? ' · ' + dups.hardlinks + ' hardlink dilewati' : '')
      + (shown < count ? ' · menampilkan ' + shown + ' teratas' : '');
    const body = byId('an-dup-body');
    if (!body) return;
    if (!dups.groups || !dups.groups.length) {
      body.innerHTML = '<div class="small dim">Tidak ada file identik ≥ 1 MiB.</div>';
      updateSel();
      return;
    }
    body.innerHTML = dups.groups.map(g => {
      const rows = (g.files || []).map(f => {
        const id = 'dup:' + f.path;
        return '<tr><td>' + (f.keep
          ? '<input type="checkbox" class="check" disabled>'
          : '<input type="checkbox" class="check" data-an-id="' + esc(id) + '"' + (sel.has(id) ? ' checked' : '') + '>') + '</td>'
          + '<td class="path mono" title="' + esc(f.path) + '">' + esc(short(f.path)) + '</td>'
          + '<td class="tiny dim">' + esc(f.modTime) + '</td>'
          + '<td>' + (f.keep ? '<span class="chip safe">simpan</span>' : '<span class="chip warn">duplikat</span>') + '</td></tr>';
      }).join('');
      return '<div class="dupgroup"><div class="duphead"><b>' + human(g.size) + '</b><span class="small dim">' + (g.files || []).length + ' file</span>'
        + '<span class="spacer"></span><span class="tiny dim">bisa dibebaskan ' + human(g.wasted || 0) + '</span></div>'
        + '<table><tbody>' + rows + '</tbody></table></div>';
    }).join('');
    updateSel();
  }

  /* ---------------- selection + action ---------------- */
  function countSel(prefix) { let n = 0; for (const id of sel) if (id.startsWith(prefix + ':')) n++; return n; }

  function updateSel() {
    const set = (id, prefix, label) => {
      const el = byId(id); if (el) el.textContent = countSel(prefix) + ' dipilih';
      const b = byId(id.replace('-sel', '-trash'));
      if (b) b.disabled = countSel(prefix) === 0;
    };
    set('an-large-sel', 'large');
    set('an-dup-sel', 'dup');
    set('an-stale-sel', 'stale');
    set('an-art-sel', 'artifact');
  }

  function selectSafe(prefix, filter) {
    for (const id of sel) if (id.startsWith(prefix + ':')) sel.delete(id);
    const list = prefix === 'stale' ? (overview && overview.stale || [])
      : prefix === 'artifact' ? (overview && overview.artifacts || [])
      : (dups && dups.groups || []).flatMap(g => (g.files || []).filter(f => !f.keep));
    for (const x of list) {
      if (filter && !filter(x)) continue;
      sel.add(prefix + ':' + x.path);
    }
    if (prefix === 'stale') renderStale();
    if (prefix === 'artifact') renderArtifacts();
    if (prefix === 'dup') renderDups();
    updateSel();
  }

  function openAnalysisModal(prefix, label) {
    const ids = [...sel].filter(id => id.startsWith(prefix + ':'));
    if (!ids.length) { toast('Pilih item dulu'); return; }
    const paths = ids.map(id => id.slice(prefix.length + 1));
    const total = paths.length;
    byId('m-title').textContent = 'Pindahkan ' + total + ' item ke Trash?';
    byId('m-sub').textContent = 'Item masuk ke trash freedesktop dan bisa di-restore. Guard path yang sama dengan layar Bersihkan tetap dipakai.';
    byId('m-body').innerHTML = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">' + total + ' item</span><span class="pill">' + esc(label) + '</span><span class="pill">mode: <b>trash</b></span></div>'
      + '<div class="cmdbox">' + paths.slice(0, 6).map(p => esc(short(p))).join('\n') + (total > 6 ? '\n… ' + (total - 6) + ' lainnya' : '') + '</div>'
      + '<div class="tiny dim" style="margin-top:8px">Ukuran yang dibebaskan diukur ulang setelah pemindahan.</div>';
    const old = byId('m-ok');
    const ok = old.cloneNode(true);
    old.replaceWith(ok);
    ok.textContent = 'Ya, pindahkan';
    ok.disabled = false;
    ok.addEventListener('click', async () => {
      ok.disabled = true; ok.textContent = 'Memindahkan…';
      try {
        const res = await bridge.AnalysisClean({ ids, mode: 'trash', dryRun: false });
        showAnalysisResult(res, ids, prefix);
      } catch (e) {
        ok.disabled = false; ok.textContent = 'Ya, pindahkan';
        toast('Gagal: ' + errMsg(e));
      }
    });
    byId('backdrop').classList.add('open');
  }

  function showAnalysisResult(res, ids, prefix) {
    const freed = res.freed || 0;
    byId('m-title').textContent = 'Selesai';
    byId('m-sub').textContent = 'Pemindahan ke trash selesai.';
    const rows = (res.items || []).map(it =>
      '<div class="row" style="gap:8px;padding:5px 0;border-bottom:1px solid var(--line)">'
      + '<span class="chip ' + (it.status === 'removed' ? 'safe' : 'warn') + '">' + esc(it.status) + '</span>'
      + '<span class="mono small" style="overflow:hidden;text-overflow:ellipsis">' + esc(short(it.path)) + '</span>'
      + '<span class="spacer"></span><span class="tiny dim">' + human(it.freed || 0) + '</span></div>'
    ).join('');
    byId('m-body').innerHTML = '<div class="row" style="gap:8px;margin-bottom:10px"><span class="pill">' + ids.length + ' item</span>'
      + '<span class="pill"><b>' + human(freed) + '</b> dibebaskan</span></div>'
      + (rows || '<div class="small dim">Tidak ada item yang berubah.</div>')
      + (res.auditError ? '<div class="small" style="color:var(--danger);margin-top:8px">Audit: ' + esc(res.auditError) + '</div>' : '');
    const old = byId('m-ok');
    const ok = old.cloneNode(true);
    old.replaceWith(ok);
    ok.textContent = 'Tutup'; ok.disabled = false;
    ok.addEventListener('click', () => byId('backdrop').classList.remove('open'));
    for (const id of ids) sel.delete(id);
    updateSel();
    bridge.AnalysisRefresh().then(() => {
      loaded.tree = loaded.overview = loaded.dups = false;
      ensureAnalysis(activeTab);
    }).catch(() => {});
  }

  async function ensureAnalysis(tab) {
    activeTab = tab || activeTab;
    await ensureHome();
    if (activeTab === 'tree' && !loaded.tree) await loadTree(home);
    if ((activeTab === 'large' || activeTab === 'stale' || activeTab === 'artifacts') && !loaded.overview) await loadOverview();
    if (activeTab === 'dup' && !loaded.dups) await loadDups();
  }

  /* ---------------- events ---------------- */
  const rebind = (el, fn) => {
    if (!el) return null;
    const c = el.cloneNode(true);
    el.replaceWith(c);
    c.addEventListener('click', fn);
    return c;
  };

  rebind(byId('dash-refresh'), () => refreshDashboard(true));
  rebind(byId('dash-export'), () => doExport());

  document.querySelectorAll('#tabs button').forEach(b => b.addEventListener('click', () => {
    ensureAnalysis(b.dataset.tab);
  }));

  rebind(byId('an-refresh'), async () => {
    try { await bridge.AnalysisRefresh(); } catch (e) { /* tetap coba muat ulang */ }
    loaded.tree = loaded.overview = loaded.dups = false;
    tree = overview = dups = null;
    sel.clear();
    updateSel();
    ensureAnalysis(activeTab);
  });
  rebind(byId('an-open'), () => {
    if (treePath) bridge.OpenInFileManager(treePath).catch(e => toast('Gagal membuka: ' + errMsg(e)));
  });
  rebind(byId('an-large-trash'), () => openAnalysisModal('large', 'File besar'));
  rebind(byId('an-dup-trash'), () => openAnalysisModal('dup', 'Duplikat'));
  rebind(byId('an-stale-select'), () => selectSafe('stale'));
  rebind(byId('an-stale-trash'), () => openAnalysisModal('stale', 'File lama'));
  rebind(byId('an-art-select'), () => selectSafe('artifact', a => !a.active && !a.protected));
  rebind(byId('an-art-trash'), () => openAnalysisModal('artifact', 'Artefak build'));

  const sizeSel = byId('an-large-size');
  if (sizeSel) sizeSel.addEventListener('change', renderLarge);

  document.addEventListener('change', e => {
    const c = e.target.closest('input[data-an-id]');
    if (!c) return;
    if (c.checked) sel.add(c.dataset.anId); else sel.delete(c.dataset.anId);
    updateSel();
  });

  document.addEventListener('click', e => {
    const row = e.target.closest('[data-an-path]');
    if (row) {
      e.preventDefault();
      const p = row.dataset.anPath;
      if (p && p !== treePath) loadTree(p);
      return;
    }
    const open = e.target.closest('[data-an-open]');
    if (open) {
      const p = open.dataset.anOpen;
      bridge.OpenInFileManager(p).catch(err => toast('Gagal membuka: ' + errMsg(err)));
    }
  });

  /* ---------------- boot ---------------- */
  const baseNavigate = navigate;
  navigate = function (view) {
    baseNavigate(view);
    if (view === 'dashboard') refreshDashboard(false);
    if (view === 'analysis') ensureAnalysis(activeTab);
  };
  if (current === 'dashboard') refreshDashboard(false);
  if (current === 'analysis') ensureAnalysis(activeTab);
})();
