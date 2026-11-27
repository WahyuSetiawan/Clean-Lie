/*
 * Fixture `window.go` untuk screenshot README.
 *
 * Di-load SEBELUM ui/demo.js / ui/core.js oleh shoot.mjs, sehingga UI berjalan
 * persis seperti dengan backend Go asli, tapi memakai data contoh (bukan data
 * mesin siapa pun). Angka & path di bawah ini sintetis.
 */
(() => {
  const MiB = 1024 * 1024;
  const GiB = 1024 * MiB;
  const HOME = '/home/wahyu';
  const NOW = '2026-11-27';

  const item = (id, group, path, bytes, risk, extra = {}) => ({
    id, group, path, root: path.split('/').slice(0, -1).join('/') || path,
    whole: true, label: path.split('/').pop(), bytes, files: Math.max(1, Math.round(bytes / (MiB * 3))),
    risk, fallbackDelete: true, ...extra,
  });

  const groups = [
    {
      id: 'dev.node', label: 'Node (npm/pnpm)', risk: 'warn', bytes: 15.6 * GiB, files: 214_302,
      items: [
        item('npm-cacache', 'dev.node', '~/.npm/_cacache', 7.0 * GiB, 'safe', { commandLine: 'npm cache clean --force', note: 'cache tarball npm, regenerate' }),
        item('npm-npx', 'dev.node', '~/.npm/_npx', 2.6 * GiB, 'safe', { note: 'cache paket npx, regenerate' }),
        item('pnpm-cache', 'dev.node', '~/.cache/pnpm', 2.1 * GiB, 'safe', { commandLine: 'pnpm store prune', note: 'metadata & tarball pnpm' }),
        item('pnpm-store', 'dev.node', '~/.local/share/pnpm/store', 1.2 * GiB, 'safe', { commandLine: 'pnpm store prune', note: 'store konten pnpm; pakai store prune, jangan rm' }),
        item('playwright', 'dev.node', '~/.cache/ms-playwright', 1.9 * GiB, 'warn', { note: 'browser test Playwright; harus diunduh ulang bila dihapus' }),
        item('cypress', 'dev.node', '~/.cache/Cypress', 819 * MiB, 'warn', { note: 'binary Cypress per versi; cek package.json sebelum hapus' }),
      ],
    },
    {
      id: 'dev.jvm', label: 'JVM (Gradle)', risk: 'safe', bytes: 6.3 * GiB, files: 88_140,
      items: [
        item('gradle-caches', 'dev.jvm', '~/.gradle/caches', 3.5 * GiB, 'safe', { note: 'cache dependency Gradle per versi; build berikutnya re-download' }),
        item('gradle-wrapper', 'dev.jvm', '~/.gradle/wrapper/dists', 1.0 * GiB, 'safe', { note: 'distribusi Gradle lama yang diunduh wrapper' }),
        item('maven', 'dev.jvm', '~/.m2/repository', 1.8 * GiB, 'safe', { note: 'cache Maven lokal, re-download saat build' }),
      ],
    },
    {
      id: 'dev.python', label: 'Python (uv/pip)', risk: 'safe', bytes: 1.9 * GiB, files: 41_002,
      items: [
        item('uv-cache', 'dev.python', '~/.cache/uv', 1.5 * GiB, 'safe', { commandLine: 'uv cache prune', note: 'cache wheel/sdist uv' }),
        item('uv-share', 'dev.python', '~/.local/share/uv', 307 * MiB, 'safe', { commandLine: 'uv cache prune', note: 'cache python terkelola uv' }),
        item('pip-cache', 'dev.python', '~/.cache/pip', 102 * MiB, 'safe', { commandLine: 'pip cache purge', note: 'cache wheel/sdist pip' }),
      ],
    },
    {
      id: 'dev.dart', label: 'Dart / Flutter', risk: 'safe', bytes: 1.5 * GiB, files: 30_204,
      items: [
        item('pub-cache', 'dev.dart', '~/.pub-cache/hosted', 1.5 * GiB, 'safe', { commandLine: 'dart pub cache clean', note: 'paket pub; diunduh ulang saat pub get' }),
      ],
    },
    {
      id: 'dev.rustgo', label: 'Rust / Go', risk: 'safe', bytes: 2.0 * GiB, files: 52_388,
      items: [
        item('cargo-registry', 'dev.rustgo', '~/.cargo/registry', 512 * MiB, 'safe', { note: 'cache crate; bin/ dan config tidak disentuh' }),
        item('go-modcache', 'dev.rustgo', '~/go/pkg/mod', 410 * MiB, 'safe', { commandLine: 'go clean -modcache', note: 'modul Go; diunduh ulang saat build' }),
        item('go-build', 'dev.rustgo', '~/.cache/go-build', 1.1 * GiB, 'safe', { commandLine: 'go clean -cache', note: 'cache kompilasi Go' }),
        item('gopls', 'dev.rustgo', '~/.cache/gopls', 20 * MiB, 'safe', { note: 'cache language server gopls' }),
      ],
    },
    {
      id: 'dev.android', label: 'Android SDK/AVD', risk: 'danger', bytes: 27.9 * GiB, files: 161_402,
      items: [
        item('android-avd', 'dev.android', '~/.android/avd', 14.2 * GiB, 'danger', { note: '2 AVD — data & snapshot hilang' }),
        item('android-sysimg', 'dev.android', '~/Android/Sdk/system-images', 7.8 * GiB, 'danger', { commandLine: 'sdkmanager --uninstall "system-images;android-30;..."', note: 'system image lama' }),
        item('android-ndk', 'dev.android', '~/Android/Sdk/ndk', 4.1 * GiB, 'danger', { note: 'build native gagal sampai re-install' }),
        item('android-emulator', 'dev.android', '~/Android/Sdk/emulator', 819 * MiB, 'danger', { note: 'emulator binary, unduh ulang' }),
        item('android-platforms', 'dev.android', '~/Android/Sdk/platforms', 717 * MiB, 'danger', { note: 'platform API, dipakai compileSdk' }),
        item('android-buildtools', 'dev.android', '~/Android/Sdk/build-tools', 410 * MiB, 'warn', { note: 'versi lama bisa dihapus' }),
      ],
    },
    {
      id: 'app', label: 'Cache aplikasi', risk: 'warn', bytes: 10.5 * GiB, files: 190_233,
      items: [
        item('app-opencode', 'app', '~/.cache/opencode', 3.0 * GiB, 'safe', { note: 'session & model cache, regenerate' }),
        item('app-qmd', 'app', '~/.cache/qmd', 2.3 * GiB, 'safe', { note: 'index embedding, regenerate' }),
        item('app-phpactor', 'app', '~/.cache/phpactor', 2.2 * GiB, 'safe', { note: 'index PHP, regenerate' }),
        item('app-codebase-memory', 'app', '~/.cache/codebase-memory-mcp', 1.0 * GiB, 'safe', { note: 'index kode, regenerate' }),
        item('app-nix-cache', 'app', '~/.cache/nix', 905 * MiB, 'safe', { note: 'eval cache Nix, regenerate' }),
        item('app-brave', 'app', '~/.cache/BraveSoftware', 1.2 * GiB, 'blocked', { note: 'Brave sedang berjalan — tutup dulu' }),
      ],
    },
    {
      id: 'sys.log', label: 'Log & journal', risk: 'warn', bytes: 4.3 * GiB, files: 12_004,
      items: [
        item('journal', 'sys.log', '/var/log/journal', 4.0 * GiB, 'warn', { requiresRoot: true, commandLine: 'journalctl --vacuum-size=500M', note: 'vacuum journal ke target (butuh root)' }),
        item('log-rotated', 'sys.log', '/var/log', 320 * MiB, 'safe', { note: 'file log yang sudah dirotasi' }),
      ],
    },
    {
      id: 'trash', label: 'Trash', risk: 'warn', bytes: 2.1 * GiB, files: 3_402,
      items: [
        item('trash', 'trash', '~/.local/share/Trash/files', 2.1 * GiB, 'warn', { note: 'setelah dikosongkan tidak bisa di-restore' }),
      ],
    },
  ];

  const report = {
    scannedAt: NOW + 'T10:12:04+07:00',
    durationMs: 4_820,
    bytes: groups.reduce((a, g) => a + g.bytes, 0),
    files: groups.reduce((a, g) => a + g.files, 0),
    groups,
  };

  const tree = {
    depth: 1,
    bytes: 186.4 * GiB,
    files: 1_284_402,
    dirs: 41_233,
    errors: 0,
    scannedAt: NOW + 'T10:12:02+07:00',
    durationMs: 1_240,
    root: {
      name: 'wahyu', path: HOME, bytes: 186.4 * GiB, files: 1_284_402, dir: true,
      children: [
        { name: 'Projects', path: HOME + '/Projects', bytes: 61.2 * GiB, files: 402_112, dir: true },
        { name: 'Android', path: HOME + '/Android', bytes: 28.2 * GiB, files: 161_402, dir: true },
        { name: '.cache', path: HOME + '/.cache', bytes: 24.8 * GiB, files: 190_233, dir: true },
        { name: '.local', path: HOME + '/.local', bytes: 9.4 * GiB, files: 88_040, dir: true },
        { name: 'Downloads', path: HOME + '/Downloads', bytes: 6.1 * GiB, files: 12_402, dir: true },
        { name: 'go', path: HOME + '/go', bytes: 4.2 * GiB, files: 61_220, dir: true },
        { name: '.npm', path: HOME + '/.npm', bytes: 3.1 * GiB, files: 71_002, dir: true },
        { name: '.gradle', path: HOME + '/.gradle', bytes: 2.8 * GiB, files: 44_120, dir: true },
        { name: '.pub-cache', path: HOME + '/.pub-cache', bytes: 1.5 * GiB, files: 30_204, dir: true },
        { name: 'Documents', path: HOME + '/Documents', bytes: 1.2 * GiB, files: 8_412, dir: true },
        { name: 'Pictures', path: HOME + '/Pictures', bytes: 1.1 * GiB, files: 6_004, dir: true },
        { name: '.config', path: HOME + '/.config', bytes: 900 * MiB, files: 21_040, dir: true },
        { name: 'lain-lain', path: HOME + '/.var', bytes: 41.9 * GiB, files: 218_000, dir: true },
      ],
    },
  };

  const nix = {
    storePath: '/nix/store',
    storeBytes: 82.0 * GiB,
    storePaths: 412_345,
    freeBytes: 120.0 * GiB,
    totalBytes: 512.0 * GiB,
    journalBytes: 4.0 * GiB,
    agent: true,
    gcCommand: 'pkexec nix-collect-garbage -d',
    scannedAt: NOW + 'T10:12:03+07:00',
    durationMs: 2_140,
    profiles: [
      { name: 'system', path: '/nix/var/nix/profiles/system', count: 4, current: 412, oldest: '2026-08-14 21:04:11' },
      { name: 'home-manager', path: HOME + '/.local/state/nix/profiles/home-manager', count: 2, current: 233, oldest: '2026-10-02 09:12:40' },
      { name: 'user', path: HOME + '/.local/state/nix/profiles/profile', count: 0, error: 'tidak ada generasi' },
    ],
    generations: [
      { profile: 'system', profilePath: '/nix/var/nix/profiles/system', number: 412, date: '2026-11-24 20:41:02', ageDays: 3, current: true, closureBytes: 21.4 * GiB },
      { profile: 'system', profilePath: '/nix/var/nix/profiles/system', number: 411, date: '2026-11-10 18:22:51', ageDays: 17, suggested: true, closureBytes: 20.9 * GiB },
      { profile: 'system', profilePath: '/nix/var/nix/profiles/system', number: 408, date: '2026-10-06 07:15:33', ageDays: 52, suggested: true, closureBytes: 20.1 * GiB },
      { profile: 'system', profilePath: '/nix/var/nix/profiles/system', number: 402, date: '2026-08-14 21:04:11', ageDays: 105, suggested: true, closureBytes: 19.6 * GiB },
      { profile: 'home-manager', profilePath: HOME + '/.local/state/nix/profiles/home-manager', number: 233, date: '2026-11-22 09:03:12', ageDays: 5, current: true, closureBytes: 3.2 * GiB },
      { profile: 'home-manager', profilePath: HOME + '/.local/state/nix/profiles/home-manager', number: 232, date: '2026-10-02 09:12:40', ageDays: 56, suggested: true, closureBytes: 3.1 * GiB },
    ],
  };

  const dead = {
    count: 12_480,
    bytes: 34.0 * GiB,
    source: 'nix-store --gc --print-dead',
    scannedAt: NOW + 'T10:12:06+07:00',
    durationMs: 21_400,
  };

  const overview = {
    root: HOME,
    files: 1_284_402,
    dirs: 41_233,
    errors: 0,
    scannedAt: NOW + 'T10:12:05+07:00',
    durationMs: 3_120,
    large: [
      { path: HOME + '/Downloads/ubuntu-24.04.3-desktop-amd64.iso', bytes: 5.7 * GiB, modTime: '2026-09-02 14:20' },
      { path: HOME + '/Videos/render/final-cut.mp4', bytes: 3.9 * GiB, modTime: '2026-11-01 09:12' },
      { path: HOME + '/Projects/lab/data/dump.sql', bytes: 2.4 * GiB, modTime: '2026-10-19 22:41' },
      { path: HOME + '/Downloads/android-studio-2025.1.tar.gz', bytes: 1.6 * GiB, modTime: '2026-08-21 11:02' },
      { path: HOME + '/Projects/lab/vm/disk.qcow2', bytes: 1.2 * GiB, modTime: '2026-11-12 18:33' },
    ],
    stale: [
      { path: HOME + '/Projects/legacy/backup-2024.zip', bytes: 812 * MiB, modTime: '2024-03-11 10:04', ageDays: 991 },
      { path: HOME + '/Downloads/receipt-2023.pdf', bytes: 4 * MiB, modTime: '2023-12-30 08:21', ageDays: 1_063 },
      { path: HOME + '/Projects/lab/old-assets.tar', bytes: 340 * MiB, modTime: '2024-07-02 16:45', ageDays: 878 },
      { path: HOME + '/Documents/export-lama.csv', bytes: 96 * MiB, modTime: '2024-01-19 13:10', ageDays: 1_043 },
    ],
    artifacts: [
      { project: HOME + '/Projects/voi-android-app', name: 'build/', path: HOME + '/Projects/voi-android-app/build', bytes: 6.4 * GiB, modTime: '2026-11-20 19:04', ageDays: 7, active: true },
      { project: HOME + '/Projects/cleanlie', name: 'node_modules/', path: HOME + '/Projects/cleanlie/node_modules', bytes: 1.8 * GiB, modTime: '2026-11-25 22:12', ageDays: 2, active: true },
      { project: HOME + '/Projects/web-scraper', name: 'node_modules/', path: HOME + '/Projects/web-scraper/node_modules', bytes: 940 * MiB, modTime: '2026-08-14 10:31', ageDays: 105 },
      { project: HOME + '/Projects/rust-cli', name: 'target/', path: HOME + '/Projects/rust-cli/target', bytes: 2.2 * GiB, modTime: '2026-09-28 20:02', ageDays: 60 },
      { project: HOME + '/Projects/cleanlie', name: 'frontend/dist/assets/', path: HOME + '/Projects/cleanlie/frontend/dist/assets', bytes: 210 * MiB, modTime: '2026-11-26 08:44', ageDays: 1, active: true },
      { project: HOME + '/Projects/legacy', name: 'dist/', path: HOME + '/Projects/legacy/dist', bytes: 410 * MiB, modTime: '2025-02-11 17:20', ageDays: 654, protected: true },
    ],
  };

  const dups = {
    root: HOME,
    groupCount: 3,
    wasted: 3.1 * GiB,
    candidates: 4_120,
    hashed: 812,
    errors: 0,
    scannedAt: NOW + 'T10:12:08+07:00',
    durationMs: 2_880,
    groups: [
      {
        size: 1.4 * GiB, hash: 'sha256:9f2c…a41', wasted: 1.4 * GiB,
        files: [
          { path: HOME + '/Downloads/ubuntu-24.04.iso', bytes: 1.4 * GiB, modTime: '2026-09-02 14:20', keep: true },
          { path: HOME + '/Projects/lab/iso/ubuntu-24.04.iso', bytes: 1.4 * GiB, modTime: '2026-08-11 09:02', keep: false },
        ],
      },
      {
        size: 980 * MiB, hash: 'sha256:12ab…77c', wasted: 980 * MiB,
        files: [
          { path: HOME + '/Projects/lab/data/dump.sql', bytes: 980 * MiB, modTime: '2026-10-19 22:41', keep: true },
          { path: HOME + '/Projects/lab/data/dump-copy.sql', bytes: 980 * MiB, modTime: '2026-10-20 08:12', keep: false },
        ],
      },
      {
        size: 760 * MiB, hash: 'sha256:ac90…3ee', wasted: 760 * MiB,
        files: [
          { path: HOME + '/Videos/render/intro.mov', bytes: 760 * MiB, modTime: '2026-11-01 09:12', keep: true },
          { path: HOME + '/Videos/render/intro-final.mov', bytes: 760 * MiB, modTime: '2026-11-03 21:40', keep: false },
        ],
      },
    ],
  };

  const android = {
    sdkPath: HOME + '/Android/Sdk',
    avdHome: HOME + '/.android/avd',
    avdBytes: 14.2 * GiB,
    sdkBytes: 13.7 * GiB,
    emulatorRunning: false,
    scannedAt: NOW + 'T10:12:07+07:00',
    durationMs: 640,
    avds: [
      { name: 'Pixel_7_API_35', api: '35', abi: 'arm64-v8a', bytes: 6.2 * GiB, lastUsed: '2026-11-24', ageDays: 3, suggested: false },
      { name: 'Pixel_Tablet_API_34', api: '34', abi: 'arm64-v8a', bytes: 4.1 * GiB, lastUsed: '2026-04-11', ageDays: 230, suggested: true },
      { name: 'Nexus_5X_API_30', api: '30', abi: 'x86_64', bytes: 3.9 * GiB, lastUsed: '2025-08-02', ageDays: 482, suggested: true },
    ],
    packages: [
      { id: 'system-images;android-35;google_apis;arm64-v8a', kind: 'system-image', bytes: 3.1 * GiB, usedBy: ['Pixel_7_API_35'], suggested: false },
      { id: 'system-images;android-34;google_apis;arm64-v8a', kind: 'system-image', bytes: 2.9 * GiB, usedBy: ['Pixel_Tablet_API_34'], suggested: false },
      { id: 'system-images;android-30;google_apis;x86_64', kind: 'system-image', bytes: 1.8 * GiB, usedBy: ['Nexus_5X_API_30'], suggested: false },
      { id: 'ndk;26.1.10909125', kind: 'ndk', bytes: 2.4 * GiB, usedBy: ['voi-android-app'], suggested: false },
      { id: 'ndk;25.2.9519653', kind: 'ndk', bytes: 1.7 * GiB, usedBy: [], suggested: true },
      { id: 'build-tools;34.0.0', kind: 'build-tools', bytes: 410 * MiB, usedBy: ['voi-android-app', 'cleanlie'], suggested: false },
      { id: 'platforms;android-35', kind: 'platform', bytes: 717 * MiB, usedBy: ['voi-android-app'], suggested: false },
    ],
  };

  const direnv = {
    roots: [HOME + '/Projects'],
    bytes: 3.4 * GiB,
    files: 48_210,
    errors: 0,
    scannedAt: NOW + 'T10:12:09+07:00',
    durationMs: 1_840,
    entries: [
      { id: 'project:/home/wahyu/Projects/voi-android-app/.direnv', kind: 'project', project: HOME + '/Projects/voi-android-app', path: HOME + '/Projects/voi-android-app/.direnv', bytes: 1.2 * GiB, files: 21_402, modTime: '2026-11-26 20:10', ageDays: 1, active: true },
      { id: 'project:/home/wahyu/Projects/cleanlie/.direnv', kind: 'project', project: HOME + '/Projects/cleanlie', path: HOME + '/Projects/cleanlie/.direnv', bytes: 412 * MiB, files: 8_120, modTime: '2026-11-25 22:12', ageDays: 2, active: true },
      { id: 'project:/home/wahyu/Projects/web-scraper/.direnv', kind: 'project', project: HOME + '/Projects/web-scraper', path: HOME + '/Projects/web-scraper/.direnv', bytes: 640 * MiB, files: 11_033, modTime: '2026-08-14 10:31', ageDays: 105, active: false },
      { id: 'cache:~/.cache/direnv', kind: 'cache', project: 'Cache global', path: HOME + '/.cache/direnv', bytes: 980 * MiB, files: 6_400, modTime: '2026-11-27 08:02', ageDays: 0, active: false },
      { id: 'cache:~/.cache/nix-direnv', kind: 'cache', project: 'Cache global', path: HOME + '/.cache/nix-direnv', bytes: 210 * MiB, files: 1_255, modTime: '2026-11-26 19:40', ageDays: 1, active: false },
    ],
  };

  const settings = {
    path: '~/.config/cleanlie/config.json',
    config: {
      theme: 'system', language: 'id', showRisky: true, deleteMode: 'trash',
      autoSelectSafe: true, journalTarget: '500M', largeBytes: 100 * MiB,
      staleDays: 365, artifactRoots: [HOME + '/Projects'], direnvRoots: [HOME + '/Projects'], excluded: [],
    },
  };

  const diagnostics = {
    version: '0.7.0',
    commit: '08ea7b4',
    goVersion: 'go1.26.8',
    platform: 'linux/amd64',
    home: HOME,
    configDir: HOME + '/.config',
    nixStore: true,
    polkitAgent: true,
    polkitDetail: 'hyprpolkitagent (pid 1412)',
    adminKind: 'polkit',
    startedAt: NOW + 'T10:11:58+07:00',
  };

  const noop = () => ({});
  /* Wails selalu mengembalikan Promise; UI memakai .then() langsung di beberapa tempat. */
  const api = {
        Diagnostics: () => diagnostics,
        DiskStatus: () => ({ total: 512 * GiB, free: 120 * GiB, used: 392 * GiB }),
        ScanJunk: () => report,
        ScanJunkGroup: () => report,
        CancelScan: () => true,
        Clean: () => ({ freed: 0, items: [] }),
        Settings: () => settings,
        SaveSettings: () => settings,
        ResetSettings: () => settings,
        NixStatus: () => nix,
        NixDeadPaths: () => dead,
        NixPreview: () => 'pkexec nix-collect-garbage -d',
        NixRefresh: () => true,
        NixDeleteGenerations: () => ({ action: 'delete-generations', exitCode: 0 }),
        NixCollectGarbage: () => ({ action: 'gc', exitCode: 0 }),
        VacuumJournal: () => ({ action: 'vacuum-journal', exitCode: 0 }),
        AndroidStatus: () => android,
        AndroidRefresh: () => android,
        AndroidPreview: () => '',
        AndroidDeleteAVD: () => ({ exitCode: 0 }),
        AndroidWipeAVD: () => ({ exitCode: 0 }),
        AndroidUninstallPackage: () => ({ exitCode: 0 }),
        AnalysisTree: () => tree,
        AnalysisOverview: () => overview,
        AnalysisDuplicates: () => dups,
        AnalysisRefresh: () => true,
        AnalysisClean: () => ({ freed: 0, items: [] }),
        DirenvScan: () => direnv,
        DirenvRefresh: () => direnv,
        DirenvCancel: () => true,
        DirenvAddRoot: () => settings,
        DirenvRemoveRoot: () => settings,
        DirenvPickRoot: () => '',
        DirenvClean: () => ({ freed: 0, items: [] }),
        OpenInFileManager: noop,
        CopyText: noop,
        OpenTrash: noop,
  };
  for (const key of Object.keys(api)) {
    const fn = api[key];
    api[key] = (...args) => Promise.resolve(fn(...args));
  }
  window.go = { main: { App: api } };
})();
