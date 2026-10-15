(() => {
  'use strict';

  const config = document.body.dataset;
  const frame = document.getElementById('terminal-frame');
  const reconnect = document.getElementById('reconnect-terminal');
  const links = Array.from(document.querySelectorAll('[data-view-link], [data-open-view]'));
  const panels = { workspace: document.getElementById('workspace-panel'), admin: document.getElementById('admin-panel') };
  let expired = false;

  function startTerminal() {
    if (!expired && !frame.hasAttribute('src')) frame.src = config.terminalUrl;
  }

  function routeView() {
    const pathname = location.pathname;
    if (panels.admin && (pathname === config.adminUrl || pathname === config.adminUrl + '/')) return 'admin';
    if (pathname === config.serviceUrl) return 'workspace';
    return null;
  }

  function show(view, focus = false) {
    if (!panels[view]) return;
    config.view = view;
    for (const [name, panel] of Object.entries(panels)) if (panel) panel.hidden = name !== view;
    for (const link of links) {
      if (!link.dataset.viewLink) continue;
      if (link.dataset.viewLink === view) link.setAttribute('aria-current', 'page');
      else link.removeAttribute('aria-current');
    }
    document.title = (view === 'admin' ? 'Administration' : 'Workspace') + ' | Linuxus';
    if (view === 'workspace') startTerminal();
    document.dispatchEvent(new CustomEvent('linuxus:viewchange', { detail: { view } }));
    if (focus && !document.getElementById('account-dialog')?.open) {
      document.getElementById(view === 'admin' ? 'page-title' : 'workspace-title').focus({ preventScroll: true });
    }
  }

  for (const link of links) link.addEventListener('click', event => {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
    if (expired) { event.preventDefault(); return; }
    const view = link.dataset.viewLink || link.dataset.openView;
    // The server-rendered links remain usable without JavaScript.
    if (!panels[view]) return;
    event.preventDefault();
    if (view === config.view) return;
    history.pushState(null, '', link.href);
    show(view, true);
  });

  reconnect.hidden = false;
  reconnect.addEventListener('click', () => { if (!expired) frame.src = config.terminalUrl; });
  window.addEventListener('popstate', () => {
    const view = routeView();
    if (view) show(view, true);
  });
  window.addEventListener('pageshow', event => {
    if (event.persisted) show(routeView() || config.view);
  });
  document.addEventListener('linuxus:sessionexpired', () => {
    expired = true;
    reconnect.disabled = true;
    for (const link of links) link.setAttribute('aria-disabled', 'true');
    // Removing the source closes a terminal belonging to a revoked session.
    frame.removeAttribute('src');
  });
  show(routeView() || config.view);
})();
