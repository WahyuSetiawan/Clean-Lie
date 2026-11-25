/* ================= M4: Android ================= */
(() => {
  const C = window.cleanlie;
  if (!C.bridge) return;
  const { bridge, byId, esc, human, rebind, setOk, errText } = C;

  let androidStatus = null;
  let androidBusy = false;
  let androidLoaded = false;
  const androidSel = new Set();

  function androidKey(type, target) { return type + ':' + target; }

  function androidItem(type, target) {
    if (!androidStatus) return null;
    if (type === 'avd') {
      const a = (androidStatus.avds || []).find(x => x.name === target);
      if (!a) return null;
      return { type: 'avd', target: a.name, label: a.name, bytes: a.bytes, running: a.running, suggested: a.suggested };
    }
    const p = (androidStatus.packages || []).find(x => x.id === target);
    if (!p) return null;
    return { type: 'package', target: p.id, label: p.id, bytes: p.bytes, pkgKind: p.kind, suggested: p.suggested, usedBy: p.usedBy || [] };
  }

  function androidItems() {
    const out = [];
    for (const key of androidSel) {
      const i = key.indexOf(':');
      const item = androidItem(key.slice(0, i), key.slice(i + 1));
      if (item) out.push(item);
    }
    return out;
  }

  function syncAndroidSel() {
    const items = androidItems();
    const total = items.reduce((a, b) => a + (b.bytes || 0), 0);
    const running = !!(androidStatus && androidStatus.emulatorRunning);
    const sel = byId('android-selected');
    if (sel) sel.textContent = items.length ? items.length + ' dipilih \u00b7 ' + human(total) : '0 dipilih';
    const del = byId('android-delete');
    if (del) del.disabled = !items.length || androidBusy || running;
    const wipe = byId('android-wipe');
    if (wipe) wipe.disabled = !items.some(i => i.type === 'avd') || androidBusy || running;
  }

  function setAndroidLoading(on) {
    const refresh = byId('android-refresh');
    if (refresh) refresh.disabled = on;
    for (const id of ['android-suggest', 'android-wipe', 'android-delete']) {
      const b = byId(id);
      if (b) b.disabled = !!on;
    }
    if (on && !androidStatus) {
      byId('android-avd-hint').textContent = 'memuat AVD\u2026';
      byId('android-sdk-hint').textContent = 'memuat paket SDK\u2026';
      byId('android-running-hint').textContent = 'memeriksa proses emulator\u2026';
      byId('android-avds').innerHTML = '<tr><td colspan="8" class="tiny dim" style="padding:18px 16px">Memuat AVD\u2026</td></tr>';
      byId('android-sysimg').innerHTML = '<tr><td colspan="4" class="tiny dim" style="padding:18px 16px">Memuat system image\u2026</td></tr>';
      byId('android-tools').innerHTML = '<tr><td colspan="4" class="tiny dim" style="padding:18px 16px">Memuat NDK & build-tools\u2026</td></tr>';
    }
  }

  async function ensureAndroid(force) {
    if (androidBusy) return;
    if (androidLoaded && !force) return;
    androidBusy = true;
    setAndroidLoading(true);
    try {
      if (force) {
        try { await bridge.AndroidRefresh(); } catch (e) { /* lanjut baca cache */ }
      }
      const st = await bridge.AndroidStatus();
      androidStatus = st;
      renderAndroid(st);
      androidLoaded = true;
    } catch (e) {
      renderAndroidFailure(e);
    } finally {
      androidBusy = false;
      setAndroidLoading(false);
      syncAndroidSel();
    }
  }

  function renderAndroidFailure(e) {
    const msg = errText(e);
    byId('android-avd-hint').textContent = 'gagal: ' + msg;
    byId('android-sdk-hint').textContent = 'gagal: ' + msg;
    byId('android-running-hint').textContent = 'gagal: ' + msg;
    byId('android-avds').innerHTML = '<tr><td colspan="8" class="tiny" style="padding:18px 16px;color:var(--danger)">' + esc(msg) + '</td></tr>';
    byId('android-sysimg').innerHTML = '<tr><td colspan="4" class="tiny" style="padding:18px 16px;color:var(--danger)">' + esc(msg) + '</td></tr>';
    byId('android-tools').innerHTML = '<tr><td colspan="4" class="tiny" style="padding:18px 16px;color:var(--danger)">' + esc(msg) + '</td></tr>';
    toast('Android: ' + msg);
  }

  function androidCheck(type, target, disabled) {
    const checked = androidSel.has(androidKey(type, target)) ? ' checked' : '';
    const dis = disabled ? ' disabled' : '';
    return '<input type="checkbox" class="check" data-ad-type="' + type + '" data-ad-target="' + esc(target) + '"' + checked + dis + '>';
  }

  function renderAndroid(st) {
    const avds = st.avds || [];
    const pkgs = st.packages || [];
    const running = !!st.emulatorRunning;

    const valid = new Set();
    for (const a of avds) valid.add(androidKey('avd', a.name));
    for (const p of pkgs) valid.add(androidKey('package', p.id));
    for (const key of [...androidSel]) if (!valid.has(key)) androidSel.delete(key);

    byId('android-avd-size').textContent = human(st.avdBytes || 0);
    byId('android-avd-hint').textContent = avds.length + ' AVD' + (running ? ' \u00b7 ada yang berjalan' : ' \u00b7 tidak ada yang jalan');
    byId('android-sdk-size').textContent = human(st.sdkBytes || 0);
    const sum = kind => pkgs.filter(p => p.kind === kind).reduce((a, p) => a + (p.bytes || 0), 0);
    byId('android-sdk-hint').textContent = 'system-images ' + human(sum('system-image')) + ' \u00b7 NDK ' + human(sum('ndk'));
    const runEl = byId('android-running');
    runEl.textContent = running ? 'jalan' : 'tidak jalan';
    runEl.style.color = running ? 'var(--danger)' : 'var(--accent)';
    byId('android-running-hint').textContent = running
      ? 'matikan emulator sebelum hapus/wipe'
      : 'aman untuk hapus/wipe';

    const avdBody = byId('android-avds');
    if (!avds.length) {
      avdBody.innerHTML = '<tr><td colspan="8" class="tiny dim" style="padding:18px 16px">Tidak ada AVD.</td></tr>';
    } else {
      avdBody.innerHTML = avds.map(a => {
        let status = a.running
          ? '<span class="chip danger">jalan</span>'
          : (a.suggested ? '<span class="chip warn">saran hapus</span>' : '<span class="tiny dim">' + (a.ageDays || 0) + ' hari</span>');
        if (a.error) status += ' <span class="chip danger">' + esc(a.error) + '</span>';
        const dis = a.running ? ' disabled' : '';
        return '<tr>'
          + '<td>' + androidCheck('avd', a.name, running || a.running) + '</td>'
          + '<td class="mono">' + esc(a.name) + '</td>'
          + '<td class="num">' + esc(a.api || '\u2014') + '</td>'
          + '<td class="mono">' + esc(a.abi || '\u2014') + '</td>'
          + '<td class="r num">' + human(a.bytes || 0) + '</td>'
          + '<td class="num">' + esc(a.lastUsed ? a.lastUsed.slice(0, 10) : '\u2014') + '</td>'
          + '<td>' + status + '</td>'
          + '<td class="r"><button class="btn ghost" style="padding:5px 9px" data-ad-act="wipe" data-ad-target="' + esc(a.name) + '"' + dis + '>Wipe</button> '
          + '<button class="btn ghost" style="padding:5px 9px" data-ad-act="delete" data-ad-target="' + esc(a.name) + '"' + dis + '>Hapus</button></td>'
          + '</tr>';
      }).join('');
    }

    const sys = pkgs.filter(p => p.kind === 'system-image');
    const sysBody = byId('android-sysimg');
    if (!sys.length) {
      sysBody.innerHTML = '<tr><td colspan="4" class="tiny dim" style="padding:18px 16px">Tidak ada system image.</td></tr>';
    } else {
      sysBody.innerHTML = sys.map(p => {
        const used = (p.usedBy && p.usedBy.length)
          ? p.usedBy.map(n => '<span class="chip safe">' + esc(n) + '</span>').join(' ')
          : (p.suggested ? '<span class="chip warn">saran hapus</span>' : '<span class="tiny dim">tidak dipakai AVD</span>');
        return '<tr>'
          + '<td>' + androidCheck('package', p.id, running) + '</td>'
          + '<td class="mono small">' + esc(p.id) + '</td>'
          + '<td class="r num">' + human(p.bytes || 0) + '</td>'
          + '<td>' + used + '</td>'
          + '</tr>';
      }).join('');
    }

    const tools = pkgs.filter(p => p.kind !== 'system-image');
    const toolsBody = byId('android-tools');
    if (!tools.length) {
      toolsBody.innerHTML = '<tr><td colspan="4" class="tiny dim" style="padding:18px 16px">Tidak ada NDK/build-tools/platform.</td></tr>';
    } else {
      toolsBody.innerHTML = tools.map(p => {
        const used = (p.usedBy && p.usedBy.length)
          ? p.usedBy.map(n => '<span class="chip safe">' + esc(n) + '</span>').join(' ')
          : '<span class="tiny dim">cek proyek</span>';
        return '<tr>'
          + '<td>' + androidCheck('package', p.id, running) + '</td>'
          + '<td class="mono small">' + esc(p.id) + '</td>'
          + '<td class="r num">' + human(p.bytes || 0) + '</td>'
          + '<td>' + used + '</td>'
          + '</tr>';
      }).join('');
    }

    bindAndroidRows(avdBody, 'avd');
    bindAndroidRows(sysBody, 'package');
    bindAndroidRows(toolsBody, 'package');
    syncAndroidSel();
  }

  function bindAndroidRows(body, type) {
    body.querySelectorAll('input[data-ad-type]').forEach(cb => {
      cb.addEventListener('change', () => {
        const key = androidKey(cb.dataset.adType, cb.dataset.adTarget);
        if (cb.checked) androidSel.add(key); else androidSel.delete(key);
        syncAndroidSel();
      });
    });
    body.querySelectorAll('button[data-ad-act]').forEach(btn => {
      btn.addEventListener('click', () => {
        const target = btn.dataset.adTarget;
        if (btn.dataset.adAct === 'wipe') { openAndroidWipe([target]); return; }
        const item = androidItem(type, target);
        if (item) openAndroidDelete([item]);
      });
    });
  }

  function selectAndroidSuggested() {
    if (!androidStatus) return;
    androidSel.clear();
    let n = 0;
    for (const a of androidStatus.avds || []) {
      if (a.suggested) { androidSel.add(androidKey('avd', a.name)); n++; }
    }
    for (const p of androidStatus.packages || []) {
      if (p.suggested) { androidSel.add(androidKey('package', p.id)); n++; }
    }
    renderAndroid(androidStatus);
    if (!n) { toast('Tidak ada item yang disarankan'); return; }
    const total = androidItems().reduce((a, b) => a + (b.bytes || 0), 0);
    toast(n + ' item disarankan dipilih \u00b7 ' + human(total));
  }

  function openAndroidModal(title, sub, body, okLabel, disabled, action) {
    byId('m-title').textContent = title;
    byId('m-sub').textContent = sub || '';
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = 'Batal';
    const cancel = byId('m-cancel');
    cancel.disabled = false;
    const ok = setOk(okLabel, disabled, async () => {
      ok.disabled = true;
      ok.textContent = 'Memproses\u2026';
      cancel.disabled = true;
      androidBusy = true;
      syncAndroidSel();
      try {
        const results = await action();
        showAndroidResults(results, okLabel);
      } catch (e) {
        showAndroidError(okLabel, e);
      } finally {
        androidBusy = false;
        cancel.disabled = false;
        syncAndroidSel();
      }
    });
    backdrop.classList.add('open');
    return ok;
  }

  function showAndroidError(label, e) {
    byId('m-cancel').disabled = false;
    byId('m-title').textContent = 'Gagal: ' + label;
    byId('m-sub').textContent = 'Tidak ada perubahan yang diklaim berhasil.';
    byId('m-body').innerHTML = '<div class="cmdbox">' + esc(errText(e)) + '</div>';
    byId('m-cancel').textContent = 'Tutup';
    setOk('Tutup', false, () => backdrop.classList.remove('open'));
  }

  function showAndroidResults(results, label) {
    byId('m-cancel').disabled = false;
    const list = results.filter(Boolean);
    const freed = list.reduce((a, r) => a + (r.freed || 0), 0);
    const good = list.filter(r => (r.deleted || (r.action === 'wipe' && r.exitCode === 0)) && !r.error);
    const ok = list.length > 0 && good.length === list.length;
    byId('m-title').textContent = (ok ? 'Selesai: ' : 'Sebagian gagal: ') + label;
    byId('m-sub').textContent = good.length + ' dari ' + list.length + ' item selesai';
    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill ' + (ok ? 'safe' : '') + '"><b>' + human(freed) + '</b> dibebaskan</span>'
      + '<span class="pill">' + good.length + ' berhasil</span></div>';
    list.forEach(r => {
      const target = r.target || '';
      const success = (r.deleted || (r.action === 'wipe' && r.exitCode === 0)) && !r.error;
      body += '<div class="row" style="gap:8px;margin-top:8px"><span class="pill ' + (success ? 'safe' : '') + '">' + (success ? 'ok' : 'gagal') + '</span>'
        + '<span class="mono small">' + esc(target) + '</span><span class="spacer"></span>'
        + '<span class="num">' + human(r.freed || 0) + '</span></div>';
      if (r.error) body += '<div class="tiny" style="color:var(--warn)">' + esc(r.error) + '</div>';
      if (r.preview) body += '<div class="cmdbox" style="max-height:100px;overflow:auto">' + esc(r.preview) + '</div>';
      const out = [r.stdout, r.stderr].filter(Boolean).join('\n').trim();
      if (out) body += '<div class="cmdbox" style="max-height:100px;overflow:auto">' + esc(out) + '</div>';
    });
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = 'Tutup';
    setOk('Tutup', false, () => {
      backdrop.classList.remove('open');
      if (good.length) ensureAndroid(true);
    });
  }

  async function openAndroidDelete(items) {
    if (!items.length) { toast('Pilih minimal satu item'); return; }
    if (androidStatus && androidStatus.emulatorRunning) { toast('Matikan emulator dulu'); return; }
    const previews = await Promise.all(items.map(i =>
      bridge.AndroidPreview(i.type === 'avd' ? 'delete-avd' : 'uninstall', i.target).catch(() => '')));
    const total = items.reduce((a, b) => a + (b.bytes || 0), 0);
    let body = '<div class="row" style="gap:8px;margin-bottom:10px"><span class="pill">' + items.length + ' item</span>'
      + '<span class="pill"><b>' + human(total) + '</b></span></div>';
    items.forEach((i, idx) => {
      body += '<div class="row" style="gap:8px;margin-top:6px"><span class="pill">' + (i.type === 'avd' ? 'AVD' : 'SDK') + '</span>'
        + '<span class="mono small">' + esc(i.label) + '</span><span class="spacer"></span>'
        + '<span class="num">' + human(i.bytes || 0) + '</span></div>'
        + '<div class="cmdbox">' + esc(previews[idx] || '') + '</div>';
    });
    body += '<div class="callout"><span class="ico">!</span><div class="small"><b>Data hilang permanen.</b> '
      + 'AVD menghapus snapshot/data emulator; SDK harus diunduh ulang bila dibutuhkan lagi.</div></div>';
    body += '<label class="row" style="gap:8px;margin-top:12px"><input type="checkbox" id="ad-confirm">'
      + '<span class="small">Saya mengerti dan tetap ingin menghapus item di atas.</span></label>';
    const ok = openAndroidModal('Hapus ' + items.length + ' item Android?', 'Konfirmasi eksplisit diperlukan.', body, 'Hapus item', true, async () => {
      const results = [];
      for (const i of items) {
        try {
          if (i.type === 'avd') results.push(await bridge.AndroidDeleteAVD(i.target));
          else results.push(await bridge.AndroidUninstallPackage(i.target));
        } catch (e) {
          results.push({ action: i.type, target: i.target, error: errText(e), freed: 0 });
        }
      }
      return results;
    });
    const gate = byId('ad-confirm');
    if (gate) gate.addEventListener('change', () => { ok.disabled = !gate.checked; });
  }

  async function openAndroidWipe(names) {
    if (!names.length) { toast('Pilih AVD dulu'); return; }
    if (androidStatus && androidStatus.emulatorRunning) { toast('Matikan emulator dulu'); return; }
    const previews = await Promise.all(names.map(n => bridge.AndroidPreview('wipe', n).catch(() => '')));
    let body = '<div class="callout"><span class="ico">!</span><div class="small"><b>Wipe menghapus data & snapshot AVD.</b> '
      + 'Emulator dijalankan headless sampai boot lalu keluar; bisa beberapa menit.</div></div>';
    names.forEach((n, idx) => {
      body += '<div class="row" style="gap:8px;margin-top:8px"><span class="pill">AVD</span>'
        + '<span class="mono small">' + esc(n) + '</span><span class="spacer"></span>'
        + '<span class="tiny dim">' + (androidItem('avd', n) ? human(androidItem('avd', n).bytes) : '') + '</span></div>'
        + '<div class="cmdbox">' + esc(previews[idx] || '') + '</div>';
    });
    body += '<label class="row" style="gap:8px;margin-top:12px"><input type="checkbox" id="ad-wipe-confirm">'
      + '<span class="small">Saya mengerti snapshot dan data emulator akan hilang.</span></label>';
    const ok = openAndroidModal('Wipe ' + names.length + ' AVD?', 'AVD tetap ada; hanya data yang di-reset.', body, 'Wipe data', true, async () => {
      const results = [];
      for (const n of names) {
        try { results.push(await bridge.AndroidWipeAVD(n)); }
        catch (e) { results.push({ action: 'wipe', target: n, error: errText(e), freed: 0 }); }
      }
      return results;
    });
    const gate = byId('ad-wipe-confirm');
    if (gate) gate.addEventListener('change', () => { ok.disabled = !gate.checked; });
  }

  rebind(byId('android-refresh'), () => { ensureAndroid(true); });
  rebind(byId('android-suggest'), selectAndroidSuggested);
  rebind(byId('android-delete'), () => openAndroidDelete(androidItems()));
  rebind(byId('android-wipe'), () => {
    const names = androidItems().filter(i => i.type === 'avd').map(i => i.target);
    openAndroidWipe(names);
  });

  C.ensureAndroid = ensureAndroid;
})();
