/* ================= M6: settings, theme, i18n ================= */
(function () {
  const bridge = window.go && window.go.main && window.go.main.App;
  const byId = id => document.getElementById(id);
  const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const MIB = 1024 * 1024;

  const STRINGS = {
    id: {
      'brand.tagline': 'cleaner untuk NixOS',
      'brand.tagline.macos': 'cleaner untuk macOS',
      'side.note': 'M10 · otomasi & cakupan',
      'nav.dashboard': 'Dashboard',
      'nav.cleanup': 'Bersihkan',
      'nav.analysis': 'Analisis',
      'nav.history': 'Riwayat',
      'nav.direnv': 'direnv',
      'nav.settings': 'Pengaturan',
      'page.dashboard.title': 'Dashboard',
      'page.dashboard.sub': 'Ringkasan disk dan reclaimable space',
      'page.cleanup.title': 'Bersihkan',
      'page.cleanup.sub': 'Pilih item per kategori · default hanya item aman',
      'page.nix.title': 'Nix',
      'page.nix.sub': 'Store, generasi, dan garbage collection',
      'page.android.title': 'Android',
      'page.android.sub': 'AVD, system image, dan NDK',
      'page.analysis.title': 'Analisis',
      'page.analysis.sub': 'Disk usage, file besar, duplikat, file lama, artefak build',
      'page.direnv.title': 'direnv',
      'page.direnv.sub': 'Cari folder .direnv yang berat dan bisa dibuat ulang',
      'page.history.title': 'Riwayat',
      'page.history.sub': 'Aktivitas pembersihan dan isi trash',
      'history.tab.activity': 'Aktivitas',
      'history.tab.trash': 'Trash',
      'history.refresh': 'Muat ulang',
      'history.openTrash': 'Buka folder trash',
      'history.openFolder': 'Buka folder',
      'history.sessions': 'Sesi pembersihan',
      'history.sessionsHint': 'Satu sesi = satu batch audit; item yang masih di trash bisa dipulihkan.',
      'history.time': 'Waktu',
      'history.action': 'Aksi',
      'history.items': 'Item',
      'history.freed': 'Dibebaskan',
      'history.undoable': 'Bisa di-undo',
      'history.actions': 'Aksi',
      'history.trashTitle': 'Isi trash',
      'history.trashHint': 'Ukuran dihitung di background. Pulihkan hanya item yang tidak bentrok.',
      'history.search': 'Cari nama atau path…',
      'history.restore': 'Pulihkan',
      'history.delete': 'Hapus permanen',
      'history.empty': 'Kosongkan…',
      'history.name': 'Nama',
      'history.origin': 'Asal',
      'history.deletedAt': 'Dihapus',
      'history.size': 'Ukuran',
      'history.state': 'Status',
      'history.undo': 'Undo sesi ini',
      'history.preview': 'Lihat isi',
      'history.selected': 'dipilih',
      'history.noSessions': 'Belum ada aktivitas pembersihan.',
      'history.noTrash': 'Trash kosong.',
      'history.notRestorable': 'tidak bisa dipulihkan',
      'history.emptyConfirm': 'Ketik KOSONGKAN atau EMPTY untuk mengosongkan seluruh trash. Tindakan ini tidak bisa dibatalkan.',
      'history.deleteConfirm': 'Hapus permanen item terpilih? Tindakan ini tidak bisa dibatalkan.',
      'history.result': 'Hasil',
      'history.close': 'Tutup',
      'history.blocked': 'terkunci proses',
      'history.truncated': 'Daftar dipotong; gunakan pencarian untuk mempersempit.',
      'history.emptyOk': 'Kosongkan',
      'history.deleteOk': 'Hapus permanen',
      'history.restoreOk': 'Pulihkan',
      'history.export': 'Ekspor laporan',
      'settings.openHistory': 'Lihat riwayat',
      'settings.openFolder': 'Folder',
      'direnv.roots': 'Folder proyek',
      'direnv.rootsHint': 'maks 32',
      'direnv.rootsNote': 'Klik Pilih folder… atau ketik path. Cleanlie mencari folder .direnv di dalamnya; nix-direnv menyimpan profil yang bisa besar.',
      'direnv.pick': 'Pilih folder…',
      'direnv.add': '+ Tambah',
      'direnv.scan': 'Scan direnv',
      'direnv.cancel': 'Batal',
      'direnv.refresh': 'Scan ulang',
      'direnv.summary': 'Ringkasan',
      'direnv.cacheNote': 'Cache global ~/.cache/direnv dan ~/.cache/nix-direnv ikut diukur. Isi nix-direnv sebagian besar symlink; ruang nyata dibebaskan oleh GC Nix.',
      'direnv.table': 'Folder .direnv',
      'direnv.tableHint': 'Ukuran allocated, symlink tidak diikuti',
      'direnv.selectSafe': 'Pilih aman',
      'direnv.trash': 'Pindah ke trash',
      'direnv.project': 'Proyek',
      'direnv.location': 'Lokasi',
      'direnv.size': 'Ukuran',
      'direnv.changed': 'Terakhir',
      'direnv.state': 'Status',
      'page.settings.title': 'Pengaturan',
      'page.settings.sub': 'Preferensi pembersihan dan tampilan',
      'theme.system': 'Sistem',
      'theme.light': 'Terang',
      'theme.dark': 'Gelap',
      'settings.appearance': 'Tampilan',
      'settings.theme': 'Tema',
      'settings.themeHint': 'Mengikuti sistem atau paksa terang/gelap',
      'settings.language': 'Bahasa',
      'settings.languageHint': 'Label, tanggal, dan format angka',
      'settings.showRisky': 'Tampilkan item berisiko',
      'settings.showRiskyHint': 'Item ⚠ dan ⛔ tetap tidak pernah dipilih otomatis',
      'settings.cleanup': 'Pembersihan',
      'settings.deleteMode': 'Mode hapus default',
      'settings.deleteModeHint': 'Trash bisa di-restore dari file manager',
      'settings.trash': 'Trash (bisa restore)',
      'settings.permanent': 'Hapus permanen',
      'settings.autoSafe': 'Item aman otomatis dicentang',
      'settings.autoSafeHint': 'Item ⚠/⛔ tidak pernah otomatis',
      'settings.notify': 'Notifikasi desktop',
      'settings.notifyHint': 'Kabari saat pembersihan atau GC selesai',
      'settings.journalTarget': 'Target journal',
      'settings.journalTargetHint': 'Vacuum system dan user journal',
      'settings.analysis': 'Analisis',
      'settings.largeThreshold': 'Ambang file besar',
      'settings.largeThresholdHint': 'Ditampilkan di tab File besar',
      'settings.staleDays': 'Ambang file lama',
      'settings.staleDaysHint': 'mtime lebih tua dari',
      'settings.days': 'hari',
      'settings.artifactRoots': 'Root scan tambahan',
      'settings.artifactRootsHint': 'Pisahkan dengan koma; dipakai untuk scan artefak',
      'settings.customRules': 'Aturan kustom',
      'settings.customRulesLabel': 'Folder cache tambahan',
      'settings.customRulesHint': 'Satu baris: path | label | risk | whole',
      'settings.customRulesNote': 'risk: safe/warn/danger · whole opsional · maks 32 aturan',
      'settings.automation': 'Otomasi',
      'settings.scheduleInterval': 'Jadwal auto-clean',
      'settings.scheduleIntervalHint': 'Timer user systemd; hanya item aman ke trash',
      'settings.weekly': 'Mingguan',
      'settings.daily': 'Harian',
      'settings.timer': 'Timer',
      'settings.installTimer': 'Pasang',
      'settings.removeTimer': 'Hapus',
      'settings.autoCleanNote': 'Auto-clean memakai --yes, mode trash, dan tidak pernah menghapus permanen.',
      'settings.timerInstalled': 'terpasang',
      'settings.timerActive': 'aktif',
      'settings.timerOff': 'belum dipasang',
      'settings.timerFailed': 'Gagal mengatur timer: ',
      'settings.timerConfirm': 'Hapus timer auto-clean?',
      'settings.exclusions': 'Pengecualian & akses',
      'settings.excluded': 'Path dikecualikan',
      'settings.excludedHint': 'Tidak pernah dipindai atau dihapus',
      'settings.rootAction': 'Aksi root',
      'settings.rootActionHint': 'Cara aksi root diprompt di sesi ini',
      'settings.history': 'Riwayat',
      'settings.historyHint': 'Audit log semua aksi hapus',
      'settings.open': 'Buka',
      'settings.save': 'Simpan pengaturan',
      'settings.reset': 'Kembalikan default',
      'settings.configPath': 'Config',
      'settings.runtime': 'Runtime',
      'settings.version': 'Versi',
      'settings.commit': 'Commit',
      'settings.platform': 'Platform',
      'settings.saved': 'Pengaturan disimpan.',
      'settings.saving': 'Menyimpan…',
      'settings.dirty': 'Perubahan belum disimpan.',
      'settings.loadFailed': 'Gagal membaca pengaturan: ',
      'settings.saveFailed': 'Gagal menyimpan: ',
      'settings.preview': 'Mode preview: perubahan tidak ditulis ke disk.',
      'settings.warning': 'Config lama: ',
      'settings.polkit': 'Polkit',
      'settings.polkitYes': 'agent terdeteksi',
      'settings.polkitNo': 'tidak ada agent',
      'settings.adminOsascript': 'dialog admin macOS',
      'analysis.tab.tree': 'Disk usage',
      'analysis.tab.large': 'File besar',
      'analysis.tab.dup': 'Duplikat',
      'analysis.tab.stale': 'File lama',
      'analysis.tab.artifacts': 'Artefak build',
      'analysis.scanAgain': 'Scan ulang',
      'analysis.openFolder': 'Buka di file manager',
      'analysis.folder': 'Folder',
      'analysis.size': 'Ukuran',
      'analysis.share': 'Proporsi',
      'analysis.map': 'Peta folder',
      'analysis.mapHint': 'klik untuk masuk',
      'analysis.largeTitle': 'File ≥',
      'analysis.allocated': 'Ukuran allocated, bukan logical',
      'analysis.selected': 'dipilih',
      'analysis.selectSafe': 'Pilih aman',
      'analysis.toTrash': 'Pindah ke trash',
      'analysis.trashDup': 'Pindahkan duplikat ke trash',
      'analysis.dupHint': 'size → hash 64 KB pertama → SHA-256 penuh',
      'analysis.staleHint': 'Tanpa folder sistem, kunci, & .git',
      'analysis.artifactsHint': 'node_modules, target, .venv, dist, build, __pycache__',
      'analysis.activeProject': 'proyek < 7 hari tidak disarankan',
      'analysis.file': 'File',
      'analysis.modified': 'Dimodifikasi',
      'analysis.lastChanged': 'Terakhir diubah',
      'analysis.action': 'Aksi',
      'analysis.project': 'Proyek',
      'analysis.artifact': 'Artefak',
      'dashboard.disk': 'Disk /',
      'dashboard.week': 'Minggu ini',
      'dashboard.freed': 'dibebaskan',
      'dashboard.moved': 'dipindah ke trash',
      'dashboard.sessions': 'sesi',
      'dashboard.trash': 'trash',
      'dashboard.export': 'Ekspor',
      'dashboard.nix': 'Nix store',
      'dashboard.cache': 'Cache dev',
      'dashboard.log': 'Log & journal',
      'dashboard.home': 'Pemakaian home',
      'dashboard.reclaim': 'Ruang yang bisa direbut kembali',
      'dashboard.status': 'Status',
      'dashboard.refresh': 'Scan ulang',
      'dashboard.quick': 'Bersihkan cepat',
      'dashboard.quickHint': 'Hanya item berisiko aman yang dipilih. Item ⚠ dan ⛔ selalu off.',
      'dashboard.cleanSafe': 'Bersihkan item aman',
      'dashboard.review': 'Tinjau item',
      'dashboard.gcOff': 'GC otomatis',
      'dashboard.gcOffHint': 'selalu manual + ketik GC',
      'dashboard.trashCross': 'Trash di filesystem berbeda',
      'dashboard.trashCrossHint': '/nix & volume lain → konfirmasi permanen',
      'dashboard.note': 'Scan hanya membaca disk. Semua aksi hapus lewat guard: dry-run, trash, lalu ukur ulang ruang yang benar-benar dibebaskan. GC tetap manual.',
      'common.scanAgain': 'Scan ulang',
      'common.cancel': 'Batal',
      'common.clear': 'Kosongkan',
      'common.open': 'Buka',
      'common.reset': 'Reset',
      'common.save': 'Simpan'
    },
    en: {
      'brand.tagline': 'cleaner for NixOS',
<<<<<<< HEAD
      'brand.tagline.macos': 'cleaner for macOS',
      'side.note': 'M9 · insight & laporan',
=======
      'side.note': 'M10 · automation & coverage',
>>>>>>> c26b989 (feat(m10): aturan kustom, hardlink, dan timer auto-clean)
      'nav.dashboard': 'Dashboard',
      'nav.cleanup': 'Clean',
      'nav.analysis': 'Analysis',
      'nav.history': 'History',
      'nav.direnv': 'direnv',
      'nav.settings': 'Settings',
      'page.dashboard.title': 'Dashboard',
      'page.dashboard.sub': 'Disk overview and reclaimable space',
      'page.cleanup.title': 'Clean',
      'page.cleanup.sub': 'Pick items by category · safe items only by default',
      'page.nix.title': 'Nix',
      'page.nix.sub': 'Store, generations, and garbage collection',
      'page.android.title': 'Android',
      'page.android.sub': 'AVDs, system images, and NDK',
      'page.analysis.title': 'Analysis',
      'page.analysis.sub': 'Disk usage, large files, duplicates, stale files, build artifacts',
      'page.direnv.title': 'direnv',
      'page.direnv.sub': 'Find heavy .direnv folders that can be regenerated',
      'page.history.title': 'History',
      'page.history.sub': 'Cleanup activity and trash contents',
      'history.tab.activity': 'Activity',
      'history.tab.trash': 'Trash',
      'history.refresh': 'Reload',
      'history.openTrash': 'Open trash folder',
      'history.openFolder': 'Open folder',
      'history.sessions': 'Cleanup sessions',
      'history.sessionsHint': 'One session = one audit batch; items still in trash can be restored.',
      'history.time': 'Time',
      'history.action': 'Action',
      'history.items': 'Items',
      'history.freed': 'Freed',
      'history.undoable': 'Undoable',
      'history.actions': 'Actions',
      'history.trashTitle': 'Trash contents',
      'history.trashHint': 'Sizes are measured in the background. Restore only items without conflicts.',
      'history.search': 'Search name or path…',
      'history.restore': 'Restore',
      'history.delete': 'Delete permanently',
      'history.empty': 'Empty…',
      'history.name': 'Name',
      'history.origin': 'Origin',
      'history.deletedAt': 'Deleted',
      'history.size': 'Size',
      'history.state': 'Status',
      'history.undo': 'Undo this session',
      'history.preview': 'Preview',
      'history.selected': 'selected',
      'history.noSessions': 'No cleanup activity yet.',
      'history.noTrash': 'Trash is empty.',
      'history.notRestorable': 'not restorable',
      'history.emptyConfirm': 'Type KOSONGKAN or EMPTY to empty the whole trash. This cannot be undone.',
      'history.deleteConfirm': 'Permanently delete selected items? This cannot be undone.',
      'history.result': 'Result',
      'history.close': 'Close',
      'history.blocked': 'process blocked',
      'history.truncated': 'List truncated; use search to narrow it down.',
      'history.emptyOk': 'Empty',
      'history.deleteOk': 'Delete permanently',
      'history.restoreOk': 'Restore',
      'history.export': 'Export report',
      'settings.openHistory': 'View history',
      'settings.openFolder': 'Folder',
      'direnv.roots': 'Project folders',
      'direnv.rootsHint': 'max 32',
      'direnv.rootsNote': 'Click Choose folder… or type a path. Cleanlie looks for .direnv inside; nix-direnv profiles can be large.',
      'direnv.pick': 'Choose folder…',
      'direnv.add': '+ Add',
      'direnv.scan': 'Scan direnv',
      'direnv.cancel': 'Cancel',
      'direnv.refresh': 'Scan again',
      'direnv.summary': 'Summary',
      'direnv.cacheNote': 'Global caches ~/.cache/direnv and ~/.cache/nix-direnv are measured too. nix-direnv is mostly symlinks; the real space is freed by Nix GC.',
      'direnv.table': '.direnv folders',
      'direnv.tableHint': 'Allocated size, symlinks not followed',
      'direnv.selectSafe': 'Select safe',
      'direnv.trash': 'Move to trash',
      'direnv.project': 'Project',
      'direnv.location': 'Location',
      'direnv.size': 'Size',
      'direnv.changed': 'Last changed',
      'direnv.state': 'Status',
      'page.settings.title': 'Settings',
      'page.settings.sub': 'Cleaning and appearance preferences',
      'theme.system': 'System',
      'theme.light': 'Light',
      'theme.dark': 'Dark',
      'settings.appearance': 'Appearance',
      'settings.theme': 'Theme',
      'settings.themeHint': 'Follow the system or force light/dark',
      'settings.language': 'Language',
      'settings.languageHint': 'Labels, dates, and number formatting',
      'settings.showRisky': 'Show risky items',
      'settings.showRiskyHint': '⚠ and ⛔ items are never selected automatically',
      'settings.cleanup': 'Cleaning',
      'settings.deleteMode': 'Default delete mode',
      'settings.deleteModeHint': 'Trash can be restored from the file manager',
      'settings.trash': 'Trash (restorable)',
      'settings.permanent': 'Delete permanently',
      'settings.autoSafe': 'Auto-select safe items',
      'settings.autoSafeHint': '⚠/⛔ items are never auto-selected',
      'settings.notify': 'Desktop notifications',
      'settings.notifyHint': 'Notify when cleaning or GC finishes',
      'settings.journalTarget': 'Journal target',
      'settings.journalTargetHint': 'Vacuum system and user journal',
      'settings.analysis': 'Analysis',
      'settings.largeThreshold': 'Large file threshold',
      'settings.largeThresholdHint': 'Used by the Large files tab',
      'settings.staleDays': 'Stale file threshold',
      'settings.staleDaysHint': 'mtime older than',
      'settings.days': 'days',
      'settings.artifactRoots': 'Extra scan roots',
      'settings.artifactRootsHint': 'Comma separated; used for artifact scans',
      'settings.customRules': 'Custom rules',
      'settings.customRulesLabel': 'Extra cache folders',
      'settings.customRulesHint': 'One line: path | label | risk | whole',
      'settings.customRulesNote': 'risk: safe/warn/danger · whole optional · max 32 rules',
      'settings.automation': 'Automation',
      'settings.scheduleInterval': 'Auto-clean schedule',
      'settings.scheduleIntervalHint': 'User systemd timer; safe items to trash only',
      'settings.weekly': 'Weekly',
      'settings.daily': 'Daily',
      'settings.timer': 'Timer',
      'settings.installTimer': 'Install',
      'settings.removeTimer': 'Remove',
      'settings.autoCleanNote': 'Auto-clean uses --yes, trash mode, and never deletes permanently.',
      'settings.timerInstalled': 'installed',
      'settings.timerActive': 'active',
      'settings.timerOff': 'not installed',
      'settings.timerFailed': 'Could not configure timer: ',
      'settings.timerConfirm': 'Remove the auto-clean timer?',
      'settings.exclusions': 'Exclusions & access',
      'settings.excluded': 'Excluded paths',
      'settings.excludedHint': 'Never scanned or deleted',
      'settings.rootAction': 'Root actions',
      'settings.rootActionHint': 'How root actions are prompted in this session',
      'settings.history': 'History',
      'settings.historyHint': 'Audit log for every delete',
      'settings.open': 'Open',
      'settings.save': 'Save settings',
      'settings.reset': 'Restore defaults',
      'settings.configPath': 'Config',
      'settings.runtime': 'Runtime',
      'settings.version': 'Version',
      'settings.commit': 'Commit',
      'settings.platform': 'Platform',
      'settings.saved': 'Settings saved.',
      'settings.saving': 'Saving…',
      'settings.dirty': 'Unsaved changes.',
      'settings.loadFailed': 'Could not read settings: ',
      'settings.saveFailed': 'Could not save: ',
      'settings.preview': 'Preview mode: changes are not written to disk.',
      'settings.warning': 'Old config: ',
      'settings.polkit': 'Polkit',
      'settings.polkitYes': 'agent detected',
      'settings.polkitNo': 'no agent',
      'settings.adminOsascript': 'macOS admin dialog',
      'analysis.tab.tree': 'Disk usage',
      'analysis.tab.large': 'Large files',
      'analysis.tab.dup': 'Duplicates',
      'analysis.tab.stale': 'Stale files',
      'analysis.tab.artifacts': 'Build artifacts',
      'analysis.scanAgain': 'Scan again',
      'analysis.openFolder': 'Open in file manager',
      'analysis.folder': 'Folder',
      'analysis.size': 'Size',
      'analysis.share': 'Share',
      'analysis.map': 'Folder map',
      'analysis.mapHint': 'click to enter',
      'analysis.largeTitle': 'Files ≥',
      'analysis.allocated': 'Allocated size, not logical',
      'analysis.selected': 'selected',
      'analysis.selectSafe': 'Select safe',
      'analysis.toTrash': 'Move to trash',
      'analysis.trashDup': 'Move duplicates to trash',
      'analysis.dupHint': 'size → first 64 KB hash → full SHA-256',
      'analysis.staleHint': 'System folders, keys, & .git excluded',
      'analysis.artifactsHint': 'node_modules, target, .venv, dist, build, __pycache__',
      'analysis.activeProject': 'projects < 7 days are not suggested',
      'analysis.file': 'File',
      'analysis.modified': 'Modified',
      'analysis.lastChanged': 'Last changed',
      'analysis.action': 'Action',
      'analysis.project': 'Project',
      'analysis.artifact': 'Artifact',
      'dashboard.disk': 'Disk /',
      'dashboard.week': 'This week',
      'dashboard.freed': 'freed',
      'dashboard.moved': 'moved to trash',
      'dashboard.sessions': 'sessions',
      'dashboard.trash': 'trash',
      'dashboard.export': 'Export',
      'dashboard.nix': 'Nix store',
      'dashboard.cache': 'Dev cache',
      'dashboard.log': 'Logs & journal',
      'dashboard.home': 'Home usage',
      'dashboard.reclaim': 'Reclaimable space',
      'dashboard.status': 'Status',
      'dashboard.refresh': 'Scan again',
      'dashboard.quick': 'Quick clean',
      'dashboard.quickHint': 'Only safe items are selected. ⚠ and ⛔ items always stay off.',
      'dashboard.cleanSafe': 'Clean safe items',
      'dashboard.review': 'Review items',
      'dashboard.gcOff': 'Automatic GC',
      'dashboard.gcOffHint': 'always manual + type GC',
      'dashboard.trashCross': 'Trash is on a different filesystem',
      'dashboard.trashCrossHint': '/nix & other volumes → permanent confirmation',
      'dashboard.note': 'Scans are read-only. Every delete goes through the guard: dry-run, trash, then re-measure the space actually freed. GC stays manual.',
      'common.scanAgain': 'Scan again',
      'common.cancel': 'Cancel',
      'common.clear': 'Clear',
      'common.open': 'Open',
      'common.reset': 'Reset',
      'common.save': 'Save'
    }
  };

  let LANG = 'id';
  window.__t = function (key) {
    const table = STRINGS[LANG] || STRINGS.id;
    return table[key] || (STRINGS.id[key] || null);
  };
  const t = key => window.__t(key) || key;

  const DEFAULTS = {
    theme: 'system', language: 'id', showRisky: false, deleteMode: 'trash',
    autoSelectSafe: true, showRisky: true, notify: true, journalTarget: '500M', largeBytes: 100 * MIB,
    staleDays: 365, artifactRoots: [], direnvRoots: [], excluded: [], customRules: []
  };
  let state = Object.assign({}, DEFAULTS);
  let view = null;
  let warning = '';

  function isMac() {
    return String(window.__cleanliePlatform || '').startsWith('darwin');
  }

  function applyLanguage(lang) {
    LANG = lang === 'en' ? 'en' : 'id';
    document.documentElement.lang = LANG;
    document.querySelectorAll('[data-i18n]').forEach(el => {
      let key = el.dataset.i18n;
      if (key === 'brand.tagline' && isMac()) key = 'brand.tagline.macos';
      const text = t(key);
      if (text) el.textContent = text;
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
      const text = t(el.dataset.i18nPlaceholder);
      if (text) el.placeholder = text;
    });
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
      const text = t(el.dataset.i18nTitle);
      if (text) el.title = text;
    });
    if (typeof current === 'string') {
      const title = t('page.' + current + '.title');
      const sub = t('page.' + current + '.sub');
      const tEl = document.getElementById('page-title');
      const sEl = document.getElementById('page-sub');
      if (tEl && title) tEl.textContent = title;
      if (sEl && sub) sEl.textContent = sub;
    }
  }

  function shortPath(p) {
    if (window.cleanlie && window.cleanlie.shortPath) return window.cleanlie.shortPath(p);
    return p || '';
  }

  function setToggle(el, on) {
    if (!el) return;
    el.classList.toggle('on', !!on);
    el.setAttribute('aria-checked', on ? 'true' : 'false');
  }

  function renderExcluded() {
    const box = byId('excluded-list');
    if (!box) return;
    box.innerHTML = state.excluded.length
      ? state.excluded.map((p, i) =>
          '<span class="tag"><span class="mono">' + esc(shortPath(p)) + '</span>' +
          '<button type="button" data-rm-excluded="' + i + '" aria-label="Hapus ' + esc(p) + '">×</button></span>'
        ).join('')
      : '<span class="tiny dim">Belum ada path dikecualikan.</span>';
  }

  function setWarning(text) {
    warning = text || '';
    const box = byId('settings-warning');
    const txt = byId('settings-warning-text');
    if (!box || !txt) return;
    box.hidden = !warning;
    txt.textContent = warning;
  }

  function customRulesText(list) {
    return (list || []).map(r => {
      const parts = [r.path || '', r.label || '', r.risk || 'warn'];
      if (r.whole) parts.push('whole');
      return parts.join(' | ');
    }).join('\n');
  }

  function parseCustomRules(text) {
    return String(text || '').split('\n').map(line => line.trim()).filter(Boolean).map(line => {
      const p = line.split('|').map(s => s.trim());
      const whole = /^(whole|folder|1|true|yes)$/i.test(p[3] || '');
      return { path: p[0] || '', label: p[1] || '', risk: (p[2] || 'warn').toLowerCase(), whole: whole };
    }).filter(r => r.path);
  }

  function renderSchedule(st) {
    const chip = byId('schedule-state');
    const detail = byId('schedule-detail');
    const interval = byId('schedule-interval');
    if (!st) return;
    if (interval && st.interval) interval.value = st.interval;
    if (chip) {
      chip.textContent = st.installed ? t('settings.timerInstalled') + (st.active ? ' · ' + t('settings.timerActive') : '') : t('settings.timerOff');
      chip.className = 'chip ' + (st.installed ? (st.active ? 'safe' : 'warn') : '');
    }
    if (detail) detail.textContent = st.detail || '';
  }

  function fillSettings() {
    const theme = byId('theme-select');
    const lang = byId('lang-select');
    if (theme) theme.value = state.theme;
    if (lang) lang.value = state.language;
    setToggle(byId('show-risky'), state.showRisky);
    setToggle(byId('auto-safe'), state.autoSelectSafe);
    setToggle(byId('notify-desktop'), state.notify);
    const del = byId('delete-mode');
    if (del) del.value = state.deleteMode;
    const journal = byId('journal-target');
    if (journal) journal.value = state.journalTarget;
    const large = byId('large-mb');
    if (large) large.value = String(Math.round((state.largeBytes || 0) / MIB));
    const stale = byId('stale-days');
    if (stale) stale.value = String(state.staleDays);
    const jt = byId('nix-journal-target');
    if (jt) {
      const v = String(state.journalTarget || '500M').toUpperCase().replace(/\s+/g, '').replace(/B$/, '');
      if ([...jt.options].some(o => o.value === v)) jt.value = v;
    }
    const roots = byId('artifact-roots');
    if (roots) roots.value = (state.artifactRoots || []).map(shortPath).join(', ');
    const custom = byId('custom-rules');
    if (custom) custom.value = customRulesText(state.customRules);
    renderExcluded();
  }

  function applyView(v, message) {
    view = v || {};
    const c = view.config || DEFAULTS;
    state = Object.assign({}, DEFAULTS, c);
    window.__getSettings = () => state;
    state.artifactRoots = Array.isArray(c.artifactRoots) ? c.artifactRoots.slice() : [];
    state.direnvRoots = Array.isArray(c.direnvRoots) ? c.direnvRoots.slice() : [];
    state.excluded = Array.isArray(c.excluded) ? c.excluded.slice() : [];
    state.customRules = Array.isArray(c.customRules) ? c.customRules.slice() : [];
    setTheme(state.theme);
    fillSettings();
    applyLanguage(state.language);
    const pathEl = byId('settings-path');
    if (pathEl) pathEl.textContent = view.path || '~/.config/cleanlie/config.json';
    const ver = byId('settings-version');
    if (ver && window.__cleanlieVersion) ver.textContent = 'v' + window.__cleanlieVersion;
    setWarning(view.warning ? t('settings.warning') + view.warning : '');
    const status = byId('settings-status');
    if (status) status.textContent = message || '';
  }

  function markDirty() {
    const status = byId('settings-status');
    if (status) status.textContent = t('settings.dirty');
  }

  function readForm() {
    const num = (id, fallback) => {
      const n = Number(byId(id) && byId(id).value);
      return Number.isFinite(n) && n > 0 ? n : fallback;
    };
    return {
      theme: (byId('theme-select') && byId('theme-select').value) || state.theme,
      language: (byId('lang-select') && byId('lang-select').value) || state.language,
      showRisky: !!(byId('show-risky') && byId('show-risky').classList.contains('on')),
      deleteMode: (byId('delete-mode') && byId('delete-mode').value) || state.deleteMode,
      autoSelectSafe: !!(byId('auto-safe') && byId('auto-safe').classList.contains('on')),
      notify: !!(byId('notify-desktop') && byId('notify-desktop').classList.contains('on')),
      journalTarget: ((byId('journal-target') && byId('journal-target').value) || '500M').trim(),
      largeBytes: Math.round(num('large-mb', 100) * MIB),
      staleDays: Math.round(num('stale-days', 365)),
      artifactRoots: ((byId('artifact-roots') && byId('artifact-roots').value) || '')
        .split(',').map(x => x.trim()).filter(Boolean),
      direnvRoots: (state.direnvRoots || []).slice(),
      excluded: state.excluded.slice(),
      customRules: parseCustomRules(byId('custom-rules') && byId('custom-rules').value)
    };
  }

  async function loadSettings() {
    if (!bridge || !bridge.Settings) {
      applyView({ config: DEFAULTS, path: '~/.config/cleanlie/config.json' }, t('settings.preview'));
      return;
    }
    try {
      applyView(await bridge.Settings());
    } catch (e) {
      applyView({ config: DEFAULTS, path: '~/.config/cleanlie/config.json' }, t('settings.loadFailed') + errMsg(e));
    }
    try {
      const d = await bridge.Diagnostics();
      window.__cleanlieVersion = d.version;
      const ver = byId('settings-version');
      if (ver) ver.textContent = 'v' + d.version;
      const chip = byId('settings-polkit');
      const detail = byId('settings-polkit-detail');
      if (chip) {
        chip.textContent = d.adminKind === 'osascript'
          ? t('settings.adminOsascript')
          : (d.polkitAgent ? t('settings.polkitYes') : t('settings.polkitNo'));
        chip.className = 'chip ' + (d.polkitAgent ? 'safe' : 'warn');
      }
      if (detail) detail.textContent = d.polkitDetail || '';
      document.querySelectorAll('[data-i18n="settings.polkit"]').forEach(el => el.textContent = t('settings.polkit'));
      if (isMac()) applyLanguage(LANG);
    } catch (e) { /* diagnostik opsional */ }
    try {
      if (bridge.ScheduleStatus) renderSchedule(await bridge.ScheduleStatus());
    } catch (e) { /* status timer opsional */ }
  }

  async function saveSettings(reset) {
    const btn = byId('settings-save');
    const status = byId('settings-status');
    if (!bridge || !(bridge.SaveSettings || bridge.ResetSettings)) {
      if (status) status.textContent = t('settings.preview');
      return;
    }
    if (btn) btn.disabled = true;
    if (status) status.textContent = t('settings.saving');
    try {
      const next = reset ? await bridge.ResetSettings() : await bridge.SaveSettings(readForm());
      applyView(next, t('settings.saved'));
    } catch (e) {
      if (status) status.textContent = t('settings.saveFailed') + errMsg(e);
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  function addExcluded() {
    const input = byId('excluded-input');
    if (!input) return;
    const p = input.value.trim();
    if (!p) return;
    if (!state.excluded.includes(p)) state.excluded.push(p);
    input.value = '';
    renderExcluded();
    markDirty();
  }

  async function installTimer() {
    const btn = byId('schedule-install');
    const interval = (byId('schedule-interval') && byId('schedule-interval').value) || 'weekly';
    if (!bridge || !bridge.ScheduleInstall) return;
    if (btn) btn.disabled = true;
    try {
      renderSchedule(await bridge.ScheduleInstall(interval));
      toast(t('settings.saved'));
    } catch (e) {
      toast(t('settings.timerFailed') + errMsg(e));
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  async function removeTimer() {
    if (!bridge || !bridge.ScheduleRemove) return;
    if (!window.confirm(t('settings.timerConfirm'))) return;
    const btn = byId('schedule-remove');
    if (btn) btn.disabled = true;
    try {
      renderSchedule(await bridge.ScheduleRemove());
      toast(t('settings.saved'));
    } catch (e) {
      toast(t('settings.timerFailed') + errMsg(e));
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  function bind() {
    if (!byId('view-settings')) return;
    const theme = byId('theme-select');
    if (theme) theme.addEventListener('change', markDirty);
    const lang = byId('lang-select');
    if (lang) lang.addEventListener('change', () => {
      state.language = lang.value;
      applyLanguage(lang.value);
      markDirty();
    });
    ['show-risky', 'auto-safe', 'notify-desktop'].forEach(id => {
      const el = byId(id);
      if (el) el.addEventListener('click', () => {
        setToggle(el, !el.classList.contains('on'));
        markDirty();
      });
    });
    const del = byId('delete-mode');
    if (del) del.addEventListener('change', markDirty);
    ['journal-target', 'large-mb', 'stale-days', 'artifact-roots', 'custom-rules'].forEach(id => {
      const el = byId(id);
      if (el) el.addEventListener('input', markDirty);
    });
    const schedInst = byId('schedule-install');
    if (schedInst) schedInst.addEventListener('click', installTimer);
    const schedDel = byId('schedule-remove');
    if (schedDel) schedDel.addEventListener('click', removeTimer);
    const add = byId('excluded-add');
    if (add) add.addEventListener('click', addExcluded);
    const input = byId('excluded-input');
    if (input) input.addEventListener('keydown', e => { if (e.key === 'Enter') { e.preventDefault(); addExcluded(); } });
    const list = byId('excluded-list');
    if (list) list.addEventListener('click', e => {
      const btn = e.target.closest('[data-rm-excluded]');
      if (!btn) return;
      state.excluded.splice(Number(btn.dataset.rmExcluded), 1);
      renderExcluded();
      markDirty();
    });
    const save = byId('settings-save');
    if (save) save.addEventListener('click', () => saveSettings(false));
    const reset = byId('settings-reset');
    if (reset) reset.addEventListener('click', () => saveSettings(true));
    const hist = byId('history-open');
    if (hist) hist.addEventListener('click', () => {
      navigate('history');
    });
    const folder = byId('history-folder');
    if (folder) folder.addEventListener('click', () => {
      if (bridge) bridge.OpenInFileManager('~/.local/state/cleanlie').catch(e => toast(t('settings.saveFailed') + errMsg(e)));
    });
    window.__themeChanged = mode => { state.theme = mode; markDirty(); };
  }

  window.addEventListener('DOMContentLoaded', () => {
    bind();
    loadSettings();
  });
  if (document.readyState !== 'loading') {
    bind();
    loadSettings();
  }
})();

