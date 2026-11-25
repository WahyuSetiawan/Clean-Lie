// Smoke test UI tanpa Wails: jsdom + mock binding Go.
// Fokus: scan nyata masuk ke tabel, gate permanen, dan alur trash→hasil.
// Jalankan: cd frontend/test && npm install && npm test
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import assert from 'node:assert/strict';
import { JSDOM } from 'jsdom';

const here = dirname(fileURLToPath(import.meta.url));
const html = fs.readFileSync(join(here, '..', 'dist', 'index.html'), 'utf8');

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
      History: async () => []
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

  console.log('UI smoke OK: scan, gate permanen, trash, hasil, rescan');
} finally {
  w.close();
}
