/* ================= M8: Riwayat — aktivitas & trash ================= */
(() => {
  const C = window.cleanlie;
  const { bridge, byId, esc, human, shortPath, setOk, errText } = C;
  const t = (k, fallback) => (window.__t && window.__t(k)) || fallback;

  const demoTrash = [
    { id: 'demo-a', name: 'npm-cache', originalPath: '~/.npm/_cacache', deletedAt: '2026-11-25T10:12:00Z', bytes: 734003200, restorable: true, reason: '' },
    { id: 'demo-b', name: 'old-target', originalPath: '~/Projects/app/target', deletedAt: '2026-11-24T18:30:00Z', bytes: 1258291200, restorable: false, reason: 'tujuan restore sudah ada' },
  ];
  const demoSessions = [{
    id: 'demo-session', batch: 'demo-session', action: '', mode: 'trash', startedAt: '2026-11-25T10:12:00Z',
    removed: 2, failed: 0, restorable: 1, freed: 0, restorableBytes: 734003200,
    items: [
      { path: '~/.npm/_cacache', status: 'removed', trashPath: 'demo-a' },
      { path: '~/Projects/app/target', status: 'removed', trashPath: 'demo-b' },
    ],
  }];

  let sessions = [], trash = [], loadedSessions = false, loadedTrash = false;
  let active = 'activity', search = '', busy = false;
  const selected = new Set();

  const close = () => byId('backdrop').classList.remove('open');
  const fmtTime = s => {
    if (!s) return '—';
    const d = new Date(s);
    if (isNaN(d.getTime())) return s;
    return d.toLocaleString(document.documentElement.lang === 'en' ? 'en-GB' : 'id-ID', { dateStyle: 'short', timeStyle: 'short' });
  };
  const actionLabel = s => s.action === 'restore' ? 'restore' : s.action === 'trash-delete' ? 'hapus permanen' : s.action === 'trash-empty' ? 'kosongkan' : (s.mode || 'clean');
  const chip = s => s.action === 'restore' ? 'safe' : s.action ? 'warn' : (s.failed ? 'warn' : 'safe');

  function note(msg) { const el = byId('history-note'); if (el) el.textContent = msg || ''; }

  async function loadSessions() {
    if (!bridge) { sessions = demoSessions; loadedSessions = true; renderSessions(); return; }
    note('Memuat sesi…');
    try {
      const page = await bridge.HistorySessions(50);
      sessions = (page && page.sessions) || [];
      loadedSessions = true;
      renderSessions(page && page.truncated);
    } catch (e) {
      note('Gagal memuat riwayat: ' + errText(e));
    }
  }

  async function loadTrash() {
    if (!bridge) { trash = demoTrash; loadedTrash = true; renderTrash(); return; }
    note('Memuat trash…');
    try {
      await bridge.TrashRefresh();
      const rep = await bridge.TrashList();
      trash = (rep && rep.entries) || [];
      loadedTrash = true;
      renderTrash(rep && rep.truncated);
    } catch (e) {
      note('Gagal memuat trash: ' + errText(e));
    }
  }

  function renderSessions(truncated) {
    const body = byId('history-sessions');
    if (!body) return;
    const count = byId('history-session-count');
    if (count) count.textContent = sessions.length + ' sesi';
    if (byId('nav-history-count')) byId('nav-history-count').textContent = sessions.length;
    if (!sessions.length) {
      body.innerHTML = '<tr><td colspan="6" class="tiny dim" style="padding:18px 16px">' + esc(t('history.noSessions', 'Belum ada aktivitas pembersihan.')) + '</td></tr>';
      return;
    }
    body.innerHTML = sessions.map(s => {
      const items = s.items || [];
      const detail = items.slice(0, 8).map(it =>
        '<div class="row" style="gap:8px;padding:3px 0"><span class="chip ' + (it.status === 'removed' ? 'safe' : 'warn') + '">' + esc(it.status || '?') + '</span>'
        + '<span class="mono small" style="overflow:hidden;text-overflow:ellipsis">' + esc(shortPath(it.path || '—')) + '</span></div>'
      ).join('') + (items.length > 8 ? '<div class="tiny dim">… +' + (items.length - 8) + ' item</div>' : '');
      return '<tr>'
        + '<td class="tiny">' + esc(fmtTime(s.startedAt)) + '</td>'
        + '<td><span class="chip ' + chip(s) + '">' + esc(actionLabel(s)) + '</span><div class="tiny dim">' + esc(s.batch || s.id) + '</div></td>'
        + '<td class="r num">' + (s.removed || 0) + (s.failed ? ' <span class="tiny" style="color:var(--warn)">+' + s.failed + ' gagal</span>' : '') + '</td>'
        + '<td class="r num">' + human(s.freed || 0) + '</td>'
        + '<td class="r num">' + (s.restorable || 0) + (s.restorableBytes ? '<div class="tiny dim">' + human(s.restorableBytes) + '</div>' : '') + '</td>'
        + '<td class="r"><button class="btn ghost" data-undo="' + esc(s.id) + '"' + (s.restorable ? '' : ' disabled') + '>' + esc(t('history.undo', 'Undo sesi ini')) + '</button></td>'
        + '</tr><tr><td colspan="6" style="padding-top:0"><details><summary class="tiny dim">' + items.length + ' item' + (s.itemsTruncated ? ' (dipotong)' : '') + '</summary>' + detail + '</details></td></tr>';
    }).join('');
    if (truncated) note(t('history.truncated', 'Daftar dipotong; gunakan pencarian untuk mempersempit.'));
  }

  function renderTrash(truncated) {
    const body = byId('history-trash-body');
    if (!body) return;
    const q = search.trim().toLowerCase();
    const list = trash.filter(e => !q || (e.name || '').toLowerCase().includes(q) || (e.originalPath || '').toLowerCase().includes(q));
    if (byId('nav-history-count')) byId('nav-history-count').textContent = trash.length;
    for (const id of [...selected]) if (!trash.some(e => e.id === id)) selected.delete(id);

    if (!list.length) {
      body.innerHTML = '<tr><td colspan="7" class="tiny dim" style="padding:18px 16px">' + esc(trash.length ? 'Tidak ada hasil pencarian.' : t('history.noTrash', 'Trash kosong.')) + '</td></tr>';
      updateSel();
      return;
    }
    body.innerHTML = list.map(e => {
      const checked = selected.has(e.id) ? 'checked' : '';
      const size = e.bytes < 0 ? '…' : human(e.bytes);
      return '<tr>'
        + '<td><input type="checkbox" data-trash-id="' + esc(e.id) + '" ' + checked + ' aria-label="Pilih ' + esc(e.name) + '"></td>'
        + '<td><b>' + esc(e.name) + '</b>' + (e.rootKind === 'volume' ? ' <span class="tiny dim">volume</span>' : '') + '</td>'
        + '<td class="mono small" style="overflow:hidden;text-overflow:ellipsis">' + esc(e.originalPath ? shortPath(e.originalPath) : '—') + '</td>'
        + '<td class="tiny">' + esc(fmtTime(e.deletedAt)) + '</td>'
        + '<td class="r num">' + size + '</td>'
        + '<td>' + (e.restorable ? '<span class="chip safe">Pulihkan</span>' : '<span class="chip warn">' + esc(t('history.notRestorable', 'tidak bisa dipulihkan')) + '</span><div class="tiny dim">' + esc(e.reason || '') + '</div>') + '</td>'
        + '<td class="r"><button class="btn ghost" data-preview="' + esc(e.id) + '">' + esc(t('history.preview', 'Lihat isi')) + '</button> '
        + '<button class="btn ghost" data-restore="' + esc(e.id) + '"' + (e.restorable ? '' : ' disabled') + '>' + esc(t('history.restore', 'Pulihkan')) + '</button></td>'
        + '</tr>';
    }).join('');
    if (truncated) note(t('history.truncated', 'Daftar dipotong; gunakan pencarian untuk mempersempit.'));
    updateSel();
  }

  function updateSel() {
    const n = selected.size;
    if (byId('history-selected')) byId('history-selected').textContent = n + ' ' + t('history.selected', 'dipilih');
    const restore = byId('history-restore');
    const del = byId('history-delete');
    if (restore) restore.disabled = busy || n === 0;
    if (del) del.disabled = busy || n === 0;
    const all = byId('history-check-all');
    if (all) {
      const ids = [...document.querySelectorAll('#history-trash-body [data-trash-id]')].map(x => x.dataset.trashId);
      all.checked = ids.length > 0 && ids.every(id => selected.has(id));
      all.indeterminate = !all.checked && ids.some(id => selected.has(id));
    }
  }

  function showResult(res, title) {
    const r = res || {};
    const items = r.items || [];
    const failed = items.filter(i => i.status === 'failed' || i.status === 'skipped');
    byId('m-title').textContent = title || t('history.result', 'Hasil');
    byId('m-sub').textContent = (r.restored || r.deleted || 0) + ' item selesai';
    let body = '<div class="row" style="gap:8px;flex-wrap:wrap;margin-bottom:10px">'
      + (r.restored ? '<span class="pill safe"><b>' + r.restored + '</b> dipulihkan</span>' : '')
      + (r.deleted ? '<span class="pill safe"><b>' + r.deleted + '</b> dihapus</span>' : '')
      + (r.skipped ? '<span class="pill">' + r.skipped + ' dilewati</span>' : '')
      + (r.failed ? '<span class="pill warn">' + r.failed + ' gagal</span>' : '')
      + (r.bytes ? '<span class="pill"><b>' + human(r.bytes) + '</b></span>' : '')
      + '</div>';
    if (r.auditError) body += '<div class="small" style="color:var(--warn);margin:8px 0">Audit: ' + esc(r.auditError) + '</div>';
    if (failed.length) body += '<div class="cmdbox">' + failed.slice(0, 8).map(i => esc(shortPath(i.originalPath || i.name || '')) + ' — ' + esc(i.reason || i.status)).join('\n') + '</div>';
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = t('history.close', 'Tutup');
    setOk(t('history.close', 'Tutup'), false, () => { close(); refreshAfterMutation(); });
    byId('backdrop').classList.add('open');
  }

  function refreshAfterMutation() {
    selected.clear();
    loadedSessions = false;
    loadedTrash = false;
    if (current === 'history') ensureHistory(true);
  }

  async function doRestore(ids) {
    if (!ids.length || busy) return;
    busy = true; updateSel();
    try {
      const res = await bridge.TrashRestore(ids, false);
      showResult(res, 'Hasil restore');
    } catch (e) {
      showResult({ failed: 1, items: [{ status: 'failed', reason: errText(e) }] }, 'Restore gagal');
    } finally { busy = false; updateSel(); }
  }

  function confirmRestore(ids) {
    if (!ids.length) return;
    const n = ids.length;
    byId('m-title').textContent = 'Pulihkan ' + n + ' item?';
    byId('m-sub').textContent = 'Item yang tidak bisa dipulihkan akan dilewati; tidak ada file yang ditimpa.';
    byId('m-body').innerHTML = '<div class="small">Item akan kembali ke path asal dari .trashinfo. Konflik tidak di-rename otomatis.</div>';
    byId('m-cancel').textContent = 'Batal';
    setOk(t('history.restoreOk', 'Pulihkan'), false, () => { close(); doRestore(ids); });
    byId('backdrop').classList.add('open');
  }

  function confirmDelete(ids) {
    if (!ids.length) return;
    byId('m-title').textContent = 'Hapus permanen ' + ids.length + ' item?';
    byId('m-sub').textContent = t('history.deleteConfirm', 'Hapus permanen item terpilih? Tindakan ini tidak bisa dibatalkan.');
    byId('m-body').innerHTML = '<div class="field" style="border:0;padding:0"><div class="label"><b>Saya mengerti tidak bisa di-undo</b><span>Wajib dicentang untuk melanjutkan</span></div><div class="ctrl"><input type="checkbox" id="history-delete-confirm"></div></div>';
    byId('m-cancel').textContent = 'Batal';
    const ok = setOk(t('history.deleteOk', 'Hapus permanen'), true, async () => {
      close();
      busy = true; updateSel();
      try {
        const res = await bridge.TrashDelete(ids, true, false);
        showResult(res, 'Hasil hapus permanen');
      } catch (e) {
        showResult({ failed: 1, items: [{ status: 'failed', reason: errText(e) }] }, 'Hapus gagal');
      } finally { busy = false; updateSel(); }
    });
    const cb = byId('history-delete-confirm');
    cb.addEventListener('change', () => { ok.disabled = !cb.checked; });
    byId('backdrop').classList.add('open');
  }

  function confirmEmpty() {
    byId('m-title').textContent = t('history.empty', 'Kosongkan trash?');
    byId('m-sub').textContent = t('history.emptyConfirm', 'Ketik KOSONGKAN atau EMPTY untuk mengosongkan seluruh trash.');
    byId('m-body').innerHTML = '<div class="field" style="border:0;padding:0"><div class="label"><b>Ketik KOSONGKAN atau EMPTY</b><span>Semua isi trash dihapus permanen</span></div><div class="ctrl"><input type="text" id="history-empty-input" autocomplete="off" style="width:180px"></div></div>';
    byId('m-cancel').textContent = 'Batal';
    const ok = setOk(t('history.emptyOk', 'Kosongkan'), true, async () => {
      const val = byId('history-empty-input').value.trim();
      close();
      busy = true; updateSel();
      try {
        const res = await bridge.TrashEmpty(val, false);
        showResult(res, 'Hasil kosongkan trash');
      } catch (e) {
        showResult({ failed: 1, items: [{ status: 'failed', reason: errText(e) }] }, 'Kosongkan gagal');
      } finally { busy = false; updateSel(); }
    });
    const inp = byId('history-empty-input');
    inp.addEventListener('input', () => {
      const v = inp.value.trim().toUpperCase();
      ok.disabled = !(v === 'KOSONGKAN' || v === 'EMPTY');
    });
    byId('backdrop').classList.add('open');
    setTimeout(() => inp.focus(), 30);
  }

  async function undoSession(id) {
    if (!bridge || busy) return;
    busy = true;
    try {
      const preview = await bridge.UndoSession(id, true);
      byId('m-title').textContent = 'Undo sesi ini?';
      byId('m-sub').textContent = (preview.restored || 0) + ' item masih bisa dipulihkan; ' + (preview.skipped || 0) + ' dilewati.';
      byId('m-body').innerHTML = '<div class="small">Item permanen atau yang dibuat lewat perintah resmi tidak bisa di-undo. Konflik tidak ditimpa.</div>';
      byId('m-cancel').textContent = 'Batal';
      setOk(t('history.undo', 'Undo sesi ini'), false, async () => {
        close();
        busy = true; updateSel();
        try {
          const res = await bridge.UndoSession(id, false);
          showResult(res, 'Hasil undo');
        } catch (e) {
          showResult({ failed: 1, items: [{ status: 'failed', reason: errText(e) }] }, 'Undo gagal');
        } finally { busy = false; updateSel(); }
      });
      byId('backdrop').classList.add('open');
    } catch (e) {
      if (window.toast) toast('Undo gagal: ' + errText(e));
    } finally { busy = false; }
  }

  async function exportReport() {
    if (!bridge || !bridge.ExportReport) return;
    const sel = byId('history-export-format');
    const btn = byId('history-export');
    const format = sel ? sel.value : 'md';
    if (btn) btn.disabled = true;
    try {
      const path = await bridge.ExportReport(format);
      note('Laporan: ' + path);
      if (window.toast) toast('Laporan diekspor: ' + path);
    } catch (e) {
      if (window.toast) toast('Ekspor gagal: ' + errText(e));
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  async function ensureHistory(force) {
    if (force) { loadedSessions = false; loadedTrash = false; }
    if (active === 'activity') { if (!loadedSessions) await loadSessions(); }
    else { if (!loadedTrash) await loadTrash(); }
  }

  function switchTab(tab) {
    active = tab === 'trash' ? 'trash' : 'activity';
    document.querySelectorAll('#history-tabs button').forEach(b => b.classList.toggle('active', b.dataset.htab === active));
    byId('history-activity').hidden = active !== 'activity';
    byId('history-trash').hidden = active !== 'trash';
    ensureHistory(false);
  }

  /* ---------------- events ---------------- */
  document.querySelectorAll('#history-tabs button').forEach(b => b.addEventListener('click', () => switchTab(b.dataset.htab)));
  C.rebind(byId('history-refresh'), () => { ensureHistory(true); });
  C.rebind(byId('history-export'), exportReport);
  C.rebind(byId('history-open-trash'), async () => {
    if (!bridge) return;
    try { await bridge.OpenTrash(); } catch (e) { if (window.toast) toast('Gagal membuka trash: ' + errText(e)); }
  });
  const searchEl = byId('history-search');
  if (searchEl) searchEl.addEventListener('input', () => { search = searchEl.value; renderTrash(false); });
  const all = byId('history-check-all');
  if (all) all.addEventListener('change', () => {
    for (const cb of document.querySelectorAll('#history-trash-body [data-trash-id]')) {
      if (all.checked) selected.add(cb.dataset.trashId); else selected.delete(cb.dataset.trashId);
      cb.checked = all.checked;
    }
    updateSel();
  });
  const tbody = byId('history-trash-body');
  if (tbody) {
    tbody.addEventListener('change', e => {
      const cb = e.target.closest('[data-trash-id]');
      if (!cb) return;
      if (cb.checked) selected.add(cb.dataset.trashId); else selected.delete(cb.dataset.trashId);
      updateSel();
    });
    tbody.addEventListener('click', e => {
      const prev = e.target.closest('[data-preview]');
      if (prev) { C.previewTrash(prev.dataset.preview); return; }
      const rest = e.target.closest('[data-restore]');
      if (rest && !rest.disabled) { confirmRestore([rest.dataset.restore]); }
    });
  }
  const sessionsBody = byId('history-sessions');
  if (sessionsBody) sessionsBody.addEventListener('click', e => {
    const btn = e.target.closest('[data-undo]');
    if (btn && !btn.disabled) undoSession(btn.dataset.undo);
  });
  C.rebind(byId('history-restore'), () => confirmRestore([...selected]));
  C.rebind(byId('history-delete'), () => confirmDelete([...selected]));
  C.rebind(byId('history-empty'), confirmEmpty);

  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn('trash:size', ev => {
      const updates = (ev && ev.updates) || [];
      if (!updates.length) return;
      const map = new Map(trash.map(e => [e.id, e]));
      for (const u of updates) {
        const e = map.get(u.id);
        if (e) { e.bytes = u.bytes; if (u.err) e.reason = u.err; }
      }
      if (current === 'history' && active === 'trash') renderTrash(false);
    });
    window.runtime.EventsOn('trash:done', () => { if (current === 'history') note(''); });
    window.runtime.EventsOn('trash:changed', () => { if (current === 'history') { loadedSessions = false; loadedTrash = false; ensureHistory(true); } });
    window.runtime.EventsOn('trash:progress', p => { if (current === 'history') note('Memproses ' + (p.done || 0) + '/' + (p.total || 0) + ' · ' + (p.name || '')); });
  }

  C.ensureHistory = ensureHistory;
  C.switchHistoryTab = switchTab;
  if (current === 'history') ensureHistory(false);
})();
