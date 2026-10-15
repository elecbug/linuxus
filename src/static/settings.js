(() => {
  'use strict';

  const form = document.getElementById('password-form');
  const controls = document.getElementById('password-controls');
  const status = document.getElementById('password-status');
  const save = document.getElementById('save-password');
  const current = document.getElementById('current-password');
  const password = document.getElementById('account-new-password');
  const confirmation = document.getElementById('account-confirm-password');
  let writing = false;
  let expired = false;

  function clearPasswords() {
    for (const input of [current, password, confirmation]) { input.value = ''; input.setCustomValidity(''); }
  }

  function notice(message, kind = 'danger') {
    status.textContent = message;
    status.hidden = !message;
    status.className = 'notice ' + kind;
    status.setAttribute('role', kind === 'danger' ? 'alert' : 'status');
  }

  for (const input of [current, password, confirmation]) input.addEventListener('input', () => {
    for (const field of [current, password, confirmation]) field.setCustomValidity('');
  });

  form.addEventListener('submit', async event => {
    event.preventDefault();
    if (writing || expired) return;
    for (const input of [current, password]) input.setCustomValidity(new TextEncoder().encode(input.value).length > 72 ? 'Use a password of at most 72 UTF-8 bytes.' : '');
    confirmation.setCustomValidity(password.value === confirmation.value ? '' : 'New passwords do not match.');
    if (!form.reportValidity()) return;
    const body = new URLSearchParams(new FormData(form));
    writing = true;
    controls.disabled = true;
    save.textContent = 'Changing password…';
    notice('');
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000);
    try {
      const response = await fetch(form.action, { method: 'POST', credentials: 'same-origin',
        headers: { Accept: 'application/json' }, body, signal: controller.signal });
      if (response.ok) {
        clearPasswords();
        document.dispatchEvent(new CustomEvent('linuxus:sessionexpired', { detail: { message: 'Password changed. Sign in with your new password.' } }));
        location.assign(document.body.dataset.loginUrl + '?password=changed');
        return;
      }
      const data = await response.json().catch(() => ({}));
      if (response.status === 401) {
        clearPasswords();
        document.dispatchEvent(new Event('linuxus:sessionexpired'));
      }
      notice(data.error || 'Unable to change your password. Please try again later.');
    } catch (error) {
      notice(error.name === 'AbortError' ? 'The request timed out. Sign in again or reload Settings to check the result before retrying.' : 'The connection was interrupted. Sign in again or reload Settings to check the result before retrying.');
    } finally {
      clearTimeout(timeout);
      writing = false;
      controls.disabled = expired;
      save.textContent = 'Change password';
    }
  });

  document.addEventListener('linuxus:viewchange', event => { if (event.detail.view !== 'settings') clearPasswords(); });
  document.addEventListener('linuxus:sessionexpired', () => { expired = true; clearPasswords(); controls.disabled = true; });
  window.addEventListener('pagehide', clearPasswords);
})();
