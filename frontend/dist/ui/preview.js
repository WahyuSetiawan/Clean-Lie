/* ================= M8: preview isi file/folder ================= */
(() => {
  const C = window.cleanlie;
  const { bridge, byId, esc, human, shortPath, setOk, errText } = C;
  const t = (k, fallback) => (window.__t && window.__t(k)) || fallback;

  const base = p => String(p || '').split('/').filter(Boolean).pop() || p || '';
  const size = n => (n == null || n < 0) ? '—' : human(n);
  let lastFocus = null;

  function close() {
    byId('backdrop').classList.remove('open');
    if (lastFocus && typeof lastFocus.focus === 'function') lastFocus.focus();
    lastFocus = null;
  }

  function openPreview(res, title) {
    const r = res || {};
    lastFocus = document.activeElement;
    byId('m-title').textContent = t('history.preview', 'Lihat isi') + (title ? ' · ' + title : '');
    byId('m-sub').textContent = shortPath(r.path || '');
    const chips = [
      '<span class="pill">' + esc(r.kind || '?') + '</span>',
      '<span class="pill"><b>' + size(r.bytes) + '</b></span>',
      r.files != null ? '<span class="pill">' + r.files + ' file</span>' : '',
      r.modTime ? '<span class="pill">' + esc(r.modTime.replace('T', ' ').slice(0, 16)) + '</span>' : '',
      r.blocked ? '<span class="chip locked">' + esc(t('history.blocked', 'terkunci proses')) + '</span>' : '',
      r.timedOut ? '<span class="chip warn">timeout 1,5 s</span>' : '',
      r.truncated ? '<span class="chip warn">' + esc(t('history.truncated', 'daftar dipotong')) + '</span>' : '',
    ].filter(Boolean).join('');

    let body = '<div class="row" style="gap:8px;flex-wrap:wrap;margin-bottom:10px">' + chips + '</div>';
    if (r.error) body += '<div class="cmdbox">' + esc(r.error) + '</div>';
    if (r.blocked) body += '<div class="small" style="color:var(--warn);margin:8px 0">' + esc(r.blocked) + '</div>';
    if (r.entries && r.entries.length) {
      body += '<div class="card" style="padding:0;overflow:hidden"><table><thead><tr><th>Nama</th><th class="r">Ukuran</th><th>Diubah</th></tr></thead><tbody>'
        + r.entries.map(e => '<tr><td class="mono small" style="overflow:hidden;text-overflow:ellipsis">'
          + (e.dir ? '▸ ' : e.symlink ? '↗ ' : '') + esc(e.name) + '</td><td class="r num">' + size(e.bytes) + '</td><td class="tiny dim">'
          + esc((e.modTime || '').replace('T', ' ').slice(0, 16)) + '</td></tr>').join('')
        + '</tbody></table></div>';
    } else if (r.kind === 'dir') {
      body += '<div class="small dim">Folder kosong atau isinya tidak terbaca.</div>';
    }
    body += '<div class="row" style="gap:8px;margin-top:12px"><button class="btn ghost" id="preview-open">'
      + esc(t('history.openFolder', 'Buka folder')) + '</button></div>';

    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = t('history.close', 'Tutup');
    C.rebind(byId('m-cancel'), close);
    setOk(t('history.close', 'Tutup'), false, close);
    const open = byId('preview-open');
    if (open) open.addEventListener('click', async () => {
      if (!bridge) { close(); return; }
      try { await bridge.OpenInFileManager(r.path); } catch (e) { /* toast dari pemanggil */ }
    });
    byId('backdrop').classList.add('open');
  }

  async function previewCleanup(id) {
    if (!bridge || !id) return;
    try {
      const res = await bridge.PreviewCleanup(id);
      openPreview(res, base(res.path));
    } catch (e) {
      if (window.toast) toast('Preview gagal: ' + errText(e));
    }
  }

  async function previewTrash(id) {
    if (!bridge || !id) return;
    try {
      const res = await bridge.PreviewTrash(id);
      openPreview(res, base(res.path));
    } catch (e) {
      if (window.toast) toast('Preview gagal: ' + errText(e));
    }
  }

  C.openPreview = openPreview;
  C.previewCleanup = previewCleanup;
  C.previewTrash = previewTrash;
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && byId('backdrop').classList.contains('open') && lastFocus) close();
  });
})();
