#!/usr/bin/env node
/*
 * Ambil screenshot UI Clean-Lie untuk README.
 *
 * UI statis di frontend/dist/ dijalankan di browser biasa dengan
 * `window.go.main.App` palsu (lihat fixtures.js), lalu tiap layar ditangkap
 * lewat Chrome DevTools Protocol. Hasilnya deterministik: tidak butuh mesin
 * Linux, data pribadi, atau binding Wails.
 *
 * Pemakaian:
 *   node docs/screenshots/harness/shoot.mjs
 *   CHROMIUM=/path/to/chromium node docs/screenshots/harness/shoot.mjs
 */
import { spawn } from 'node:child_process';
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(HERE, '../../..');
const DIST = path.join(ROOT, 'frontend/dist');
const OUT = path.resolve(ROOT, 'docs/screenshots');
const STAGE = path.join(tmpdir(), 'cleanlie-shots');
const PORT = 8731;
const DSF = 2;
const WIDTH = 1440;
const HEIGHT = 900;

const VIEWS = [
  { view: 'dashboard', file: 'dashboard.png' },
  { view: 'cleanup', file: 'cleanup.png' },
  { view: 'nix', file: 'nix.png' },
  { view: 'android', file: 'android.png' },
  { view: 'analysis', file: 'analysis.png' },
  { view: 'direnv', file: 'direnv.png' },
  { view: 'settings', file: 'settings.png' },
];

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.woff2': 'font/woff2',
  '.json': 'application/json',
};

const sleep = ms => new Promise(r => setTimeout(r, ms));

async function stage() {
  await rm(STAGE, { recursive: true, force: true });
  await cp(DIST, STAGE, { recursive: true });
  await cp(path.join(HERE, 'fixtures.js'), path.join(STAGE, 'ui/fixtures.js'));

  const indexPath = path.join(STAGE, 'index.html');
  const html = await readFile(indexPath, 'utf8');
  const tag = '<script src="ui/demo.js"></script>';
  if (!html.includes(tag)) throw new Error('index.html: tag demo.js tidak ditemukan');
  await writeFile(indexPath, html.replace(tag, '<script src="ui/fixtures.js"></script>\n  ' + tag));
}

function serve() {
  const server = createServer(async (req, res) => {
    try {
      const url = new URL(req.url, 'http://x');
      let file = path.join(STAGE, decodeURIComponent(url.pathname));
      if (url.pathname === '/' || url.pathname.endsWith('/')) file = path.join(file, 'index.html');
      if (!file.startsWith(STAGE)) { res.writeHead(403).end(); return; }
      const body = await readFile(file);
      res.writeHead(200, {
        'content-type': MIME[path.extname(file)] || 'application/octet-stream',
        'cache-control': 'no-store',
      });
      res.end(body);
    } catch {
      res.writeHead(404).end('not found');
    }
  });
  return new Promise(resolve => server.listen(PORT, '127.0.0.1', () => resolve(server)));
}

class CDP {
  constructor(ws) {
    this.ws = ws;
    this.id = 0;
    this.pending = new Map();
    this.events = new Map();
    ws.addEventListener('message', ev => {
      const msg = JSON.parse(ev.data);
      if (msg.id && this.pending.has(msg.id)) {
        const { resolve, reject } = this.pending.get(msg.id);
        this.pending.delete(msg.id);
        msg.error ? reject(new Error(msg.error.message)) : resolve(msg.result);
      } else if (msg.method && this.events.has(msg.method)) {
        for (const fn of this.events.get(msg.method)) fn(msg.params);
        this.events.delete(msg.method);
      }
    });
  }
  static async connect(url) {
    const ws = new WebSocket(url);
    await Promise.race([
      new Promise((resolve, reject) => {
        ws.addEventListener('open', resolve, { once: true });
        ws.addEventListener('error', () => reject(new Error('WebSocket CDP ditolak')), { once: true });
      }),
      sleep(5000).then(() => { throw new Error('WebSocket CDP timeout'); }),
    ]);
    return new CDP(ws);
  }
  send(method, params = {}) {
    const id = ++this.id;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }
  once(method) {
    return new Promise(resolve => {
      if (!this.events.has(method)) this.events.set(method, []);
      this.events.get(method).push(resolve);
    });
  }
  close() { this.ws.close(); }
}

const APP_BINARIES = [
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
  '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser',
  '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
];

async function chromiumBin() {
  const explicit = process.env.CHROMIUM;
  if (explicit) return explicit;
  for (const app of APP_BINARIES) if (existsSync(app)) return app;
  for (const name of ['chromium', 'chromium-browser', 'google-chrome', 'chrome', 'brave']) {
    const which = spawn('sh', ['-c', `command -v ${name}`], { stdio: ['ignore', 'pipe', 'ignore'] });
    let out = '';
    which.stdout.on('data', d => { out += d; });
    const code = await new Promise(r => which.on('close', r));
    if (code === 0 && out.trim()) return out.trim();
  }
  throw new Error('Chromium tidak ditemukan. Set CHROMIUM=/path/to/chromium');
}

async function main() {
  await stage();
  const server = await serve();
  const bin = await chromiumBin();
  const profile = path.join(tmpdir(), 'cleanlie-shots-profile');
  await rm(profile, { recursive: true, force: true });

  const chrome = spawn(bin, [
    '--headless=new',
    '--disable-gpu',
    '--hide-scrollbars',
    '--no-first-run',
    '--no-default-browser-check',
    '--force-color-profile=srgb',
    '--remote-allow-origins=*',
    '--remote-debugging-port=9333',
    `--user-data-dir=${profile}`,
    'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] });

  let chromeErr = '';
  chrome.stderr.on('data', d => { chromeErr += d; });

  try {
    let targets;
    for (let i = 0; i < 100; i++) {
      try {
        const res = await fetch('http://127.0.0.1:9333/json/list');
        targets = await res.json();
        if (targets.length) break;
      } catch { /* belum siap */ }
      await sleep(100);
    }
    if (!targets || !targets.length) throw new Error('DevTools tidak siap:\n' + chromeErr);
    const page = targets.find(t => t.type === 'page') || targets[0];
    const cdp = await CDP.connect(page.webSocketDebuggerUrl);

    await cdp.send('Page.enable');
    await cdp.send('Runtime.enable');
    await cdp.send('Emulation.setDeviceMetricsOverride', {
      width: WIDTH, height: HEIGHT, deviceScaleFactor: DSF, mobile: false,
    });

    await mkdir(OUT, { recursive: true });

    for (const { view, file } of VIEWS) {
      const loaded = cdp.once('Page.loadEventFired');
      await cdp.send('Page.navigate', { url: `http://127.0.0.1:${PORT}/index.html?v=${view}#/${view}` });
      await Promise.race([loaded, sleep(5000)]);
      await sleep(250);

      await cdp.send('Runtime.evaluate', {
        expression: `
          (() => {
            if (typeof navigate === 'function') navigate(${JSON.stringify(view)});
            const style = document.createElement('style');
            style.textContent = '*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}';
            document.head.appendChild(style);
            window.scrollTo(0, 0);
          })()
        `,
      });

      /* beri waktu binding palsu + render tabel selesai */
      const deadline = Date.now() + 6000;
      while (Date.now() < deadline) {
        const r = await cdp.send('Runtime.evaluate', {
          expression: `document.getElementById('view-${view}').innerText`,
          returnByValue: true,
        });
        if (!/memuat/i.test(r.result.value || '')) break;
        await sleep(200);
      }
      /* render ulang setelah home diketahui (path jadi ~/...) */
      await cdp.send('Runtime.evaluate', {
        expression: `typeof navigate === 'function' && navigate(${JSON.stringify(view)})`,
      });
      await sleep(300);

      const shot = await cdp.send('Page.captureScreenshot', { format: 'png', fromSurface: true });
      const raw = path.join(tmpdir(), `cleanlie-${view}.png`);
      await writeFile(raw, Buffer.from(shot.data, 'base64'));

      /* 2x capture lalu turunkan ke 1x: teks tetap tajam, file kecil */
      await new Promise((resolve, reject) => {
        const magick = spawn('magick', [raw, '-resize', '50%', '-strip', path.join(OUT, file)], { stdio: 'inherit' });
        magick.on('close', code => code === 0 ? resolve() : reject(new Error('magick gagal: ' + code)));
      });
      console.log(`ok  docs/screenshots/${file}  (${view})`);
    }

    cdp.close();
  } finally {
    chrome.kill('SIGKILL');
    server.close();
    await rm(STAGE, { recursive: true, force: true });
    await rm(profile, { recursive: true, force: true });
  }
}

main().catch(err => {
  console.error(err.message || err);
  process.exit(1);
});
