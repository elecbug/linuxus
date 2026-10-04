// Run with Node 20+ and Playwright installed outside the application runtime.
// PLAYWRIGHT_MODULE may point to an existing Playwright installation.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const root = fileURLToPath(new URL('..', import.meta.url));
const assets = path.join(root, 'src/internal/auth/handler');
const base = '/ops/classroom';
const values = { BaseURL: base, APIURL: base + '/api/users', LoginURL: '/auth/login',
  ServiceURL: '/workspace/', LogoutURL: '/auth/logout', AdminID: 'admin', Token: 'initial-token' };
const html = (await readFile(path.join(assets, 'admin.html'), 'utf8')).replace(/{{\.(\w+)}}/g, (_, key) => values[key]);
const css = await readFile(path.join(assets, 'admin.css'), 'utf8');
const js = await readFile(path.join(assets, 'admin.js'), 'utf8');
const browser = await chromium.launch({ headless: true });
let passed = 0;

function gate() {
  let release;
  const promise = new Promise(resolve => { release = resolve; });
  return { promise, release };
}

async function waitUntil(predicate, message) {
  const deadline = Date.now() + 5000;
  while (!await predicate()) {
    assert.ok(Date.now() < deadline, message);
    await new Promise(resolve => setTimeout(resolve, 10));
  }
}

async function fixture(viewport = { width: 1440, height: 1000 }) {
  const context = await browser.newContext({ viewport });
  const page = await context.newPage();
  const failures = [];
  page.on('pageerror', error => failures.push(error.message));
  const api = { gets: 0, posts: 0, active: 0, maximum: 0, getStatus: 200, postStatus: 200,
    holdGet: null, holdPost: null, updated: true, reauthenticate: false, payloads: [], requests: [] };
  const data = { csrf_token: 'fresh-token', status_available: true,
    templates: { linux: {}, c: {}, python: {} }, classes: { 'class-a': 'python' }, users: [
      { id: 'admin', locked: false, state: 'running', sessions: 1, template: 'linux', mounted: true, used_bytes: 536870912, total_bytes: 2147483648 },
      { id: 'alice', locked: false, state: 'running', sessions: 2, class: 'class-a', mounted: true, used_bytes: 2147483648, total_bytes: 8589934592 },
      { id: 'bob', locked: true, state: 'stopped', sessions: 0, mounted: true, used_bytes: 429496730, total_bytes: 2147483648 },
      { id: 'carol', locked: true, maintenance: true, state: 'stopped', sessions: 0, mounted: false }
    ] };
  await page.route('http://linuxus.test/**', async route => {
    const request = route.request();
    const pathname = new URL(request.url()).pathname;
    api.requests.push(pathname);
    try {
      if (pathname === base || pathname === base + '/') return await route.fulfill({ contentType: 'text/html', body: html });
      if (pathname === base + '/assets/admin.css') return await route.fulfill({ contentType: 'text/css', body: css });
      if (pathname === base + '/assets/admin.js') return await route.fulfill({ contentType: 'text/javascript', body: js });
      if (pathname === '/static/favicon.png') return await route.fulfill({ status: 204 });
      assert.equal(pathname, base + '/api/users', 'unexpected or hardcoded public route');
      if (request.method() === 'GET') {
        api.gets++; api.active++; api.maximum = Math.max(api.maximum, api.active);
        const status = api.getStatus;
        const body = status === 200 ? JSON.stringify(data) : 'Status service unavailable';
        const held = api.holdGet; api.holdGet = null;
        try {
          if (held) await held.promise;
          return await route.fulfill({ status, contentType: status === 200 ? 'application/json' : 'text/plain', body });
        } finally { api.active--; }
      }
      assert.equal(request.method(), 'POST');
      assert.equal(request.headers()['x-csrf-token'], 'fresh-token', 'mutation did not use the latest CSRF token');
      api.posts++;
      const payload = request.postDataJSON(); api.payloads.push(payload);
      const held = api.holdPost; api.holdPost = null;
      if (held) await held.promise;
      const user = data.users.find(user => user.id === payload.user_id);
      if (api.updated) {
        if (payload.action === 'lock') user.locked = true;
        if (payload.action === 'unlock') user.locked = false;
        if (payload.action === 'template') user.template = payload.value;
        if (payload.action === 'class') { user.class = payload.value; user.template = ''; }
      }
      const headers = {};
      if (api.updated) headers['X-Linuxus-Account-Updated'] = 'true';
      if (api.reauthenticate) headers['X-Linuxus-Reauthenticate'] = 'true';
      return await route.fulfill({ status: api.postStatus, headers,
        contentType: api.postStatus === 200 ? 'application/json' : 'text/plain',
        body: api.postStatus === 200 ? '{"ok":true}' : 'Account updated; Manager operation failed (HTTP 503)' });
    } catch (error) {
      // Aborted reads deliberately finish after editing or navigation has begun.
      if (!/already handled|closed|disposed|intercepted/i.test(error.message)) failures.push(error.stack);
    }
  });
  await page.clock.install();
  await page.goto('http://linuxus.test' + base);
  await waitUntil(async () => await page.locator('#total-users').textContent() === '4', 'initial dashboard did not load');
  await waitUntil(async () => await page.locator('#refresh').isEnabled(), 'initial refresh did not finish');
  return { page, api, data, failures, async close() { await context.close(); assert.deepEqual(failures, [], 'browser errors'); } };
}

async function refreshAndWait(f) {
  const before = f.api.gets;
  await f.page.locator('#refresh').click();
  await waitUntil(() => f.api.gets > before, 'manual refresh did not request status');
  await waitUntil(async () => await f.page.locator('#refresh').isEnabled(), 'manual refresh did not finish');
}

async function run(name, test) {
  const f = await fixture();
  try { await test(f); passed++; console.log('PASS:', name); }
  finally { await f.close(); }
}

try {
  await run('custom paths, summary, search, filters, and maintenance controls', async ({ page, api }) => {
    assert.equal(await page.locator('#running-users').textContent(), '2');
    assert.equal(await page.locator('#active-sessions').textContent(), '3');
    assert.equal(await page.locator('#locked-users').textContent(), '2');
    assert.ok(await page.getByRole('button', { name: 'Manage carol', exact: true }).isDisabled());
    await page.locator('#search').fill('class-a');
    assert.equal(await page.locator('#users tr:visible').count(), 1);
    await page.locator('#search').fill('');
    await page.locator('#filter').selectOption('locked');
    assert.equal(await page.locator('#users tr:visible').count(), 2);
    assert.ok(!api.requests.some(value => value.startsWith('/admin')));
  });

  await run('refresh preserves row identity, search text, focus, and filters', async f => {
    const { page, data } = f;
    await page.locator('#search').fill('alice');
    await page.locator('#filter').selectOption('running');
    await page.locator('#search').focus();
    await page.evaluate(() => { document.querySelector('[data-user-id="alice"]').testIdentity = true; });
    data.users[1].sessions = 4;
    await page.clock.runFor(15500);
    await waitUntil(async () => (await page.locator('[data-user-id="alice"] .row-note').allTextContents()).includes('4 active sessions'), 'automatic refresh did not update sessions');
    assert.equal(await page.locator('#search').inputValue(), 'alice');
    assert.equal(await page.locator('#filter').inputValue(), 'running');
    assert.equal(await page.evaluate(() => document.activeElement.id), 'search');
    assert.ok(await page.evaluate(() => document.querySelector('[data-user-id="alice"]').testIdentity));
  });

  await run('overlapping manual refreshes share one pending request', async f => {
    const held = gate(); f.api.holdGet = held;
    const before = f.api.gets;
    await f.page.locator('#refresh').click();
    await waitUntil(() => f.api.gets === before + 1, 'held status request did not start');
    await f.page.evaluate(() => { for (let i = 0; i < 5; i++) document.getElementById('refresh').dispatchEvent(new Event('click')); });
    await f.page.clock.runFor(5000);
    assert.equal(f.api.gets, before + 1);
    assert.ok(await f.page.locator('#refresh').isDisabled());
    held.release();
    await waitUntil(async () => await f.page.locator('#refresh').isEnabled(), 'refresh did not recover after pending request');
    assert.equal(f.api.maximum, 1);
  });

  await run('dialogs preserve password input and pause polling', async ({ page, api }) => {
    await page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await page.locator('#operation').selectOption('password');
    await page.locator('#new-password').fill('new-password');
    await page.locator('#confirm-password').fill('new-password');
    const before = api.gets;
    await page.clock.runFor(45000);
    assert.equal(api.gets, before);
    assert.equal(await page.locator('#new-password').inputValue(), 'new-password');
    assert.equal(await page.locator('#operation').inputValue(), 'password');
    assert.ok(await page.locator('#refresh').isDisabled());
    await page.locator('#cancel-dialog').click();
    await waitUntil(() => api.gets > before, 'closing the dialog did not resume updates');
    assert.equal(await page.locator('#new-password').inputValue(), '');
  });

  await run('mutations are single-flight, block closing, and refresh once complete', async ({ page, api }) => {
    await page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await page.locator('#operation').selectOption('password');
    await page.locator('#new-password').fill('confirmed-password');
    await page.locator('#confirm-password').fill('confirmed-password');
    const held = gate(); api.holdPost = held;
    await page.locator('#apply-action').click();
    await waitUntil(() => api.posts === 1, 'mutation did not begin');
    await page.evaluate(() => document.getElementById('account-form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })));
    const before = api.gets;
    await page.keyboard.press('Escape');
    await page.clock.runFor(20000);
    assert.equal(api.posts, 1); assert.equal(api.gets, before);
    assert.ok(await page.locator('#account-dialog').isVisible());
    assert.ok(await page.locator('#apply-action').isDisabled());
    held.release();
    await waitUntil(async () => !await page.locator('#account-dialog').isVisible(), 'successful mutation left the dialog open');
    await waitUntil(() => api.gets > before, 'mutation did not refresh status');
    assert.equal(await page.locator('#new-password').inputValue(), '');
    assert.match(await page.locator('#action-notice').textContent(), /Reset password completed/);
    await page.clock.runFor(100);
    assert.equal(api.gets, before + 1, 'dialog close duplicated the post-action refresh');
  });

  await run('password confirmation prevents requests with mismatched input', async ({ page, api }) => {
    await page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await page.locator('#operation').selectOption('password');
    await page.locator('#new-password').fill('first-password');
    await page.locator('#confirm-password').fill('second-password');
    await page.locator('#apply-action').click();
    assert.equal(api.posts, 0);
    assert.equal(await page.locator('#confirm-password').evaluate(element => element.validationMessage), 'Passwords do not match.');
  });

  await run('failed refresh retains data and can recover', async f => {
    f.api.getStatus = 503;
    await refreshAndWait(f);
    assert.equal(await f.page.locator('#users tr').count(), 4);
    assert.match(await f.page.locator('#load-error').textContent(), /last successful update/);
    assert.equal(await f.page.locator('#connection-status').textContent(), 'Update failed');
    f.api.getStatus = 200;
    await refreshAndWait(f);
    assert.ok(await f.page.locator('#load-error').isHidden());
  });

  await run('partial mutations remain visible after status refresh', async ({ page, api }) => {
    api.postStatus = 503;
    await page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await page.locator('#apply-action').click();
    await waitUntil(async () => !await page.locator('#account-dialog').isVisible(), 'partial mutation left an unsafe retry form open');
    await waitUntil(async () => await page.locator('[data-user-id="alice"] .badge').first().textContent() === 'Locked', 'partial mutation did not update account status');
    assert.match(await page.locator('#action-notice').textContent(), /Account updated; Manager operation failed/);
    assert.equal(api.posts, 1);
  });

  await run('expired sessions stop polling and offer the configured login route', async f => {
    f.api.getStatus = 401;
    await f.page.locator('#refresh').click();
    await f.page.locator('#session-notice').waitFor({ state: 'visible' });
    const before = f.api.gets;
    await f.page.clock.runFor(60000);
    assert.equal(f.api.gets, before);
    assert.ok(await f.page.getByRole('button', { name: 'Manage alice', exact: true }).isDisabled());
    assert.equal(await f.page.locator('#session-notice a').getAttribute('href'), '/auth/login');
  });

  await run('changing the current administrator handles revoked sessions without retry', async ({ page, api }) => {
    api.reauthenticate = true; api.postStatus = 503;
    await page.getByRole('button', { name: 'Manage admin', exact: true }).click();
    assert.equal(await page.locator('#operation option[value="lock"]').count(), 0);
    await page.locator('#operation').selectOption('disconnect');
    const before = api.gets;
    await page.locator('#apply-action').click();
    await page.locator('#session-notice').waitFor({ state: 'visible' });
    await page.clock.runFor(60000);
    assert.equal(api.gets, before);
    assert.match(await page.locator('#action-notice').textContent(), /Account updated/);
  });

  await run('a cancelled old response cannot overwrite a completed account action', async ({ page, api }) => {
    const held = gate(); api.holdGet = held;
    const before = api.gets;
    await page.locator('#refresh').click();
    await waitUntil(() => api.gets > before, 'old request did not begin');
    await page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await page.locator('#apply-action').click();
    await waitUntil(async () => await page.locator('[data-user-id="alice"] .badge').first().textContent() === 'Locked', 'latest account action was not rendered');
    held.release();
    await waitUntil(() => api.active === 0, 'cancelled request did not finish');
    assert.equal(await page.locator('[data-user-id="alice"] .badge').first().textContent(), 'Locked');
  });

  await run('automatic refresh pauses in hidden tabs and supports manual control', async ({ page, api }) => {
    const before = api.gets;
    await page.evaluate(() => { Object.defineProperty(document, 'hidden', { configurable: true, get: () => true }); document.dispatchEvent(new Event('visibilitychange')); });
    await page.clock.runFor(45000);
    assert.equal(api.gets, before);
    await page.evaluate(() => { delete document.hidden; document.dispatchEvent(new Event('visibilitychange')); });
    await waitUntil(() => api.gets > before, 'visible tab did not refresh');
    await waitUntil(async () => await page.locator('#refresh').isEnabled(), 'visible tab refresh did not finish');
    await page.locator('#auto-refresh').uncheck();
    const paused = api.gets;
    await page.clock.runFor(45000);
    assert.equal(api.gets, paused);
    await page.locator('#refresh').click();
    await waitUntil(() => api.gets > paused, 'manual refresh did not work with automatic refresh disabled');
  });

  const f = await fixture();
  try {
    const screenshotDir = process.env.LINUXUS_UI_SCREENSHOTS;
    if (screenshotDir) {
      await mkdir(screenshotDir, { recursive: true });
      await f.page.screenshot({ path: path.join(screenshotDir, 'admin-desktop.png'), fullPage: true });
    }
    await f.page.setViewportSize({ width: 390, height: 844 });
    const overflow = await f.page.evaluate(() => ({ width: innerWidth, document: document.documentElement.scrollWidth,
      elements: Array.from(document.querySelectorAll('body *')).filter(element => !element.closest('.table-scroll') && element.getBoundingClientRect().right > innerWidth + 1).map(element => ({ tag: element.tagName, id: element.id, className: element.className, right: element.getBoundingClientRect().right })) }));
    assert.ok(overflow.document <= overflow.width, 'mobile page overflows horizontally: ' + JSON.stringify(overflow));
    if (screenshotDir) await f.page.screenshot({ path: path.join(screenshotDir, 'admin-mobile.png'), fullPage: true });
    await f.page.getByRole('button', { name: 'Manage alice', exact: true }).click();
    await f.page.locator('#operation').selectOption('template');
    assert.equal(await f.page.locator('#assignment').inputValue(), 'python', 'inherited template was not selected');
    if (screenshotDir) await f.page.screenshot({ path: path.join(screenshotDir, 'admin-dialog.png'), fullPage: true });
    passed++; console.log('PASS: responsive layout and inherited template selection');
  } finally { await f.close(); }
  console.log(`PASS: ${passed} browser scenarios; no live services or user data were accessed.`);
} finally {
  await browser.close();
}
