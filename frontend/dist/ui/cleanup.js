/* ================= M2: data nyata + eksekusi (layar Bersihkan) ================= */
(() => {
  const C = window.cleanlie;
  if (!C.bridge) return; // preview browser: biarkan data demo wireframe
  const { bridge, byId, esc, human, shortPath, ensureHome, rebind, setOk } = C;

  const catMap = {
    'dev.node': 'node', 'dev.jvm': 'jvm', 'dev.python': 'python', 'dev.dart': 'dart',
    'dev.rustgo': 'rustgo', 'dev.android': 'android', 'app': 'app', 'sys.log': 'log', 'trash': 'trash'
  };
  const catOrder = ['node', 'jvm', 'python', 'dart', 'rustgo', 'android', 'app', 'log', 'build', 'trash'];
  let report = null, scanPromise = null, scanning = false, cleaning = false;

  async function ensureScan(force) {
    if (scanPromise) return scanPromise;
    if (report && !force) return report;
    scanPromise = (async () => {
      scanning = true;
      setScanUI(true);
      try {
        await ensureHome();
        const r = await bridge.ScanJunk();
        applyReport(r);
        return r;
      } catch (e) {
        const msg = typeof e === 'string' ? e : (e && e.message) || String(e);
        if (!/context canceled|canceled/i.test(msg)) toast('Scan gagal: ' + msg);
        return null;
      } finally {
        scanning = false;
        setScanUI(false);
      }
    })().finally(() => { scanPromise = null; });
    return scanPromise;
  }

  function applyReport(r) {
    report = r;
    ITEMS.length = 0;
    for (const g of r.groups) {
      const cat = catMap[g.id] || 'app';
      for (const it of g.items) {
        const rootOnly = !!it.requiresRoot;
        ITEMS.push({
          id: it.id, g: cat, p: shortPath(it.path), full: it.path,
          mb: it.bytes / 1048576, bytes: it.bytes, risk: rootOnly ? 'blocked' : it.risk,
          note: rootOnly ? 'butuh aksi root (M3)' : (it.commandLine ? '\u2192 ' + it.commandLine : (it.note || '')),
          cmd: it.commandLine || '', requiresRoot: rootOnly
        });
      }
    }
    selected.clear();
    const autoSafe = !window.__getSettings || window.__getSettings().autoSelectSafe;
    if (autoSafe) for (const it of ITEMS) if (it.risk === 'safe') selected.add(it.p);

    refreshCats();
    const visible = catOrder.filter(c => ITEMS.some(i => i.g === c));
    if (!visible.includes(activeCat)) activeCat = visible[0] || 'node';
    document.querySelectorAll('.view button[data-cat]').forEach(x => x.classList.toggle('active', x.dataset.cat === activeCat));
    renderCleanup();
  }

  function refreshCats() {
    let warn = 0, locked = 0;
    for (const btn of document.querySelectorAll('.cat')) {
      const g = btn.dataset.cat;
      const items = ITEMS.filter(i => i.g === g);
      const mb = items.reduce((a, b) => a + b.mb, 0);
      const sz = btn.querySelector('.sz');
      if (sz) sz.textContent = mb >= 1024 ? (mb / 1024).toFixed(1) + ' G' : Math.round(mb) + ' M';
      btn.style.display = items.length ? '' : 'none';
      warn += items.filter(i => i.risk === 'warn' || i.risk === 'danger').length;
      locked += items.filter(i => i.risk === 'blocked').length;
    }
    const foot = byId('cat-foot-text');
    if (foot) foot.textContent = warn + ' item butuh konfirmasi' + (locked ? ' \u00b7 ' + locked + ' terkunci' : '');
    const hint = document.querySelector('.actionbar .hint');
    if (hint) hint.textContent = warn + ' item \u26a0 \u00b7 ' + locked + ' item \u26d4 terkunci';
    const safeMb = ITEMS.filter(i => i.risk === 'safe').reduce((a, b) => a + b.mb, 0);
    const qs = byId('quick-size');
    if (qs) qs.textContent = (safeMb / 1024).toFixed(1) + ' GB';
  }

  function setScanUI(on) {
    const c = byId('scan-cancel');
    if (c) { c.hidden = !on; c.disabled = false; c.textContent = 'Batal'; }
    const sa = byId('select-all');
    if (sa) sa.disabled = on;
    for (const id of ['btn-trash', 'btn-permanent']) {
      const b = byId(id);
      if (b) b.disabled = on || scanning || cleaning;
    }
    if (on) {
      const sub = byId('cat-sub');
      if (sub) sub.textContent = 'Memindai katalog\u2026';
    }
  }

  function selectedItems() {
    return ITEMS.filter(i => selected.has(i.p) && i.risk !== 'blocked');
  }

  function openClean(mode) {
    if (cleaning) return;
    const items = selectedItems();
    if (!items.length) { toast('Pilih minimal satu item'); return; }
    const total = items.reduce((a, b) => a + b.bytes, 0);
    const cmds = [...new Set(items.map(i => i.cmd).filter(Boolean))];
    const permanent = mode === 'permanent';

    byId('m-title').textContent = permanent
      ? 'Hapus permanen ' + items.length + ' item?'
      : 'Pindahkan ' + items.length + ' item ke Trash?';
    byId('m-sub').textContent = permanent
      ? 'Tindakan ini tidak bisa dibatalkan dan tidak masuk trash.'
      : 'Item masuk ke trash freedesktop dan bisa di-restore dari file manager.';

    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">' + items.length + ' item</span>'
      + '<span class="pill"><b>' + human(total) + '</b> perkiraan</span>'
      + '<span class="pill">mode: <b>' + mode + '</b></span></div>';

    if (cmds.length) {
      body += '<div class="cmdbox">' + cmds.slice(0, 6).map(esc).join('\n') + (cmds.length > 6 ? '\n\u2026' : '') + '</div>';
    }
    body += '<div class="tiny dim" style="margin-top:8px">' + items.slice(0, 6).map(i => esc(i.p)).join(' \u00b7 ')
      + (items.length > 6 ? ' \u00b7 +' + (items.length - 6) + ' lainnya' : '') + '</div>';
    body += '<div class="tiny dim" style="margin-top:8px">Ukuran akhir diukur ulang setelah eksekusi (before \u2212 after), bukan diasumsikan.</div>';

    if (permanent) {
      body += '<div class="field" style="border:0;padding:12px 0 0"><div class="label"><b>Saya mengerti tidak bisa di-undo</b>'
        + '<span>Wajib dicentang untuk melanjutkan</span></div><div class="ctrl"><input type="checkbox" id="perm-confirm"></div></div>';
    }
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = 'Batal';

    const ok = setOk(permanent ? 'Hapus permanen' : 'Ya, pindahkan', permanent, () => doClean(items, mode));
    if (permanent) {
      const cb = byId('perm-confirm');
      cb.addEventListener('change', () => { ok.disabled = !cb.checked; });
    }
    backdrop.classList.add('open');
  }

  async function doClean(items, mode) {
    cleaning = true;
    const ids = items.map(i => i.id);
    setOk('Memproses\u2026', true);
    setScanUI(false);
    try {
      const res = await bridge.Clean({ ids, mode, confirmPermanent: mode === 'permanent', dryRun: false });
      showResult(res, mode);
      report = null;
      ensureScan(true);
    } catch (e) {
      const msg = typeof e === 'string' ? e : (e && e.message) || String(e);
      byId('m-title').textContent = 'Pembersihan gagal';
      byId('m-body').innerHTML = '<div class="cmdbox">' + esc(msg) + '</div>';
      setOk('Tutup', false, () => backdrop.classList.remove('open'));
    } finally {
      cleaning = false;
      setScanUI(false);
    }
  }

  function showResult(res, mode) {
    const removed = res.items.filter(i => i.status === "removed");
    const failed = res.items.filter(i => i.status === "failed" || i.status === "skipped" || i.status === "kept");
    byId('m-title').textContent = mode === 'permanent' ? 'Hasil hapus permanen' : 'Hasil pemindahan ke Trash';
    byId('m-sub').textContent = removed.length + ' dari ' + res.items.length + ' item berhasil';

    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill safe"><b>' + human(res.freed) + '</b> dibebaskan</span>'
      + '<span class="pill">' + removed.length + ' berhasil</span>'
      + '<span class="pill">' + failed.length + ' gagal/skip</span></div>';

    if (res.auditError) {
      body += '<div class="tiny" style="color:var(--warn);margin-top:8px">Audit gagal ditulis: ' + esc(res.auditError) + '</div>';
    }


    if (failed.length) {
      body += '<div class="tiny dim" style="margin:8px 0 4px">Item yang tidak selesai:</div>'
        + '<div class="cmdbox">' + failed.slice(0, 8).map(i => esc(shortPath(i.path)) + ' \u2014 ' + esc(i.reason || i.status)).join('\n')
        + (failed.length > 8 ? '\n\u2026' : '') + '</div>';
    }
    body += '<div class="row" style="gap:8px;margin-top:12px">'
      + '<button class="btn ghost" id="res-trash">Buka trash</button>'
      + '<button class="btn ghost" id="res-copy">Salin log</button>'
      + (mode === 'trash' && res.batch ? '<button class="btn ghost" id="res-undo">Undo sesi ini</button>' : '')
      + '</div>'
      + '<div class="tiny dim mono" style="margin-top:10px">' + esc(shortPath(res.auditPath)) + '</div>';
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = 'Tutup';
    setOk('Scan ulang & tutup', false, () => {
      backdrop.classList.remove('open');
      ensureScan(true);
    });

    const log = JSON.stringify(res, null, 2);
    byId('res-copy').addEventListener('click', async () => {
      try { await bridge.CopyText(log); toast('Log disalin'); } catch (e) { toast('Gagal menyalin: ' + e); }
    });
    byId('res-trash').addEventListener('click', async () => {
      try { await bridge.OpenTrash(); } catch (e) { toast('Gagal membuka trash: ' + e); }
    });
    const undo = byId('res-undo');
    if (undo) undo.addEventListener('click', async () => {
      undo.disabled = true;
      undo.textContent = 'Mengembalikan…';
      try {
        const ur = await bridge.UndoSession(res.batch, false);
        toast((ur.restored || 0) + ' item dipulihkan, ' + (ur.skipped || 0) + ' dilewati');
        backdrop.classList.remove('open');
        report = null;
        ensureScan(true);
      } catch (e) {
        undo.disabled = false;
        undo.textContent = 'Undo sesi ini';
        toast('Undo gagal: ' + (typeof e === 'string' ? e : (e && e.message) || String(e)));
      }
    });
  }

  /* tombol aksi: lepas listener simulasi wireframe lalu pasang yang nyata */
  rebind(byId('btn-trash'), () => openClean('trash'));
  rebind(byId('btn-permanent'), () => openClean('permanent'));
  rebind(byId('quick-clean'), async () => {
    navigate("cleanup");
    const r = await ensureScan();
    if (r) {
      const mode = window.__getSettings ? window.__getSettings().deleteMode : 'trash';
      openClean(mode === 'permanent' ? 'permanent' : 'trash');
    }
  });

  const cancelBtn = byId('scan-cancel');
  if (cancelBtn) cancelBtn.addEventListener('click', () => {
    cancelBtn.textContent = 'Membatalkan\u2026';
    cancelBtn.disabled = true;
    bridge.CancelScan();
  });

  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn('scan:progress', p => {
      if (!scanning) return;
      const sub = byId('cat-sub');
      if (sub) sub.textContent = 'Memindai ' + p.done + '/' + p.total + ' \u00b7 ' + shortPath(p.path);
    });
    window.runtime.EventsOn('clean:progress', p => {
      if (!cleaning) return;
      byId('m-sub').textContent = 'Memproses ' + p.done + '/' + p.total + ' \u00b7 ' + shortPath(p.path);
    });
    window.runtime.EventsOn('scan:done', () => { /* ringkasan sudah dari ScanJunk */ });
  }

  C.ensureScan = ensureScan;
  C.report = () => report;

  const rowsEl = byId('rows');
  if (rowsEl) rowsEl.addEventListener('click', e => {
    const b = e.target.closest('[data-preview-id]');
    if (!b || !b.dataset.previewId) return;
    e.preventDefault();
    if (C.previewCleanup) C.previewCleanup(b.dataset.previewId);
  });
})();
