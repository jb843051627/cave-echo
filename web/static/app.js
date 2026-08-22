async function fetchJSON(url, options) {
  const res = await fetch(url, options);
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      if (body && body.error) message = body.error;
    } catch (e) { /* ignore */ }
    throw new Error(message);
  }
  return res.json();
}

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key === 'text') node.textContent = value;
    else node.setAttribute(key, value);
  }
  for (const child of children) node.appendChild(child);
  return node;
}

function fmt(value) {
  return typeof value === 'number' ? value.toFixed(2) : (value ?? '-');
}

async function loadHealth() {
  try {
    const data = await fetchJSON('/healthz');
    document.getElementById('health').textContent = `服务正常 · 读数 ${data.readings_ingested}`;
    document.getElementById('health').classList.add('ok');
  } catch (e) {
    document.getElementById('health').textContent = '服务不可用';
  }
}

async function loadOverview() {
  const overview = await fetchJSON('/api/overview');
  const tbody = document.querySelector('#sites tbody');
  tbody.replaceChildren();
  for (const summary of overview.sites || []) {
    tbody.appendChild(el('tr', {}, [
      el('td', { text: summary.site.code }),
      el('td', { text: summary.site.name }),
      el('td', { text: summary.site.protection_grade }),
      el('td', { text: summary.site.status }),
      el('td', { text: String(summary.chamber_count) }),
      el('td', { text: String(summary.sensor_count) }),
      el('td', { text: String(summary.active_alert_count) }),
      el('td', { text: `${(summary.completeness * 100).toFixed(0)}%` }),
      el('td', { text: summary.protection_message || '-' }),
    ]));
  }
  const notices = document.getElementById('seasonal');
  notices.replaceChildren(...(overview.seasonal_notices || []).map((n) => el('div', { class: 'notice', text: n })));
}

async function loadAlerts() {
  const data = await fetchJSON('/api/alerts?status=active&limit=50');
  const tbody = document.querySelector('#alerts tbody');
  tbody.replaceChildren();
  for (const alert of data.alerts || []) {
    const actions = el('td', {});
    if (alert.status === 'open') {
      actions.appendChild(el('button', {
        text: '确认',
        onclick: () => act(`/api/alerts/${alert.id}/acknowledge`),
      }));
    }
    if (alert.severity !== 'critical' || alert.status === 'acknowledged') {
      actions.appendChild(el('button', {
        text: '关闭',
        onclick: () => act(`/api/alerts/${alert.id}/close`),
      }));
    }
    tbody.appendChild(el('tr', {}, [
      el('td', { text: alert.kind }),
      el('td', { text: alert.severity }),
      el('td', { text: alert.status }),
      el('td', { text: alert.chamber_id || '-' }),
      el('td', { text: alert.message }),
      el('td', { text: new Date(alert.first_seen_at).toISOString().slice(0, 16).replace('T', ' ') + 'Z' }),
      el('td', { text: String(alert.occurrences) }),
      actions,
    ]));
  }
}

async function act(url) {
  await fetchJSON(url, { method: 'POST' });
  await Promise.all([loadAlerts(), loadOverview()]);
}

async function loadChamber() {
  const chamberID = document.getElementById('chamber').value.trim();
  if (!chamberID) return;
  const trendEl = document.getElementById('trend');
  const gasEl = document.getElementById('gas');
  trendEl.textContent = '加载中…';
  gasEl.textContent = '';
  try {
    const snapshot = await fetchJSON(`/api/chambers/${chamberID}`);
    const t = snapshot.drip_trend || {};
    trendEl.textContent =
      `洞室 ${snapshot.chamber.name}\n` +
      `平均滴速 ${fmt(t.average_rate)} 滴/分 · 近期 ${fmt(t.recent_rate)} · 斜率 ${fmt(t.slope_per_day)}/天（${t.direction}）\n` +
      `活动告警 ${snapshot.active_alerts}`;
    const g = snapshot.latest_gas;
    gasEl.textContent = g
      ? `气体样本：CO₂ ${g.co2_ppm} ppm · O₂ ${g.oxygen_percent}% · Rn ${g.radon_bqm3} Bq/m³`
      : '暂无气体样本';
  } catch (e) {
    trendEl.textContent = `加载失败：${e.message}`;
  }
}

document.getElementById('load-chamber').addEventListener('click', loadChamber);

async function refreshAll() {
  await Promise.all([loadHealth(), loadOverview(), loadAlerts()]);
}

refreshAll();
setInterval(refreshAll, 30000);
