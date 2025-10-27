function setText(id, text) {
  const el = document.getElementById(id);
  if (el) el.textContent = text;
}

async function fetchJSON(url) {
  const r = await fetch(url, { cache: 'no-store' });
  if (!r.ok) throw new Error(`${r.status} ${r.statusText}`);
  return r.json();
}

function renderInfo(data) {
  const appInfo = document.getElementById('app-info');
  appInfo.innerHTML = `
    <div class="info-item"><div class="info-label">Name:</div><div>${data.name}</div></div>
    <div class="info-item"><div class="info-label">Version:</div><div>${data.version}</div></div>
    <div class="info-item"><div class="info-label">Environment:</div><div>${data.environment}</div></div>
    <div class="info-item"><div class="info-label">Build Time:</div><div>${data.buildTime || 'Not available'}</div></div>
    <div class="info-item"><div class="info-label">Uptime:</div><div id="uptime">${data.uptime}</div></div>
    <div class="info-item"><div class="info-label">Hostname:</div><div>${data.hostname}</div></div>
  `;
}

async function refreshInfo() {
  try {
    const data = await fetchJSON('/api/info');
    renderInfo(data);
    document.getElementById('error').hidden = true;
  } catch (e) {
    const err = document.getElementById('error');
    err.textContent = `Error loading app info: ${e.message}`;
    err.hidden = false;
  }
}

async function refreshHealth() {
  // health endpoints return {"status":"..."}; treat non-200 as bad
  const tryFetch = async (url) => {
    try { const r = await fetch(url, { cache: 'no-store' }); return r.ok; }
    catch { return false; }
  };
  setText('live-status', (await tryFetch('/live')) ? 'OK' : 'FAIL');
  setText('ready-status', (await tryFetch('/ready')) ? 'OK' : 'WAIT');
}

document.addEventListener('DOMContentLoaded', async () => {
  await refreshInfo();
  await refreshHealth();
  // Light auto-refresh of uptime/health every 5s
  setInterval(refreshInfo, 5000);
  setInterval(refreshHealth, 5000);
});
