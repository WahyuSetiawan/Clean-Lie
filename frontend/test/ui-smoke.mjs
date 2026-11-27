// Smoke test UI tanpa Wails: jsdom + mock binding Go.
// Fokus: scan nyata masuk ke tabel, gate permanen, dan alur trash→hasil.
// Jalankan: cd frontend/test && npm install && npm test
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import assert from 'node:assert/strict';
import { JSDOM } from 'jsdom';

const here = dirname(fileURLToPath(import.meta.url));
const dist = join(here, '..', 'dist');

// jsdom tidak memuat aset relatif tanpa ResourceLoader; inline dulu agar tetap satu dokumen.
const html = fs
  .readFileSync(join(dist, 'index.html'), 'utf8')
  .replace(/<link[^>]+href="([^"]+\.css)"[^>]*>/g, (_, f) => `<style>${fs.readFileSync(join(dist, f), 'utf8')}</style>`)
  .replace(/<script src="([^"]+)"><\/script>/g, (_, f) => `<script>${fs.readFileSync(join(dist, f), 'utf8')}</script>`);

const mkItem = (id, group, path, bytes, risk, extra) => ({
  id, group, root: path, whole: true, label: path.split('/').pop(), path,
  bytes, files: 1, risk, fallbackDelete: true, ...(extra || {})
});

const report = {
  scannedAt: 'now', durationMs: 12, bytes: 3150000, files: 5, groups: [
    { id: 'dev.node', label: 'Node', risk: 'warn', bytes: 2100000, files: 2, items: [
      mkItem('npm-cacache', 'dev.node', '/home/u/.npm/_cacache', 1500000, 'safe', { commandLine: 'npm cache clean --force' }),
      mkItem('playwright:chromium-1', 'dev.node', '/home/u/.cache/ms-playwright/chromium-1', 600000, 'warn', { whole: false, root: '/home/u/.cache/ms-playwright' })
    ]},
    { id: 'dev.android', label: 'Android', risk: 'danger', bytes: 900000, files: 2, items: [
      mkItem('avd:test.avd', 'dev.android', '/home/u/.android/avd/test.avd', 900000, 'danger', { whole: false, root: '/home/u/.android/avd' })
    ]},
    { id: 'sys.log', label: 'Log', risk: 'warn', bytes: 150000, files: 2, items: [
      mkItem('journal', 'sys.log', '/var/log/journal', 150000, 'warn', { requiresRoot: true })
    ]}
  ]
};

let scans = 0;
let cleaned = null;
let nixStatusCalls = 0;
let nixDeadCalls = 0;
let androidStatusCalls = 0;
let androidDeleted = null;
let androidUninstalled = null;
let androidWiped = null;
let gcConfirm = null;
let analysisTreeCalls = 0;
let analysisCleanReq = null;
let analysisOpened = null;
let direnvScanCalls = 0;
let direnvAdded = null;
let direnvRemoved = null;
let direnvCleanReq = null;
let direnvPickCalls = 0;
let previewCleanupID = null;
let insightCalls = 0;
let exportFormat = null;
let direnvPickPath = '/home/u/Projects';
let savedSettings = null;
let resetSettingsCalls = 0;
let scheduleStatus = { installed: false, enabled: false, active: false, interval: 'weekly', servicePath: '/home/u/.config/systemd/user/cleanlie-clean.service', timerPath: '/home/u/.config/systemd/user/cleanlie-clean.timer', detail: 'timer belum dipasang' };
let scheduleInstalledInterval = null;
let scheduleRemoveCalls = 0;
let settingsView = {
  path: '/home/u/.config/cleanlie/config.json',
  config: {
    theme: 'system', language: 'id', showRisky: true, deleteMode: 'trash',
    autoSelectSafe: true, notify: true, journalTarget: '500M', largeBytes: 104857600,
    staleDays: 365, artifactRoots: [], direnvRoots: [], excluded: [], customRules: []
  }
};

const nixStatus = {
  storePath: '/nix/store', storeBytes: 70000000000, storePaths: 40000,
  journalBytes: 4000000000, freeBytes: 10000000000, totalBytes: 200000000000,
  agent: false, gcCommand: 'sudo nix-collect-garbage -d', errors: [],
  profiles: [
    { name: 'system', path: '/nix/var/nix/profiles/system', count: 1, current: 52 },
    { name: 'home-manager', path: '/home/u/.local/state/nix/profiles/home-manager', count: 1, current: 412 }
  ],
  generations: [
    { profile: 'system', profilePath: '/nix/var/nix/profiles/system', number: 52, date: '2026-11-25 01:21:31', ageDays: 0, current: true, closureBytes: 26000000000 },
    { profile: 'home-manager', profilePath: '/home/u/.local/state/nix/profiles/home-manager', number: 411, date: '2026-10-01 08:00:00', ageDays: 55, current: false, suggested: true, closureBytes: 1000000000 }
  ]
};
const nixDead = { count: 500, bytes: 9000000000, source: 'nix-store --gc --print-dead', durationMs: 30000 };

const androidStatus = {
  avdBytes: 14000000000, sdkBytes: 13000000000,
  sdkPath: '/home/u/Android/Sdk', avdHome: '/home/u/.android/avd',
  emulatorRunning: false, scannedAt: 'now', durationMs: 120, errors: [],
  avds: [
    { name: 'test_devices', api: '36', abi: 'x86_64', bytes: 5300000000, lastUsed: '2026-06-02 10:00', ageDays: 169, running: false, suggested: true },
    { name: 'voi_api35', api: '35', abi: 'x86_64', bytes: 8900000000, lastUsed: '2026-11-18 10:00', ageDays: 7, running: false, suggested: false }
  ],
  packages: [
    { kind: 'system-image', id: 'system-images;android-35;google_apis;x86_64', bytes: 3500000000, ageDays: 154, usedBy: [], suggested: true },
    { kind: 'system-image', id: 'system-images;android-36;google_apis;x86_64', bytes: 4300000000, ageDays: 154, usedBy: ['test_devices'], suggested: false },
    { kind: 'ndk', id: 'ndk;27.0.12077973', bytes: 2000000000, ageDays: 154, usedBy: [], suggested: false }
  ]
};

const dom = new JSDOM(html, {
  runScripts: 'dangerously',
  url: 'http://localhost/index.html',
  pretendToBeVisual: true,
  beforeParse(window) {
    window.addEventListener('error', e => { throw e.error || new Error(e.message); });
    window.runtime = { EventsOn() {}, ClipboardSetText: async () => {} };
    window.confirm = () => true;
    window.go = { main: { App: {
      Diagnostics: async () => ({ home: '/home/u', version: 'test' }),
      ScanJunk: async () => { scans++; return report; },
      Clean: async req => {
        cleaned = req;
        return {
          mode: req.mode, startedAt: 'now', durationMs: 5, before: 1500000, after: 0, freed: 1500000,
          auditPath: '/home/u/.local/state/cleanlie/history.jsonl',
          batch: 'batch-1',
          items: req.ids.map(id => ({
            id, path: '/home/u/.npm/_cacache', label: 'x', status: 'removed',
            bytesBefore: 1500000, bytesAfter: 0, freed: 1500000, mode: req.mode
          }))
        };
      },
      CancelScan: async () => true,
      CopyText: async () => {},
      OpenTrash: async () => {},
      History: async () => [],
      HistorySessions: async () => ({
        truncated: false,
        sessions: [{
          id: 'batch-1', batch: 'batch-1', action: '', mode: 'trash', startedAt: '2026-11-25T10:00:00Z',
          removed: 1, failed: 0, restorable: 1, freed: 0, restorableBytes: 1500000,
          items: [{ time: '2026-11-25T10:00:00Z', mode: 'trash', id: 'npm-cacache', path: '/home/u/.npm/_cacache', status: 'removed', trashPath: '/home/u/.local/share/Trash/files/_cacache', bytesBefore: 1500000 }]
        }]
      }),
      TrashList: async () => ({
        roots: [], bytes: 1500000, scannedAt: 'now', durationMs: 4, truncated: false,
        entries: [
          { id: 'trash-a', name: '_cacache', originalPath: '/home/u/.npm/_cacache', deletedAt: '2026-11-25T10:00:00Z', bytes: 1500000, restorable: true, reason: '', rootKind: 'home' },
          { id: 'trash-b', name: 'target', originalPath: '/home/u/Projects/app/target', deletedAt: '2026-11-24T10:00:00Z', bytes: -1, restorable: false, reason: 'tujuan restore sudah ada', rootKind: 'home' }
        ]
      }),
      TrashRefresh: async () => true,
      TrashRestore: async ids => ({ restored: ids.length, deleted: 0, skipped: 0, failed: 0, bytes: 1500000, items: ids.map(id => ({ id, name: id, status: 'restored', bytes: 1500000 })) }),
      TrashDelete: async ids => ({ restored: 0, deleted: ids.length, skipped: 0, failed: 0, bytes: 1500000, items: ids.map(id => ({ id, name: id, status: 'deleted', bytes: 1500000 })) }),
      TrashEmpty: async () => ({ restored: 0, deleted: 2, skipped: 0, failed: 0, bytes: 1500000, items: [] }),
      UndoSession: async () => ({ restored: 1, deleted: 0, skipped: 0, failed: 0, bytes: 1500000, items: [] }),
      PreviewCleanup: async id => { previewCleanupID = id; return { path: '/home/u/.npm/_cacache', kind: 'dir', bytes: 1500000, files: 2, entries: [{ name: 'index', path: '/home/u/.npm/_cacache/index', bytes: 1000, dir: false, symlink: false, modTime: '2026-11-25T10:00:00Z' }] }; },
      PreviewTrash: async () => ({ path: '/home/u/.local/share/Trash/files/_cacache', kind: 'dir', bytes: 1500000, files: 2, entries: [] }),
      NixStatus: async () => { nixStatusCalls++; return nixStatus; },
      NixDeadPaths: async () => { nixDeadCalls++; return nixDead; },
      NixPreview: async () => 'pkexec /nix/var/nix/profiles/system --delete-generations 411',
      NixCollectGarbage: async confirm => {
        gcConfirm = confirm;
        return { action: 'gc', command: [], preview: 'pkexec nix-collect-garbage -d', exitCode: 0, used: 'pkexec', durationMs: 1200, stdout: 'deleted 500 paths' };
      },
      NixDeleteGenerations: async req => ({ action: 'delete-generations', preview: 'nix-env ...', exitCode: 0, used: 'user', durationMs: 100, deleted: req.numbers.length }),
      VacuumJournal: async () => ({ action: 'vacuum-journal', exitCode: 0, durationMs: 50 }),
      AndroidStatus: async () => { androidStatusCalls++; return androidStatus; },
      AndroidRefresh: async () => true,
      AndroidPreview: async (kind, target) => 'avdmanager ' + kind + ' ' + target,
      AndroidDeleteAVD: async name => {
        androidDeleted = name;
        return { action: 'delete-avd', target: name, preview: 'avdmanager delete avd -n ' + name, exitCode: 0, freed: 5300000000, deleted: true };
      },
      AndroidUninstallPackage: async id => {
        androidUninstalled = id;
        return { action: 'uninstall', target: id, preview: 'sdkmanager --uninstall ' + id, exitCode: 0, freed: 3500000000, deleted: true };
      },
      AndroidWipeAVD: async name => {
        androidWiped = name;
        return { action: 'wipe', target: name, preview: 'emulator -avd ' + name + ' -wipe-data', exitCode: 0, freed: 0 };
      },
      DiskStatus: async () => ({ total: 200000000000, free: 80000000000, used: 120000000000 }),
      Insight: async () => {
        insightCalls++;
        return {
          version: '0.9.0', generatedAt: '2026-11-27T12:00:00Z',
          disk: { total: 200000000000, free: 80000000000, used: 120000000000 },
          week: { days: 7, since: '2026-11-20T12:00:00Z', sessions: 3, removed: 4, failed: 0, movedBytes: 4500000000, freedBytes: 1500000000, restorable: 2, restorableBytes: 1500000, trashEntries: 2, trashBytes: 1500000 },
          trend: [{ date: '2026-11-26', free: 78000000000, used: 122000000000 }, { date: '2026-11-27', free: 80000000000, used: 120000000000 }],
          sessions: [], trash: { entries: [] }
        };
      },
      ExportReport: async format => { exportFormat = format; return '/home/u/.local/state/cleanlie/reports/cleanlie-20261127-120000.' + format; },
      AnalysisTree: async (root, depth) => {
        analysisTreeCalls++;
        return {
          root: { name: root.split('/').pop(), path: root, bytes: 3000000000, files: 4, dir: true, children: [
            { name: '.cache', path: root + '/.cache', bytes: 2000000000, files: 2, dir: true },
            { name: '.npm', path: root + '/.npm', bytes: 1000000000, files: 2, dir: true }
          ]},
          depth: depth || 1, bytes: 3000000000, files: 4, dirs: 2, errors: 0, durationMs: 12
        };
      },
      AnalysisOverview: async root => ({
        root, files: 10, dirs: 3, errors: 0, durationMs: 20,
        large: [
          { path: root + '/big.iso', bytes: 1200000000, modTime: '2026-01-01 10:00', ageDays: 300 },
          { path: root + '/small.iso', bytes: 200000000, modTime: '2026-11-01 10:00', ageDays: 24 }
        ],
        stale: [
          { path: root + '/old.log', bytes: 500000, modTime: '2024-01-01 10:00', ageDays: 600 }
        ],
        artifacts: [
          { project: root + '/proj', name: 'node_modules', path: root + '/proj/node_modules', bytes: 700000000, modTime: '2026-10-01 10:00', ageDays: 55, active: false, protected: false },
          { project: root + '/baru', name: 'target', path: root + '/baru/target', bytes: 900000000, modTime: '2026-11-25 10:00', ageDays: 0, active: true, protected: false }
        ]
      }),
      AnalysisDuplicates: async root => ({
        root, groups: [{ size: 1048576, wasted: 2097152, files: [
          { path: root + '/a.bin', bytes: 1048576, modTime: '2026-01-01 10:00', ageDays: 300, keep: true },
          { path: root + '/b.bin', bytes: 1048576, modTime: '2026-02-01 10:00', ageDays: 280, keep: false },
          { path: root + '/c.bin', bytes: 1048576, modTime: '2026-03-01 10:00', ageDays: 250, keep: false }
        ]}],
        groupCount: 1, wasted: 2097152, candidates: 3, hashed: 3, errors: 0, durationMs: 40
      }),
      AnalysisRefresh: async () => true,
      AnalysisClean: async req => {
        analysisCleanReq = req;
        return { mode: 'trash', before: 1200000000, after: 0, freed: 1200000000, items: req.ids.map(id => ({
          id, path: id.split(':').slice(1).join(':'), label: 'x', status: 'removed', bytesBefore: 1200000000, bytesAfter: 0, freed: 1200000000
        })), auditPath: '/home/u/.local/state/cleanlie/history.jsonl' };
      },
      DirenvScan: async () => {
        direnvScanCalls++;
        return {
          roots: ['/home/u/Projects'], missing: [], bytes: 700000000, files: 12, errors: 0, durationMs: 15, scannedAt: 'now',
          entries: [
            { id: 'direnv:/home/u/Projects/app/.direnv', kind: 'project', project: '/home/u/Projects/app', path: '/home/u/Projects/app/.direnv', bytes: 600000000, files: 10, modTime: '2026-10-01 10:00', ageDays: 56, active: false },
            { id: 'direnv:/home/u/.cache/direnv', kind: 'cache', project: '/home/u/.cache', path: '/home/u/.cache/direnv', bytes: 100000000, files: 2, modTime: '2026-11-25 10:00', ageDays: 1, active: true }
          ]
        };
      },
      DirenvRefresh: async () => true,
      DirenvCancel: async () => true,
      DirenvPickRoot: async () => {
        direnvPickCalls++;
        return direnvPickPath;
      },
      DirenvAddRoot: async p => {
        direnvAdded = p;
        settingsView = { path: settingsView.path, config: Object.assign({}, settingsView.config, { direnvRoots: ['/home/u/Projects'] }) };
        return settingsView;
      },
      DirenvRemoveRoot: async p => {
        direnvRemoved = p;
        settingsView = { path: settingsView.path, config: Object.assign({}, settingsView.config, { direnvRoots: [] }) };
        return settingsView;
      },
      DirenvClean: async req => {
        direnvCleanReq = req;
        return {
          mode: 'trash', startedAt: 'now', durationMs: 5, before: 600000000, after: 0, freed: 600000000,
          auditPath: '/home/u/.local/state/cleanlie/history.jsonl',
          items: req.ids.map(id => ({
            id, path: id.slice('direnv:'.length), label: 'x', status: 'removed',
            bytesBefore: 600000000, bytesAfter: 0, freed: 600000000, mode: 'trash'
          }))
        };
      },
      OpenInFileManager: async p => { analysisOpened = p; },
      ScheduleStatus: async () => scheduleStatus,
      ScheduleInstall: async interval => {
        scheduleInstalledInterval = interval;
        scheduleStatus = Object.assign({}, scheduleStatus, { installed: true, enabled: true, active: true, interval, detail: 'timer aktif' });
        return scheduleStatus;
      },
      ScheduleRemove: async () => {
        scheduleRemoveCalls++;
        scheduleStatus = Object.assign({}, scheduleStatus, { installed: false, enabled: false, active: false, detail: 'timer dihapus' });
        return scheduleStatus;
      },
      Settings: async () => settingsView,
      SaveSettings: async c => {
        savedSettings = c;
        settingsView = { path: settingsView.path, config: c };
        return settingsView;
      },
      ResetSettings: async () => {
        resetSettingsCalls++;
        savedSettings = settingsView.config;
        settingsView = {
          path: settingsView.path,
          config: {
            theme: 'system', language: 'id', showRisky: true, deleteMode: 'trash',
            autoSelectSafe: true, notify: true, journalTarget: '500M', largeBytes: 104857600,
            staleDays: 365, artifactRoots: [], direnvRoots: [], excluded: [], customRules: []
          }
        };
        return settingsView;
      }
    } } };
  }
});

const w = dom.window;
const d = w.document;
const $ = id => d.getElementById(id);
const sleep = ms => new Promise(r => setTimeout(r, ms));

try {
  await sleep(300);
  d.querySelector('#nav button[data-view="cleanup"]').click();
  await sleep(500);

  assert.equal(scans, 1, 'scan dipanggil sekali saat membuka Bersihkan');
  assert.equal(d.querySelectorAll('#rows tr').length, 2, '2 item Node tampil');
  assert.equal($('sel-count').textContent, '1', 'default hanya item aman');
  assert.equal($('diag-version').textContent, 'test', 'binding tidak tertimpa fungsi navigasi');

  d.querySelector('#rows [data-preview-id]').click();
  await sleep(80);
  assert.equal(previewCleanupID, 'npm-cacache', 'preview memakai ID item');
  assert.match($('m-title').textContent, /Lihat isi/);
  $('m-cancel').click();

  // Gate hapus permanen
  $('btn-permanent').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Hapus permanen 1 item/);
  assert.equal($('m-ok').disabled, true, 'tombol permanen terkunci');
  $('perm-confirm').click();
  assert.equal($('m-ok').disabled, false, 'checkbox membuka kunci');
  $('m-cancel').click();

  // Alur trash
  $('btn-trash').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Pindahkan 1 item ke Trash/);
  assert.equal($('m-body').textContent.includes('npm cache clean --force'), true, 'command preview terlihat');
  $('m-ok').click();
  await sleep(400);

  assert.match($('m-title').textContent, /Hasil pemindahan ke Trash/);
  assert.equal(d.querySelector('#m-body .pill.safe b').textContent, '1 MiB');
  assert.equal(JSON.stringify(cleaned.ids), '["npm-cacache"]');
  assert.equal(cleaned.mode, 'trash');
  assert.equal(scans, 2, 'scan ulang setelah eksekusi');

  // Layar Nix
  d.querySelector('#nav button[data-view="nix"]').click();
  await sleep(400);
  assert.ok(nixStatusCalls >= 1, 'NixStatus dipanggil');
  assert.equal(nixDeadCalls, 1, 'NixDeadPaths dipanggil sekali');
  assert.equal(d.querySelectorAll('#nix-gens tr').length, 2, '2 generasi tampil');
  assert.match($('nix-store-size').textContent, /65\.19 GiB/);
  assert.match($('nix-dead-size').textContent, /8\.38 GiB/);
  assert.equal(d.querySelectorAll('#nix-gens button[data-nix-del]').length, 1, 'current tidak punya tombol hapus');

  $('nix-gc').click();
  await sleep(50);
  assert.match($('m-title').textContent, /garbage collection/i);
  assert.equal($('m-ok').disabled, true, 'GC terkunci sebelum ketik GC');
  $('gc-confirm').value = 'GC';
  $('gc-confirm').dispatchEvent(new w.Event('input', { bubbles: true }));
  assert.equal($('m-ok').disabled, false, 'GC terbuka setelah ketik GC');
  $('m-ok').click();
  await sleep(300);
  assert.match($('m-title').textContent, /Selesai: GC/);
  assert.equal(gcConfirm, 'GC');

  // Layar Android: default tidak ada centang, saran hanya item yang disarankan.
  d.querySelector('#nav button[data-view="android"]').click();
  await sleep(80);
  assert.equal(androidStatusCalls, 1, 'AndroidStatus dipanggil sekali');
  assert.equal(d.querySelectorAll('#android-avds tr').length, 2, '2 AVD tampil');
  assert.equal(d.querySelectorAll('#android-sysimg tr').length, 2, '2 system image tampil');
  assert.equal(d.querySelectorAll('#android-tools tr').length, 1, 'NDK tampil');
  assert.equal(d.querySelectorAll('#android-avds input[type="checkbox"]:checked').length, 0, 'default tidak ada centang');
  assert.equal($('android-delete').disabled, true, 'hapus terkunci tanpa pilihan');

  $('android-suggest').click();
  assert.equal(d.querySelectorAll('#android-avds input[type="checkbox"]:checked').length, 1, 'hanya AVD lama yang dicentang');
  assert.equal(d.querySelectorAll('#android-sysimg input[type="checkbox"]:checked').length, 1, 'hanya system image tak dipakai yang dicentang');
  assert.match($('android-selected').textContent, /2 dipilih/);
  assert.equal($('android-delete').disabled, false, 'hapus terbuka setelah pilih saran');

  $('android-delete').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Hapus 2 item Android/);
  assert.equal($('m-ok').disabled, true, 'hapus Android terkunci tanpa checkbox');
  $('ad-confirm').click();
  assert.equal($('m-ok').disabled, false, 'checkbox membuka kunci');
  $('m-ok').click();
  await sleep(80);
  assert.equal(androidDeleted, 'test_devices', 'AVD dikirim lewat nama');
  assert.equal(androidUninstalled, 'system-images;android-35;google_apis;x86_64', 'SDK dikirim lewat ID');
  assert.match($('m-title').textContent, /Selesai: Hapus item/);
  assert.match($('m-body').textContent, /8\.20 GiB/);

  // Wipe lewat tombol baris, dengan gate eksplisit.
  $('m-cancel').click();
  await sleep(50);
  d.querySelector('#android-avds button[data-ad-act="wipe"]').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Wipe 1 AVD/);
  assert.equal($('m-ok').disabled, true, 'wipe terkunci tanpa checkbox');
  $('ad-wipe-confirm').click();
  $('m-ok').click();
  await sleep(50);
  assert.equal(androidWiped, 'test_devices', 'wipe memakai nama AVD');
  assert.match($('m-title').textContent, /Selesai: Wipe data/);

  // Dashboard + Analisis M5.
  d.querySelector('#nav button[data-view="dashboard"]').click();
  await sleep(300);
  assert.notEqual($('dash-disk-used').textContent, '—', 'disk dashboard terisi');
  assert.match($('dash-cache-size').textContent, /2 MiB/, 'cache dev dari report');
  assert.match($('dash-nix-size').textContent, /65\.19 GiB/, 'Nix store dari NixStatus');
  assert.equal(d.querySelectorAll('#dash-reclaim tr').length, 3, 'reclaim top item');
  assert.equal($('dash-week-sessions').textContent, '3', 'sesi minggu ini dari Insight');
  assert.notEqual($('dash-week-freed').textContent, '—', 'freed minggu ini terisi');
  assert.ok(insightCalls >= 1, 'Insight dipanggil');
  assert.equal(d.querySelectorAll('#dash-spark polyline').length, 1, 'sparkline tren digambar');
  $('dash-export').click();
  await sleep(100);
  assert.equal(exportFormat, 'md', 'ekspor dashboard memakai format terpilih');
  assert.match($('dash-export-path').textContent, /reports\/cleanlie-/, 'path laporan tampil');

  d.querySelector('#nav button[data-view="analysis"]').click();
  await sleep(300);
  assert.equal(d.querySelectorAll('#an-tree-body tr').length, 2, 'tree 2 folder');
  assert.match($('an-crumb').textContent, /\.cache|~/, 'breadcrumb terisi');
  d.querySelector('#an-tree-body tr[data-an-path]').click();
  await sleep(250);
  assert.ok(analysisTreeCalls >= 2, 'drill-down memuat tree baru');
  $('an-open').click();
  assert.ok(analysisOpened, 'buka folder memakai binding');

  d.querySelector('#tabs button[data-tab="large"]').click();
  await sleep(250);
  assert.equal(d.querySelectorAll('#an-large-body tr').length, 2, '2 file besar');
  d.querySelector('#an-large-body input[type="checkbox"]').click();
  assert.equal($('an-large-trash').disabled, false, 'trash terbuka setelah pilih');
  $('an-large-trash').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Pindahkan 1 item ke Trash/);
  $('m-ok').click();
  await sleep(300);
  assert.equal(analysisCleanReq.mode, 'trash');
  assert.match($('m-body').textContent, /1\.12 GiB/, 'freed terukur');

  d.querySelector('#tabs button[data-tab="dup"]').click();
  await sleep(250);
  assert.match($('an-dup-summary').textContent, /1 grup/);
  d.querySelector('#an-dup-body input[data-an-id]').click();
  assert.equal($('an-dup-trash').disabled, false, 'duplikat bisa dipilih');
  d.querySelector('#tabs button[data-tab="artifacts"]').click();
  await sleep(50);
  assert.equal(d.querySelectorAll('#an-art-body tr').length, 2, '2 artefak');
  assert.equal(d.querySelectorAll('#an-art-body input[type="checkbox"]:disabled').length, 1, 'artefak aktif terkunci');
  // M7: direnv — tambah folder, scan, pilih aman, trash.
  d.querySelector('#nav button[data-view="direnv"]').click();
  await sleep(200);
  assert.equal(direnvScanCalls, 0, 'tanpa root tidak scan otomatis');
  assert.match($('direnv-summary').textContent, /Belum ada folder/);
  $('direnv-root-pick').click();
  await sleep(120);
  assert.equal(direnvPickCalls, 1, 'dialog folder dibuka');
  assert.equal(direnvAdded, '/home/u/Projects', 'folder dari picker ditambahkan');
  assert.equal($('direnv-root-input').value, '', 'input dibersihkan setelah picker');
  assert.equal(d.querySelectorAll('#direnv-roots .tag').length, 1, 'folder tampil sebagai tag');
  $('direnv-scan').click();
  await sleep(200);
  assert.equal(direnvScanCalls, 1, 'DirenvScan dipanggil');
  assert.equal(d.querySelectorAll('#direnv-body tr').length, 2, '2 entri direnv');
  assert.equal($('direnv-total').textContent, '668 MiB', 'total terukur');
  assert.equal(d.querySelectorAll('#direnv-body input[type="checkbox"]:checked').length, 0, 'default tanpa centang');
  $('direnv-select').click();
  assert.equal(d.querySelectorAll('#direnv-body input[type="checkbox"]:checked').length, 1, 'hanya entri non-aktif dipilih');
  assert.equal($('direnv-trash').disabled, false, 'trash terbuka setelah pilih');
  $('direnv-trash').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Pindahkan 1 folder \.direnv/);
  $('m-ok').click();
  await sleep(120);
  assert.equal(direnvCleanReq.mode, 'trash');
  assert.equal(JSON.stringify(direnvCleanReq.ids), '["direnv:/home/u/Projects/app/.direnv"]');
  assert.match($('m-body').textContent, /572 MiB/);
  $('m-ok').click();
  await sleep(200);
  assert.ok(direnvScanCalls >= 2, 'scan ulang setelah trash');
  d.querySelector('#direnv-roots [data-dr-rm]').click();
  await sleep(120);
  assert.equal(direnvRemoved, '/home/u/Projects', 'folder dihapus lewat binding');

  // M8: Riwayat — aktivitas, trash, preview, restore, kosongkan, undo.
  d.querySelector('#nav button[data-view="history"]').click();
  await sleep(300);
  assert.equal(d.querySelectorAll('#history-sessions tr').length, 2, '1 sesi + detail item');
  assert.equal($('nav-history-count').textContent, '1', 'badge riwayat terisi');
  $('history-export-format').value = 'json';
  $('history-export').click();
  await sleep(100);
  assert.equal(exportFormat, 'json', 'ekspor riwayat memakai format terpilih');
  d.querySelector('#history-tabs button[data-htab="trash"]').click();
  await sleep(300);
  assert.equal(d.querySelectorAll('#history-trash-body tr').length, 2, '2 item trash tampil');
  d.querySelector('#history-trash-body [data-trash-id]').click();
  assert.equal($('history-restore').disabled, false, 'restore terbuka setelah pilih');
  assert.match($('history-selected').textContent, /1 dipilih/);
  d.querySelector('#history-trash-body [data-preview]').click();
  await sleep(80);
  assert.match($('m-title').textContent, /Lihat isi/);
  $('m-cancel').click();
  $('history-restore').click();
  await sleep(50);
  assert.match($('m-title').textContent, /Pulihkan 1 item/);
  $('m-ok').click();
  await sleep(150);
  assert.match($('m-title').textContent, /Hasil restore/);
  $('m-ok').click();
  await sleep(200);

  $('history-empty').click();
  await sleep(50);
  assert.equal($('m-ok').disabled, true, 'kosongkan terkunci sebelum konfirmasi');
  $('history-empty-input').value = 'KOSONGKAN';
  $('history-empty-input').dispatchEvent(new w.Event('input', { bubbles: true }));
  assert.equal($('m-ok').disabled, false, 'kosongkan terbuka setelah ketik KOSONGKAN');
  $('m-ok').click();
  await sleep(150);
  assert.match($('m-title').textContent, /Hasil kosongkan trash/);
  $('m-ok').click();
  await sleep(150);

  d.querySelector('#history-tabs button[data-htab="activity"]').click();
  await sleep(200);
  d.querySelector('#history-sessions [data-undo]').click();
  await sleep(150);
  assert.match($('m-title').textContent, /Undo sesi ini/);
  $('m-ok').click();
  await sleep(150);
  assert.match($('m-title').textContent, /Hasil undo/);
  $('m-cancel').click();
  await sleep(100);

  // M6: settings, tema, bahasa, dan persistensi.
  d.querySelector('#nav button[data-view="settings"]').click();
  await sleep(150);
  assert.equal($('theme-select').value, 'system', 'tema default dari config');
  assert.equal($('lang-select').value, 'id', 'bahasa default dari config');
  assert.equal($('settings-path').textContent, settingsView.path, 'path config tampil');
  assert.equal($('large-mb').value, '100', 'ambang file besar dari config');
  assert.equal($('stale-days').value, '365', 'ambang file lama dari config');

  $('lang-select').value = 'en';
  $('lang-select').dispatchEvent(new w.Event('change', { bubbles: true }));
  await sleep(50);
  assert.equal($('page-title').textContent, 'Settings', 'judul halaman ikut bahasa');
  assert.equal(d.querySelector('#nav button[data-view="cleanup"] span[data-i18n]').textContent, 'Clean', 'nav ikut bahasa');
  assert.equal($('settings-save').textContent, 'Save settings', 'tombol ikut bahasa');

  $('theme-select').value = 'light';
  $('theme-select').dispatchEvent(new w.Event('change', { bubbles: true }));
  assert.equal(d.documentElement.dataset.theme, 'light', 'tema terang langsung aktif');

  $('excluded-input').value = '~/Downloads';
  $('excluded-add').click();
  assert.equal(d.querySelectorAll('#excluded-list .tag').length, 1, 'path pengecualian ditambahkan');
  $('large-mb').value = '250';
  $('stale-days').value = '30';
  $('artifact-roots').value = '~/Projects, ~/work';
  $('custom-rules').value = '~/work/cache | cache kerja | safe';
  $('settings-save').click();
  await sleep(100);
  assert.ok(savedSettings, 'SaveSettings dipanggil');
  assert.equal(savedSettings.language, 'en');
  assert.equal(savedSettings.theme, 'light');
  assert.equal(savedSettings.largeBytes, 250 * 1024 * 1024);
  assert.equal(savedSettings.staleDays, 30);
  assert.equal(JSON.stringify(savedSettings.artifactRoots), '["~/Projects","~/work"]');
  assert.equal(JSON.stringify(savedSettings.excluded), '["~/Downloads"]');
  assert.equal(savedSettings.notify, true, 'notifikasi desktop tersimpan');
  assert.equal(savedSettings.customRules.length, 1, 'aturan kustom tersimpan');
  assert.equal(savedSettings.customRules[0].path, '~/work/cache');
  assert.equal(savedSettings.customRules[0].risk, 'safe');
  assert.match($('settings-status').textContent, /Settings saved/);

  $('schedule-interval').value = 'daily';
  $('schedule-install').click();
  await sleep(80);
  assert.equal(scheduleInstalledInterval, 'daily', 'interval timer dikirim');
  assert.match($('schedule-state').textContent, /installed/, 'status timer terpasang');
  assert.match($('schedule-detail').textContent, /timer aktif/);
  $('schedule-remove').click();
  await sleep(80);
  assert.equal(scheduleRemoveCalls, 1, 'timer dihapus lewat binding');

  $('settings-reset').click();
  await sleep(100);
  assert.equal(resetSettingsCalls, 1, 'ResetSettings dipanggil');
  assert.equal($('theme-select').value, 'system');
  assert.equal($('lang-select').value, 'id');
  assert.equal($('large-mb').value, '100');

  console.log('UI smoke OK: scan, gate permanen, trash, hasil, preview, rescan, Nix, GC, Android, dashboard, analisis, direnv, Riwayat, settings, tema, i18n, aturan kustom, timer');
} finally {
  w.close();
}
