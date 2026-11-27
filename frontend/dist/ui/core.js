/* ================= M2–M4: core bersama (bridge, helper DOM, state scan) ================= */
window.__tf = function (key, vars, fallback) {
  const s = (window.__t && window.__t(key)) || fallback || key;
  return String(s).replace(/\{(\w+)\}/g, (m, k) => (vars && vars[k] != null) ? String(vars[k]) : m);
};
window.cleanlie = (() => {
  const bridge = window.go && window.go.main && window.go.main.App;

  const byId = id => document.getElementById(id);
  const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const human = b => b >= 1073741824 ? (b / 1073741824).toFixed(2) + ' GiB'
    : b >= 1048576 ? (b / 1048576).toFixed(0) + ' MiB'
    : b >= 1024 ? (b / 1024).toFixed(0) + ' KiB' : b + ' B';

  const state = { home: "", report: null };

  const shortPath = p => state.home && p.startsWith(state.home + "/") ? "~" + p.slice(state.home.length) : p;

  async function ensureHome() {
    if (state.home) return state.home;
    const d = await bridge.Diagnostics();
    state.home = d.home || "";
    return state.home;
  }

  function rebind(el, fn) {
    if (!el) return null;
    const c = el.cloneNode(true);
    el.replaceWith(c);
    c.addEventListener('click', fn);
    return c;
  }

  function setOk(label, disabled, fn) {
    let ok = byId('m-ok');
    const c = ok.cloneNode(true);
    c.textContent = label;
    c.disabled = !!disabled;
    ok.replaceWith(c);
    if (fn) c.addEventListener('click', fn);
    return c;
  }

  function errText(e) {
    return typeof e === "string" ? e : (e && e.message) || String(e);
  }

  return { bridge, byId, esc, human, shortPath, ensureHome, home: () => state.home, rebind, setOk, errText };
})();
