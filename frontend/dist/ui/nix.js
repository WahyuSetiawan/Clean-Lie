/* ================= M3: Nix ================= */
(() => {
  const C = window.cleanlie;
  if (!C.bridge) return;
  const { bridge, byId, esc, human, rebind, setOk, errText } = C;
  const tf = (k, v, f) => window.__tf ? window.__tf(k, v, f) : (f || k);

  let nixStatus = null;
  let nixDead = null;
  let nixBusy = false;
  let nixLoaded = false;

  const nixProfileName = { system: 'system', 'home-manager': 'home-manager', user: 'user (nix-env)' };


  function setNixLoading(on) {
    const refresh = byId('nix-refresh');
    if (refresh) refresh.disabled = on;
    if (on) {
      for (const id of ['nix-gc', 'nix-del-old', 'nix-vacuum']) {
        const b = byId(id);
        if (b) b.disabled = true;
      }
    }
    if (on && !nixStatus) {
      byId('nix-store-hint').textContent = tf('nix.loadingStore', null, 'memuat ukuran NAR…');
      byId('nix-dead-hint').textContent = tf('nix.loadingDead', null, 'menghitung dead paths (bisa ~30 detik)…');
      byId('nix-gen-hint').textContent = tf('nix.loadingGens', null, 'memuat generasi…');
      const tb = byId('nix-gens');
      if (tb) tb.innerHTML = '<tr><td colspan="6" class="tiny dim" style="padding:18px 16px">' + esc(tf('nix.loadingGensRow', null, 'Memuat generasi…')) + '</td></tr>';
    }
  }

  async function ensureNix(force) {
    if (nixBusy) return;
    if (nixLoaded && !force) return;
    nixBusy = true;
    setNixLoading(true);
    const jobs = [
      bridge.NixStatus().then(s => { nixStatus = s; renderNixStatus(s); })
        .catch(e => renderNixFailure('status', e)),
      bridge.NixDeadPaths().then(d => { nixDead = d; renderNixDead(d); })
        .catch(e => renderNixFailure('dead', e))
    ];
    try { await Promise.all(jobs); nixLoaded = true; }
    finally { nixBusy = false; setNixLoading(false); }
  }

  function renderNixFailure(which, e) {
    const msg = errText(e);
    if (which === 'status') {
      byId('nix-store-hint').textContent = tf('nix.fail', null, 'gagal: ') + msg;
      byId('nix-gen-hint').textContent = tf('nix.fail', null, 'gagal: ') + msg;
      const tb = byId('nix-gens');
      if (tb) tb.innerHTML = '<tr><td colspan="6" class="tiny" style="padding:18px 16px;color:var(--danger)">' + esc(msg) + '</td></tr>';
    } else {
      byId('nix-dead-size').textContent = '\u2014';
      byId('nix-dead-hint').textContent = tf('nix.fail', null, 'gagal: ') + msg;
    }
    toast('Nix: ' + msg);
  }

  function renderNixStatus(s) {
    byId('nix-store-size').textContent = human(s.storeBytes || 0);
    byId('nix-store-hint').textContent = tf('nix.pathsHint', { n: (s.storePaths || 0).toLocaleString(document.documentElement.lang === 'en' ? 'en-GB' : 'id-ID'), free: human(s.freeBytes || 0), total: human(s.totalBytes || 0) }, (s.storePaths || 0) + ' path · NAR · sisa ' + human(s.freeBytes || 0) + ' / ' + human(s.totalBytes || 0));
    byId('nix-gen-count').textContent = (s.generations || []).length;
    const parts = (s.profiles || []).filter(p => p.count).map(p => (nixProfileName[p.name] || p.name) + ' ' + p.count);
    byId('nix-gen-hint').textContent = parts.length ? parts.join(' · ') : tf('nix.noProfile', null, 'tidak ada profile');
    const dates = (s.generations || []).map(g => g.date).filter(Boolean).sort();
    byId('nix-gen-oldest').textContent = dates.length ? tf('nix.oldest', null, 'terlama: ') + dates[0] : '';
    byId('nix-journal-size').textContent = human(s.journalBytes || 0);

    const tb = byId('nix-gens');
    const gens = s.generations || [];
    if (!gens.length) {
      tb.innerHTML = '<tr><td colspan="6" class="tiny dim" style="padding:18px 16px">' + esc(tf('nix.noGenerations', null, 'Tidak ada generasi yang terbaca.')) + '</td></tr>';
    } else {
      tb.innerHTML = gens.map(g => {
        let status = g.current
          ? '<span class="gen-current">\u25cf current</span>'
          : (g.suggested ? '<span class="chip warn">saran hapus</span>' : '<span class="tiny dim">lama ' + g.ageDays + ' hari</span>');
        if (g.linkMissing) status += ' <span class="chip warn">link hilang</span>';
        const action = g.current
          ? '<span class="tiny dim">\u2014</span>'
          : '<button class="btn ghost" style="padding:5px 9px" data-nix-del="' + esc(g.profile) + '" data-nix-num="' + g.number + '">Hapus</button>';
        return '<tr><td class="mono">' + esc(nixProfileName[g.profile] || g.profile) + '</td>'
          + '<td class="num">' + g.number + '</td>'
          + '<td class="num">' + esc(g.date || '\u2014') + '</td>'
          + '<td class="r num">' + (g.closureBytes ? human(g.closureBytes) : '\u2014') + '</td>'
          + '<td>' + status + '</td>'
          + '<td class="r">' + action + '</td></tr>';
      }).join('');
      tb.querySelectorAll('button[data-nix-del]').forEach(b => {
        b.addEventListener('click', () => openDeleteOne(b.dataset.nixDel, parseInt(b.dataset.nixNum, 10)));
      });
    }

    const old = byId('nix-del-old');
    if (old) old.disabled = !gens.some(g => !g.current && g.suggested);
    const gc = byId('nix-gc');
    if (gc) gc.disabled = !nixDead;
    const vac = byId('nix-vacuum');
    if (vac) vac.disabled = false;
    if (s.errors && s.errors.length) toast('Nix: ' + s.errors[0]);
  }

  function renderNixDead(d) {
    byId('nix-dead-size').textContent = d.count ? human(d.bytes || 0) : '\u2014';
    let hint = (d.count || 0).toLocaleString('id-ID') + ' path \u00b7 ' + (d.source || 'dry-run');
    if (d.durationMs) hint += ' \u00b7 ' + (d.durationMs / 1000).toFixed(1) + ' s';
    if (d.error) hint += ' \u00b7 ' + d.error;
    byId('nix-dead-hint').textContent = hint;
    const gc = byId('nix-gc');
    if (gc) gc.disabled = false;
  }

  function openNixModal(title, sub, body, opts) {
    byId('m-title').textContent = title;
    byId('m-sub').textContent = sub || '';
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = tf('nix.cancel', null, 'Batal');
    const cancel = byId('m-cancel');
    if (cancel) cancel.disabled = false;
    const ok = setOk(opts.okLabel, !!opts.needGC, async () => {
      ok.disabled = true;
      ok.textContent = tf('nix.processing', null, 'Memproses…');
      if (cancel) cancel.disabled = true;
      try { await opts.action(); }
      catch (e) { showNixError(opts.okLabel, e); }
      finally { if (cancel) cancel.disabled = false; }
    });
    if (opts.needGC) {
      const input = byId('gc-confirm');
      if (input) input.addEventListener('input', () => { ok.disabled = input.value.trim() !== 'GC'; });
    }
    backdrop.classList.add('open');
    return ok;
  }

  function showNixError(label, e) {
    byId('m-cancel').disabled = false;
    byId('m-title').textContent = tf('nix.failed', null, 'Gagal: ') + label;
    byId('m-sub').textContent = tf('nix.noChange', null, 'Tidak ada perubahan yang diklaim berhasil.');
    byId('m-body').innerHTML = '<div class="cmdbox">' + esc(errText(e)) + '</div>';
    byId('m-cancel').textContent = tf('nix.close', null, 'Tutup');
    setOk('Tutup', false, () => backdrop.classList.remove('open'));
  }

  function showNixResult(res, label) {
    byId('m-cancel').disabled = false;
    const ok = res.exitCode === 0;
    byId('m-title').textContent = (ok ? tf('nix.done', null, 'Selesai: ') : (res.needsSudo ? tf('nix.needSudo', null, 'Perlu sudo: ') : tf('nix.failed', null, 'Gagal: '))) + label;
    byId('m-sub').textContent = 'exit ' + res.exitCode + ' \u00b7 ' + Math.round(res.durationMs || 0) + ' ms' + (res.used ? ' \u00b7 ' + res.used : '');
    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill ' + (ok ? 'safe' : '') + '">exit ' + res.exitCode + '</span>'
      + (res.deleted ? '<span class="pill">' + res.deleted + ' generasi</span>' : '')
      + (res.needsSudo ? '<span class="pill" style="color:var(--warn)">butuh sudo</span>' : '') + '</div>';
    if (res.error) body += '<div class="tiny" style="color:var(--warn);margin-bottom:8px">' + esc(res.error) + '</div>';
    if (res.preview) body += '<div class="cmdbox">' + esc(res.preview) + '</div>';
    const out = [res.stdout, res.stderr].filter(Boolean).join('\n').trim();
    if (out) body += '<div class="cmdbox" style="max-height:220px;overflow:auto">' + esc(out) + '</div>';
    if (res.needsSudo && res.sudoCommand) {
      body += '<div class="tiny" style="color:var(--warn);margin-top:10px">Jalankan perintah ini di terminal (perubahan belum terjadi):</div>'
        + '<div class="cmdbox">' + esc(res.sudoCommand) + '</div>'
        + '<button class="btn ghost" id="nix-copy-sudo">Salin perintah</button>';
    }
    byId('m-body').innerHTML = body;
    byId('m-cancel').textContent = tf('nix.close', null, 'Tutup');
    setOk('Tutup', false, () => {
      backdrop.classList.remove('open');
      if (ok) ensureNix(true);
    });
    const copy = byId('nix-copy-sudo');
    if (copy) copy.addEventListener('click', async () => {
      try { await bridge.CopyText(res.sudoCommand); toast(tf('nix.copyOk', null, 'Perintah disalin')); }
      catch (e) { toast(tf('nix.copyFail', null, 'Gagal menyalin: ') + e); }
    });
  }

  function combineNixResults(results, label) {
    if (results.length === 1) { showNixResult(results[0], label); return; }
    const combined = {
      action: label, command: [], preview: results.map(r => r.preview).filter(Boolean).join('\n'),
      sudoCommand: results.map(r => r.sudoCommand).filter(Boolean).join('\n'),
      needsSudo: results.some(r => r.needsSudo),
      used: [...new Set(results.map(r => r.used).filter(Boolean))].join(', '),
      exitCode: results.some(r => r.exitCode !== 0) ? 1 : 0,
      stdout: results.map(r => r.stdout).filter(Boolean).join('\n'),
      stderr: results.map(r => r.stderr).filter(Boolean).join('\n'),
      error: results.map(r => r.error).filter(Boolean).join(' \u00b7 '),
      durationMs: results.reduce((a, r) => a + (r.durationMs || 0), 0),
      deleted: results.reduce((a, r) => a + (r.deleted || 0), 0)
    };
    showNixResult(combined, label);
  }

  function gcGateBody(cmd) {
    return '<div class="field" style="border:0;padding:12px 0 0"><div class="label">'
      + '<b>Ketik GC untuk konfirmasi</b><span>Tindakan ini tidak bisa di-undo.</span></div>'
      + '<div class="ctrl"><input id="gc-confirm" autocomplete="off" spellcheck="false" placeholder="GC"></div></div>'
      + '<div class="cmdbox" style="margin-top:10px">' + esc(cmd || '') + '</div>';
  }

  function openGC() {
    if (!nixStatus) return;
    if (!nixDead) { toast(tf('nix.busyDead', null, 'Dead paths masih dihitung')); return; }
    const cmd = nixStatus.gcCommand || 'sudo nix-collect-garbage -d';
    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">store <b>' + human(nixStatus.storeBytes || 0) + '</b> (NAR)</span>'
      + '<span class="pill">dead <b>' + (nixDead ? human(nixDead.bytes || 0) : '\u2014') + '</b></span></div>'
      + '<div class="callout"><span class="ico">!</span><div class="small"><b>GC tidak bisa di-undo.</b> '
      + 'Hanya path yang tidak lagi direferensikan yang dihapus. Jalankan dry-run dulu bila ragu.</div></div>';
    body += gcGateBody(cmd);
    openNixModal('Jalankan garbage collection?', 'GC menghapus dead paths dan generasi lama sesuai profile.', body, {
      needGC: true, okLabel: 'Jalankan GC',
      action: async () => { showNixResult(await bridge.NixCollectGarbage('GC'), 'GC'); }
    });
  }

  async function openDeleteOld() {
    if (!nixStatus) return;
    const gens = (nixStatus.generations || []).filter(g => !g.current && g.suggested);
    if (!gens.length) { toast(tf('nix.noOld', null, 'Tidak ada generasi > 30 hari')); return; }
    const by = {};
    for (const g of gens) (by[g.profile] = by[g.profile] || []).push(g);
    const entries = Object.entries(by);
    const previews = await Promise.all(entries.map(([p, list]) =>
      bridge.NixPreview('delete', p, list.map(g => g.number)).catch(() => '')));
    let body = '<div class="row" style="gap:8px;margin-bottom:10px"><span class="pill">' + gens.length + ' generasi</span>'
      + '<span class="pill">' + entries.length + ' profile</span></div>';
    entries.forEach(([p, list], i) => {
      body += '<div class="tiny dim" style="margin-top:8px">' + esc(nixProfileName[p] || p) + ' \u00b7 ' + list.length + ' generasi</div>'
        + '<div class="cmdbox">' + esc(previews[i] || '') + '</div>';
    });
    body += '<div class="tiny dim" style="margin-top:8px">Generasi current tidak pernah ikut. Ruang baru benar-benar bebas setelah GC.</div>';
    body += gcGateBody(previews.filter(Boolean).join('\n'));
    openNixModal('Hapus ' + gens.length + ' generasi > 30 hari?', 'Profile system butuh root; home-manager dan user berjalan sebagai user.', body, {
      needGC: true, okLabel: 'Hapus generasi',
      action: async () => {
        const results = [];
        for (const [p, list] of entries) {
          results.push(await bridge.NixDeleteGenerations({ profile: p, numbers: list.map(g => g.number), confirm: 'GC' }));
        }
        combineNixResults(results, 'Hapus generasi');
      }
    });
  }

  async function openDeleteOne(profile, num) {
    const g = (nixStatus && nixStatus.generations || []).find(x => x.profile === profile && x.number === num);
    if (!g || g.current) return;
    let cmd = '';
    try { cmd = await bridge.NixPreview('delete', profile, [num]); } catch (e) { cmd = ''; }
    let body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">' + esc(nixProfileName[profile] || profile) + ' #' + num + '</span>'
      + '<span class="pill">' + esc(g.date) + '</span>'
      + '<span class="pill">closure <b>' + (g.closureBytes ? human(g.closureBytes) : '\u2014') + '</b></span></div>';
    body += '<div class="tiny dim" style="margin-bottom:8px">Ruang baru benar-benar bebas setelah GC.</div>';
    body += gcGateBody(cmd);
    openNixModal('Hapus generasi ' + (nixProfileName[profile] || profile) + ' #' + num + '?', 'Generasi current tidak bisa dihapus.', body, {
      needGC: true, okLabel: 'Hapus generasi',
      action: async () => {
        const res = await bridge.NixDeleteGenerations({ profile, numbers: [num], confirm: 'GC' });
        showNixResult(res, 'Hapus generasi');
      }
    });
  }

  async function openVacuum() {
    const sel = byId('nix-journal-target');
    const target = ((sel && sel.value) || '500 MB').replace(/\s+/g, '');
    let cmd = '';
    try { cmd = await bridge.NixPreview('journal', target); } catch (e) { cmd = ''; }
    const body = '<div class="row" style="gap:8px;margin-bottom:10px">'
      + '<span class="pill">sekarang <b>' + human((nixStatus && nixStatus.journalBytes) || 0) + '</b></span>'
      + '<span class="pill">target <b>' + esc(target) + '</b></span></div>'
      + '<div class="cmdbox">' + esc(cmd) + '</div>'
      + '<div class="tiny dim" style="margin-top:8px">Vacuum butuh root. Tanpa polkit agent, perintah sudo akan ditampilkan untuk disalin.</div>';
    openNixModal('Vacuum journal ke ' + target + '?', 'Journal lama dihapus, journal aktif dipertahankan.', body, {
      needGC: false, okLabel: 'Vacuum',
      action: async () => { showNixResult(await bridge.VacuumJournal(target), 'Vacuum journal'); }
    });
  }

  rebind(byId('nix-refresh'), () => ensureNix(true));
  rebind(byId('nix-gc'), openGC);
  rebind(byId('nix-del-old'), openDeleteOld);
  rebind(byId('nix-vacuum'), openVacuum);

  window.addEventListener('cleanlie:lang', () => {
    if (nixStatus) renderNixStatus(nixStatus);
    if (nixDead) renderNixDead(nixDead);
  });

  C.ensureNix = ensureNix;
})();
