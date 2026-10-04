(() => {
  'use strict';

  const byID = id => document.getElementById(id);
  const ui = Object.fromEntries(['users', 'refresh', 'auto-refresh', 'search', 'filter', 'result-count',
    'updated-at', 'refresh-note', 'connection-status', 'empty-state', 'table-region', 'load-error',
    'service-warning', 'action-notice', 'session-notice', 'account-dialog', 'account-form', 'dialog-title',
    'operation', 'operation-description', 'password-fields', 'new-password', 'confirm-password',
    'assignment-field', 'assignment-label', 'assignment', 'self-notice', 'dialog-error', 'apply-action',
    'close-dialog', 'cancel-dialog'].map(id => [id, byID(id)]));
  const config = document.body.dataset;
  const state = { data: null, rows: new Map(), read: null, sequence: 0, timer: null,
    writing: false, expired: false, disposed: false, target: null, updated: null,
    refreshOnClose: true, csrf: config.csrfToken, lastError: '' };
  const descriptions = {
    lock: 'Block new logins and stop the environment. Home files are preserved.',
    unlock: 'Allow this user to sign in and start their environment again.',
    disconnect: 'Close existing sessions and stop the environment. The user can sign in again.',
    restart: 'Disconnect current sessions and recreate the environment. Home files are preserved.',
    password: 'Set a new password, revoke existing sessions, and stop the environment.',
    template: 'Choose the image and starter files for the next connection. Existing home files are preserved.',
    class: 'Assign a classroom and use its template. This clears the individual template override.'
  };
  const labels = { lock: 'Lock account', unlock: 'Unlock account', disconnect: 'Disconnect sessions',
    restart: 'Restart environment', password: 'Reset password', template: 'Assign template', class: 'Assign class' };

  function notice(element, text, kind) {
    element.textContent = text;
    element.hidden = !text;
    if (kind) element.className = 'notice ' + kind;
  }

  function canPoll() {
    return !state.disposed && !state.expired && !state.writing && !ui['account-dialog'].open &&
      !document.hidden && ui['auto-refresh'].checked;
  }

  function stopTimer() {
    clearTimeout(state.timer);
    state.timer = null;
  }

  function schedule() {
    stopTimer();
    if (canPoll()) state.timer = setTimeout(refresh, 15000);
    updateControls();
  }

  function updateControls() {
    ui.refresh.disabled = state.expired || state.writing || !!state.read || ui['account-dialog'].open;
    ui.refresh.textContent = state.read ? 'Refreshing…' : 'Refresh';
    ui['auto-refresh'].disabled = state.expired;
    ui['table-region'].setAttribute('aria-busy', String(!!state.read));
    let note = 'Updates every 15 seconds';
    if (state.expired) note = 'Sign in again to resume updates';
    else if (state.writing) note = 'Updates paused while applying changes';
    else if (ui['account-dialog'].open) note = 'Updates paused while editing';
    else if (document.hidden) note = 'Updates paused while this tab is hidden';
    else if (!ui['auto-refresh'].checked) note = 'Automatic updates paused';
    ui['refresh-note'].textContent = note;
    const status = ui['connection-status'];
    status.dataset.state = state.expired || state.lastError ? 'error' : state.updated ? 'live' : 'pending';
    status.textContent = state.expired ? 'Sign-in required' : state.writing ? 'Applying changes' :
      state.read ? 'Refreshing' : state.lastError ? 'Update failed' : !state.updated ? 'Connecting' :
      ui['account-dialog'].open ? 'Editing' : canPoll() ? 'Live updates' : 'Updates paused';
    for (const row of state.rows.values()) row.manage.disabled = state.expired || state.writing || row.maintenance;
  }

  function cancelRead() {
    state.sequence++;
    if (state.read) state.read.controller.abort();
    state.read = null;
  }

  function sessionExpired() {
    state.expired = true;
    stopTimer();
    cancelRead();
    ui['session-notice'].hidden = false;
    notice(ui['load-error'], '');
    updateControls();
  }

  async function refresh() {
    if (state.disposed || state.expired || state.writing || ui['account-dialog'].open) return;
    if (state.read) return state.read.promise;
    stopTimer();
    const job = { id: ++state.sequence, controller: new AbortController(), promise: null };
    state.read = job;
    updateControls();
    job.promise = (async () => {
      let timedOut = false;
      const timeout = setTimeout(() => { timedOut = true; job.controller.abort(); }, 15000);
      try {
        const response = await fetch(config.apiUrl, { cache: 'no-store', credentials: 'same-origin', signal: job.controller.signal });
        if (job.id !== state.sequence) return;
        if (response.status === 401 || response.status === 403) { sessionExpired(); return; }
        if (!response.ok) throw new Error((await response.text()).trim() || 'Unable to load account status.');
        const data = await response.json();
        if (!data || !Array.isArray(data.users)) throw new Error('The server returned an invalid account list.');
        if (job.id !== state.sequence || state.writing || ui['account-dialog'].open || state.disposed) return;
        if (data.csrf_token) state.csrf = data.csrf_token;
        state.lastError = '';
        notice(ui['load-error'], '');
        notice(ui['service-warning'], data.warning || '', 'warning');
        render(data);
        state.updated = new Date();
        ui['updated-at'].textContent = 'Last updated at ' + state.updated.toLocaleTimeString();
      } catch (error) {
        if (job.id !== state.sequence || state.disposed || (error.name === 'AbortError' && !timedOut)) return;
        state.lastError = timedOut ? 'The status request timed out.' : error.message;
        notice(ui['load-error'], state.lastError + (state.data ? ' Showing the last successful update.' : ' Use Refresh to try again.'));
        if (!state.data) emptyState('Unable to load users', 'Check the connection, then try Refresh.');
      } finally {
        clearTimeout(timeout);
        if (state.read === job) { state.read = null; schedule(); }
      }
    })();
    return job.promise;
  }

  function node(tag, className, text) {
    const element = document.createElement(tag);
    if (className) element.className = className;
    if (text !== undefined) element.textContent = text;
    return element;
  }

  function createRow(id) {
    const row = { root: node('tr'), maintenance: false };
    row.root.dataset.userId = id;
    const cells = Array.from({ length: 6 }, () => { const cell = node('td'); row.root.append(cell); return cell; });
    row.name = node('span', 'user-name', id);
    cells[0].append(row.name);
    if (id === config.adminId) cells[0].append(node('span', 'row-note', 'You · Administrator'));
    row.account = node('span', 'badge'); cells[1].append(row.account);
    row.runtime = node('span', 'badge'); row.sessions = node('span', 'row-note'); cells[2].append(row.runtime, row.sessions);
    row.storage = node('span', 'storage-size'); row.meter = node('progress', 'storage-meter');
    row.meter.max = 100; row.meter.setAttribute('aria-label', 'Home storage used by ' + id); cells[3].append(row.storage, row.meter);
    row.template = node('span'); row.className = node('span', 'row-note'); cells[4].append(row.template, row.className);
    row.manage = node('button', 'button manage', 'Manage'); row.manage.type = 'button';
    row.manage.setAttribute('aria-label', 'Manage ' + id); row.manage.addEventListener('click', () => openDialog(id)); cells[5].append(row.manage);
    return row;
  }

  function size(bytes) {
    if (!Number.isFinite(bytes) || bytes < 0) return '—';
    if (bytes < 1024 ** 3) return (bytes / 1024 ** 2).toFixed(1) + ' MiB';
    return (bytes / 1024 ** 3).toFixed(2) + ' GiB';
  }

  function templateFor(user) {
    return user.template || (user.class ? state.data.classes?.[user.class] : 'default') || 'Unavailable';
  }

  function render(data) {
    state.data = data;
    const ids = new Set(data.users.map(user => user.id));
    for (const [id, row] of state.rows) if (!ids.has(id)) { row.root.remove(); state.rows.delete(id); }
    let position = ui.users.firstElementChild;
    for (const user of data.users) {
      let row = state.rows.get(user.id);
      if (!row) { row = createRow(user.id); state.rows.set(user.id, row); }
      if (row.root !== position) ui.users.insertBefore(row.root, position);
      position = row.root.nextElementSibling;
      row.maintenance = !!user.maintenance;
      const account = user.maintenance ? 'maintenance' : user.locked ? 'locked' : 'enabled';
      row.account.dataset.kind = account;
      row.account.textContent = { maintenance: 'Maintenance', locked: 'Locked', enabled: 'Enabled' }[account];
      const runtime = user.state || (data.status_available === false ? 'unknown' : 'stopped');
      row.runtime.dataset.kind = runtime;
      row.runtime.textContent = runtime.charAt(0).toUpperCase() + runtime.slice(1);
      row.sessions.textContent = data.status_available === false ? 'Session status unavailable' : (user.sessions || 0) + ' active sessions';
      row.storage.textContent = user.disk_error || (user.mounted ? size(user.used_bytes) + ' / ' + size(user.total_bytes) : data.status_available === false ? 'Unavailable' : 'Not mounted');
      row.meter.hidden = !!user.disk_error || !user.mounted || !Number.isFinite(user.used_bytes) || !(user.total_bytes > 0);
      if (!row.meter.hidden) {
        row.meter.value = Math.max(0, Math.min(100, user.used_bytes / user.total_bytes * 100));
        row.meter.classList.toggle('high', row.meter.value >= 90);
      }
      row.template.textContent = templateFor(user);
      row.className.textContent = user.class ? 'Class: ' + user.class : 'No class assigned';
      row.manage.title = user.maintenance ? 'Complete disk maintenance before changing this account.' : 'Manage ' + user.id;
    }
    byID('total-users').textContent = data.users.length;
    byID('running-users').textContent = data.status_available === false ? '—' : data.users.filter(user => user.state === 'running').length;
    byID('active-sessions').textContent = data.status_available === false ? '—' : data.users.reduce((total, user) => total + (user.sessions || 0), 0);
    byID('locked-users').textContent = data.users.filter(user => user.locked || user.maintenance).length;
    filterRows();
    updateControls();
  }

  function emptyState(title, description) {
    ui['empty-state'].hidden = false;
    ui['empty-state'].querySelector('strong').textContent = title;
    ui['empty-state'].querySelector('p').textContent = description;
  }

  function filterRows() {
    if (!state.data) return;
    const query = ui.search.value.trim().toLowerCase();
    const filter = ui.filter.value;
    let visible = 0;
    for (const user of state.data.users) {
      const matchesText = [user.id, user.class, templateFor(user)].filter(Boolean).join(' ').toLowerCase().includes(query);
      const matchesFilter = filter === 'all' || (filter === 'running' && user.state === 'running') ||
        (filter === 'locked' && user.locked) || (filter === 'maintenance' && user.maintenance);
      const show = matchesText && matchesFilter;
      state.rows.get(user.id).root.hidden = !show;
      if (show) visible++;
    }
    ui['result-count'].textContent = visible + ' of ' + state.data.users.length + ' users';
    ui['empty-state'].hidden = visible > 0;
    if (!visible) emptyState(state.data.users.length ? 'No matching users' : 'No users yet',
      state.data.users.length ? 'Try another search or status filter.' : 'Add an account to begin managing classroom environments.');
  }

  function option(value, text) {
    const item = node('option', '', text); item.value = value; return item;
  }

  function openDialog(id) {
    if (state.writing || state.expired || !state.data) return;
    const user = state.data.users.find(user => user.id === id);
    if (!user || user.maintenance) return;
    stopTimer(); cancelRead();
    state.target = user;
    state.refreshOnClose = true;
    ui['dialog-title'].textContent = 'Manage ' + user.id;
    ui.operation.replaceChildren();
    for (const action of [user.locked ? 'unlock' : 'lock', 'disconnect', 'restart', 'password', 'template', 'class']) {
      if (action === 'lock' && user.id === config.adminId) continue;
      const item = option(action, labels[action]);
      item.disabled = (action === 'class' && !Object.keys(state.data.classes || {}).length) || (action === 'restart' && user.locked);
      if (action === 'restart' && user.locked) item.textContent += ' (unlock first)';
      ui.operation.append(item);
    }
    updateOperation();
    ui['account-dialog'].showModal();
    updateControls();
  }

  function updateOperation() {
    const action = ui.operation.value;
    notice(ui['dialog-error'], '');
    ui['operation-description'].textContent = descriptions[action];
    ui['password-fields'].hidden = action !== 'password';
    for (const id of ['new-password', 'confirm-password']) {
      ui[id].value = ''; ui[id].required = action === 'password'; ui[id].setCustomValidity('');
    }
    ui['assignment-field'].hidden = action !== 'template' && action !== 'class';
    ui['assignment-label'].textContent = action === 'class' ? 'Class' : 'Template';
    ui.assignment.replaceChildren();
    if (action === 'class' || action === 'template') {
      const names = action === 'class' ? Object.keys(state.data.classes || {}).sort() : ['default', ...Object.keys(state.data.templates || {}).sort()];
      ui.assignment.append(option('', action === 'class' ? 'Choose a class' : 'Choose a template'));
      for (const name of names) ui.assignment.append(option(name, action === 'class' ? name + ' · ' + state.data.classes[name] : name));
      ui.assignment.value = action === 'class' ? state.target.class || '' : templateFor(state.target);
      if (!ui.assignment.value) ui.assignment.value = '';
    }
    ui.assignment.required = action === 'class' || action === 'template';
    ui['self-notice'].hidden = state.target.id !== config.adminId;
    ui['apply-action'].textContent = labels[action];
  }

  function setWriting(value) {
    state.writing = value;
    for (const control of ui['account-form'].querySelectorAll('button, input, select')) control.disabled = value;
    ui['apply-action'].textContent = value ? 'Applying…' : labels[ui.operation.value];
    updateControls();
  }

  async function submitAction(event) {
    event.preventDefault();
    if (state.writing || state.expired || !state.target) return;
    const action = ui.operation.value;
    if (action === 'password') {
      ui['confirm-password'].setCustomValidity(ui['new-password'].value === ui['confirm-password'].value ? '' : 'Passwords do not match.');
      ui['new-password'].setCustomValidity(new TextEncoder().encode(ui['new-password'].value).length <= 72 ? '' : 'Use a password of at most 72 UTF-8 bytes.');
    }
    if (!ui['account-form'].reportValidity()) return;
    const id = state.target.id;
    const value = action === 'password' ? ui['new-password'].value : action === 'class' || action === 'template' ? ui.assignment.value : '';
    stopTimer(); cancelRead(); setWriting(true);
    notice(ui['dialog-error'], ''); notice(ui['action-notice'], '');
    const controller = new AbortController();
    const closeAfterAction = () => { state.refreshOnClose = false; ui['account-dialog'].close(); };
    const timeout = setTimeout(() => controller.abort(), 60000);
    let refreshAfter = false;
    try {
      const response = await fetch(config.apiUrl, { method: 'POST', credentials: 'same-origin', signal: controller.signal,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': state.csrf }, body: JSON.stringify({ user_id: id, action, value }) });
      if (response.status === 401 || response.status === 403 || response.headers.get('X-Linuxus-Reauthenticate') === 'true') {
        if (response.headers.get('X-Linuxus-Account-Updated') === 'true') notice(ui['action-notice'],
          response.ok ? labels[action] + ' completed for ' + id + '.' : (await response.text()).trim(), response.ok ? 'success' : 'warning');
        sessionExpired(); closeAfterAction(); return;
      }
      if (!response.ok) {
        const error = (await response.text()).trim() || 'The action could not be completed.';
        if (response.headers.get('X-Linuxus-Account-Updated') === 'true') {
          notice(ui['action-notice'], error, 'warning'); refreshAfter = true; closeAfterAction();
        } else throw new Error(error);
      } else {
        notice(ui['action-notice'], labels[action] + ' completed for ' + id + '.', 'success');
        refreshAfter = true; closeAfterAction();
      }
    } catch (error) {
      notice(ui['dialog-error'], error.name === 'AbortError' ? 'The request timed out. Close this dialog and refresh to check whether the change completed before retrying.' : error.message);
    } finally {
      clearTimeout(timeout);
      setWriting(false);
      if (refreshAfter && !state.expired) await refresh();
      else schedule();
    }
  }

  ui.refresh.addEventListener('click', refresh);
  ui.search.addEventListener('input', filterRows);
  ui.filter.addEventListener('change', filterRows);
  ui['auto-refresh'].addEventListener('change', () => { stopTimer(); if (canPoll()) refresh(); else updateControls(); });
  ui.operation.addEventListener('change', updateOperation);
  for (const id of ['new-password', 'confirm-password']) ui[id].addEventListener('input', () => {
    ui['new-password'].setCustomValidity(''); ui['confirm-password'].setCustomValidity('');
  });
  ui['account-form'].addEventListener('submit', submitAction);
  for (const id of ['close-dialog', 'cancel-dialog']) ui[id].addEventListener('click', () => {
    if (!state.writing) ui['account-dialog'].close();
  });
  ui['account-dialog'].addEventListener('cancel', event => { if (state.writing) event.preventDefault(); });
  ui['account-dialog'].addEventListener('close', () => {
    ui['new-password'].value = ''; ui['confirm-password'].value = ''; state.target = null;
    if (canPoll() && state.refreshOnClose) refresh(); else schedule();
  });
  document.addEventListener('visibilitychange', () => {
    stopTimer();
    if (document.hidden) cancelRead();
    else if (canPoll()) refresh();
    updateControls();
  });
  window.addEventListener('pagehide', () => { state.disposed = true; stopTimer(); cancelRead(); });
  window.addEventListener('pageshow', event => { if (event.persisted) { state.disposed = false; if (canPoll()) refresh(); } });
  refresh();
})();
