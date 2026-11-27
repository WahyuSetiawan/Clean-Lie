/* ================= boot layar M2–M4: preload saat navigasi ================= */
(() => {
  const C = window.cleanlie;
  if (!C.bridge) return;

  /* buka scan saat pertama kali masuk layar Bersihkan / Nix / Android */
  const baseNavigate = navigate;
  navigate = function (view) {
    baseNavigate(view);
    if (view === "cleanup") C.ensureScan();
    if (view === "nix") C.ensureNix();
    if (view === "android") C.ensureAndroid();
    if (view === "history" && C.ensureHistory) C.ensureHistory();
  };
  if (current === "cleanup") C.ensureScan();
  if (current === "nix") C.ensureNix();
  if (current === "android") C.ensureAndroid();
  if (current === "history" && C.ensureHistory) C.ensureHistory();
})();
