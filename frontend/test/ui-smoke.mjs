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
let savedSettings = null;
let resetSettingsCalls = 0;
let settingsView = {
  path: '/home/u/.config/cleanlie/config.json',
  config: {
    theme: 'system', language: 'id', showRisky: true, deleteMode: 'trash',
    autoSelectSafe: true, journalTarget: '500M', largeBytes: 104857600,
    staleDays: 365, artifactRoots: [], excluded: []
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
    window.go = { main: { App: {
      Diagnostics: async () => ({ home: '/home/u', version: 'test' }),
      ScanJunk: async () => { scans++; return report; },
      Clean: async req => {
        cleaned = req;
        return {
          mode: req.mode, startedAt: 'now', durationMs: 5, before: 1500000, after: 0, freed: 1500000,
          auditPath: '/home/u/.local/state/cleanlie/history.jsonl',
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
      OpenInFileManager: async p => { analysisOpened = p; },
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
            autoSelectSafe: true, journalTarget: '500M', largeBytes: 104857600,
            staleDays: 365, artifactRoots: [], excluded: []
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
  $('settings-save').click();
  await sleep(100);
  assert.ok(savedSettings, 'SaveSettings dipanggil');
  assert.equal(savedSettings.language, 'en');
  assert.equal(savedSettings.theme, 'light');
  assert.equal(savedSettings.largeBytes, 250 * 1024 * 1024);
  assert.equal(savedSettings.staleDays, 30);
  assert.equal(JSON.stringify(savedSettings.artifactRoots), '["~/Projects","~/work"]');
  assert.equal(JSON.stringify(savedSettings.excluded), '["~/Downloads"]');
  assert.match($('settings-status').textContent, /Settings saved/);

  $('settings-reset').click();
  await sleep(100);
  assert.equal(resetSettingsCalls, 1, 'ResetSettings dipanggil');
  assert.equal($('theme-select').value, 'system');
  assert.equal($('lang-select').value, 'id');
  assert.equal($('large-mb').value, '100');

  console.log('UI smoke OK: scan, gate permanen, trash, hasil, rescan, Nix, GC, Android, dashboard, analisis, settings, tema, i18n');
} finally {
  w.close();
}
