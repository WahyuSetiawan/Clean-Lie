/* ================= M7: direnv ================= */
(() => {
  const C = window.cleanlie;
  if (!C || !C.bridge) return; // preview browser: biarkan wireframe statis
  const { bridge, byId, esc, human, shortPath, ensureHome, rebind, setOk, errText } = C;
  const sel = new Set();
  let roots = [];
  let report = null;
  let busy = false;

  const setStatus = text => {
    const el = byId('direnv-status');
    if (el) el.textContent = text || '';
  };

  function syncSettingsRoots() {
    const st = window.__getSettings && window.__getSettings();
    if (st) st.direnvRoots = roots.slice();
  }

  function renderRoots() {
    const box = byId('direnv-roots');
    if (!box) return;
    box.innerHTML = roots.length
      ? roots.map((p, i) =>
          '<span class="tag"><span class="mono">' + esc(shortPath(p)) + '</span>' +
          '<button type="button" data-dr-rm="' + i + '" aria-label="Hapus ' + esc(p) + '">×</button></span>'
        ).join('')
      : '<span class="tiny dim">Belum ada folder. Tambahkan minimal satu folder proyek.</span>';
    const scan = byId('direnv-scan');
    if (scan) scan.disabled = busy || roots.length === 0;
  }

  function updateSel() {
    const el = byId('direnv-selected');
    if (el) el.textContent = sel.size + ' dipilih';
    const trash = byId('direnv-trash');
    if (trash) trash.disabled = busy || sel.size === 0;
    const nav = byId('nav-direnv-count');
    if (nav) nav.textContent = report ? String((report.entries || []).length) : '—';
  }

  function setBusy(on) {
    busy = on;
    const scan = byId('direnv-scan');
    if (scan) scan.disabled = on || roots.length === 0;
    const cancel = byId('direnv-cancel');
    if (cancel) {
      cancel.hidden = !on;
      if (on) { cancel.disabled = false; cancel.textContent = 'Batal'; }
    }
    const add = byId('direnv-root-add');
    if (add) add.disabled = on;
    const pickBtn = byId('direnv-root-pick');
    if (pickBtn) pickBtn.disabled = on;
    const inp = byId('direnv-root-input');
    if (inp) inp.disabled = on;
    const refresh = byId('direnv-refresh');
    if (refresh) refresh.disabled = on;
    const pick = byId('direnv-select');
    if (pick) pick.disabled = on;
    updateSel();
  }

  function renderReport() {
    const body = byId('direnv-body');
    const total = byId('direnv-total');
    const summary = byId('direnv-summary');
    const missing = byId('direnv-missing');
    if (!report) {
      if (total) total.textContent = '—';
      if (summary) summary.textContent = roots.length
        ? 'Folder siap dipindai.'
        : 'Belum ada folder. Tambahkan folder proyek dulu.';
      if (missing) missing.hidden = true;
      if (body) body.innerHTML = '<tr><td colspan="6" class="tiny dim" style="padding:18px 16px">Tambahkan folder lalu jalankan scan.</td></tr>';
      updateSel();
      return;
    }
    const entries = report.entries || [];
    if (total) total.textContent = human(report.bytes || 0);
    if (summary) {
      const projects = entries.filter(e => e.kind === 'project').length;
      const caches = entries.filter(e => e.kind === 'cache').length;
      summary.textContent = projects + ' proyek · ' + caches + ' cache global · ' +
        (report.files || 0).toLocaleString('id-ID') + ' file · ' + (report.durationMs || 0) + ' ms';
    }
    if (missing) {
      const list = report.missing || [];
      missing.hidden = list.length === 0;
      missing.textContent = list.length ? 'Folder tidak ditemukan: ' + list.map(shortPath).join(', ') : '';
    }
    if (body) {
      body.innerHTML = entries.length ? entries.map(e => {
        const id = e.id;
        const chip = e.kind === 'cache'
          ? '<span class="chip safe">cache</span>'
          : e.active ? '<span class="chip warn">aktif</span>' : '<span class="chip safe">aman</span>';
        const project = e.kind === 'cache' ? 'Cache global' : shortPath(e.project);
        const loc = e.kind === 'cache' ? shortPath(e.path) : shortPath(e.project) + '/.direnv';
        return '<tr><td><input type="checkbox" class="check" data-dr-id="' + esc(id) + '"' +
          (sel.has(id) ? ' checked' : '') + '></td>'
          + '<td class="path mono" title="' + esc(e.project) + '">' + esc(project) + '</td>'
          + '<td class="path mono" title="' + esc(e.path) + '">' + esc(loc) + '</td>'
          + '<td class="r num">' + human(e.bytes || 0) + '</td>'
          + '<td class="tiny dim">' + esc(e.modTime || '') + ' · ' + (e.ageDays || 0) + ' hari</td>'
          + '<td>' + chip + '</td></tr>';
      }).join('') : '<tr><td colspan="6" class="tiny dim" style="padding:18px 16px">Tidak ada folder .direnv di root tersebut.</td></tr>';
    }
    updateSel();
  }

  async function loadRoots() {
    let next = [];
    try {
      const view = await bridge.Settings();
      next = (view && view.config && view.config.direnvRoots) || [];
    } catch (e) {
      next = roots.slice();
    }
    const changed = JSON.stringify(next) !== JSON.stringify(roots);
    roots = next;
    syncSettingsRoots();
    renderRoots();
    if (changed) {
      report = null;
      sel.clear();
      renderReport();
    }
    return changed;
  }

  async function addRoot() {
    const input = byId('direnv-root-input');
    if (!input) return;
    const p = input.value.trim();
    if (!p) return;
    try {
      const view = await bridge.DirenvAddRoot(p);
      roots = (view && view.config && view.config.direnvRoots) || roots;
      input.value = '';
      report = null;
      sel.clear();
      syncSettingsRoots();
      renderRoots();
      renderReport();
      setStatus('Folder ditambahkan.');
    } catch (e) {
      toast('Gagal menambah folder: ' + errText(e));
    }
  }

  async function pickRoot() {
    const input = byId('direnv-root-input');
    try {
      const p = await bridge.DirenvPickRoot();
      if (!p) return;
      if (input) input.value = p;
      await addRoot();
    } catch (e) {
      toast('Gagal memilih folder: ' + errText(e));
    }
  }

  async function removeRoot(index) {
    const p = roots[index];
    if (!p) return;
    try {
      const view = await bridge.DirenvRemoveRoot(p);
      roots = (view && view.config && view.config.direnvRoots) || [];
      report = null;
      sel.clear();
      syncSettingsRoots();
      renderRoots();
      renderReport();
      setStatus('Folder dihapus.');
    } catch (e) {
      toast('Gagal menghapus folder: ' + errText(e));
    }
  }

  async function scan(force) {
    if (busy) return;
    if (!roots.length) {
      toast('Tambahkan folder proyek dulu');
      return;
    }
    if (report && !force) return;
    setBusy(true);
    setStatus('Memindai ' + roots.length + ' folder…');
    try {
      await ensureHome();
      if (force) {
        try { await bridge.DirenvRefresh(); } catch (e) { /* tetap coba scan */ }
      }
      report = await bridge.DirenvScan();
      sel.clear();
      renderReport();
      const missing = (report.missing || []).length;
      setStatus('Selesai · ' + (report.durationMs || 0) + ' ms' + (missing ? ' · ' + missing + ' folder tidak ada' : ''));
    } catch (e) {
      setStatus('Gagal: ' + errText(e));
    } finally {
      setBusy(false);
    }
  }

  function selectSafe() {
    if (!report) { toast('Scan dulu'); return; }
    sel.clear();
    for (const e of report.entries || []) {
      if (!e.active) sel.add(e.id);
    }
    if (!sel.size) toast('Semua .direnv masih aktif; pilih manual bila tetap ingin dihapus');
    renderReport();
  }

  function openTrash() {
    const ids = [...sel];
    if (!ids.length) { toast('Pilih item dulu'); return; }
    const map = new Map((report && report.entries || []).map(e => [e.id, e]));
    const picked = ids.map(id => map.get(id)).filter(Boolean);
    const total = picked.reduce((a, e) => a + (e.bytes || 0), 0);
    byId('m-title').textContent = 'Pindahkan ' + ids.length + ' folder .direnv ke Trash?';
    byId('m-sub').textContent = 'Folder .direnv masuk ke trash freedesktop dan bisa di-restore. direnv/nix-direnv akan membuatnya ulang saat proyek dipakai (mungkin perlu direnv allow).';
    byId('m-body').innerHTML = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">' + ids.length + ' item</span>'
      + '<span class="pill"><b>' + human(total) + '</b> perkiraan</span>'
      + '<span class="pill">mode: <b>trash</b></span></div>'
      + '<div class="cmdbox">' + picked.slice(0, 6).map(e => esc(shortPath(e.path))).join('\n')
      + (picked.length > 6 ? '\n… +' + (picked.length - 6) + ' lainnya' : '') + '</div>'
      + '<div class="tiny dim" style="margin-top:8px">Ukuran yang dibebaskan diukur ulang setelah pemindahan.</div>';
    setOk('Ya, pindahkan', false, async () => {
      setOk('Memindahkan…', true);
      try {
        const res = await bridge.DirenvClean({ ids, mode: 'trash', dryRun: false });
        showResult(res);
      } catch (e) {
        setOk('Ya, pindahkan', false, openTrash);
        toast('Gagal: ' + errText(e));
      }
    });
    byId('backdrop').classList.add('open');
  }

  function showResult(res) {
    const freed = res.freed || 0;
    byId('m-title').textContent = 'Selesai';
    byId('m-sub').textContent = 'Pemindahan folder .direnv ke trash selesai.';
    const rows = (res.items || []).map(it =>
      '<div class="row" style="gap:8px;padding:5px 0;border-bottom:1px solid var(--line)">'
      + '<span class="chip ' + (it.status === 'removed' ? 'safe' : 'warn') + '">' + esc(it.status) + '</span>'
      + '<span class="mono small" style="overflow:hidden;text-overflow:ellipsis">' + esc(shortPath(it.path)) + '</span>'
      + '<span class="spacer"></span><span class="tiny dim">' + human(it.freed || 0) + '</span></div>'
    ).join('');
    byId('m-body').innerHTML = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill"><b>' + human(freed) + '</b> dibebaskan</span>'
      + '<span class="pill">' + (res.items || []).length + ' item</span></div>'
      + (rows || '<div class="small dim">Tidak ada item yang berubah.</div>')
      + (res.auditError ? '<div class="small" style="color:var(--danger);margin-top:8px">Audit: ' + esc(res.auditError) + '</div>' : '');
    setOk('Tutup', false, () => {
      byId('backdrop').classList.remove('open');
      report = null;
      sel.clear();
      renderReport();
      scan(true);
    });
  }

  async function ensureDirenv() {
    await loadRoots();
    if (roots.length && !report && !busy) await scan(false);
  }

  /* ---------------- events ---------------- */
  rebind(byId('direnv-root-add'), addRoot);
  rebind(byId('direnv-root-pick'), pickRoot);
  rebind(byId('direnv-scan'), () => scan(false));
  rebind(byId('direnv-refresh'), () => scan(true));
  rebind(byId('direnv-select'), selectSafe);
  rebind(byId('direnv-trash'), openTrash);

  const input = byId('direnv-root-input');
  if (input) input.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); addRoot(); }
  });
  const rootBox = byId('direnv-roots');
  if (rootBox) rootBox.addEventListener('click', e => {
    const btn = e.target.closest('[data-dr-rm]');
    if (btn) removeRoot(Number(btn.dataset.drRm));
  });
  const cancel = byId('direnv-cancel');
  if (cancel) cancel.addEventListener('click', async () => {
    cancel.disabled = true;
    cancel.textContent = 'Membatalkan…';
    try { await bridge.DirenvCancel(); } catch (e) { /* abaikan */ }
  });
  document.addEventListener('change', e => {
    const c = e.target.closest('input[data-dr-id]');
    if (!c) return;
    if (c.checked) sel.add(c.dataset.drId); else sel.delete(c.dataset.drId);
    updateSel();
  });

  /* ---------------- boot ---------------- */
  const baseNavigate = navigate;
  navigate = function (view) {
    baseNavigate(view);
    if (view === 'direnv') ensureDirenv();
  };
  renderRoots();
  renderReport();
  if (current === 'direnv') ensureDirenv();
})();
