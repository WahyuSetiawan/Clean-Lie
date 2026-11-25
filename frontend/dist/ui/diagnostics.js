/* Diagnostik: satu-satunya jalur yang benar-benar memanggil Go di M0. */
(async function () {
  const badge = document.getElementById('app-badge');
  const bridge = window.go && window.go.main && window.go.main.App;
  if (!bridge) {
    if (badge) { badge.textContent = 'preview'; badge.title = 'dibuka di browser: binding Go tidak aktif'; }
    return;
  }
  try {
    const d = await bridge.Diagnostics();
    if (badge) { badge.textContent = 'v' + d.version; badge.title = 'commit ' + d.commit; }

    const set = (id, text) => { const el = document.getElementById(id); if (el) el.textContent = text; };
    set('diag-version', d.version);
    set('diag-commit', d.commit);
    set('diag-go', d.goVersion);
    set('diag-platform', d.platform);
    set('diag-home', d.home);
    set('diag-nix', d.nixStore ? 'terdeteksi' : 'tidak ada');
    set('diag-polkit', d.polkitDetail);

    const chip = document.getElementById('st-polkit-chip');
    if (chip) {
      chip.textContent = d.polkitAgent ? 'ok' : '!';
      chip.className = 'chip ' + (d.polkitAgent ? 'safe' : 'warn');
    }
    set('st-polkit', d.polkitAgent ? 'Polkit agent terdeteksi' : 'Polkit agent tidak ada');
    set('st-polkit-detail', d.polkitAgent ? 'aksi root via pkexec' : 'fallback: perintah sudo siap-copy');
  } catch (e) {
    console.error('diagnostics', e);
    if (badge) badge.title = 'binding error: ' + e;
  }
})();
