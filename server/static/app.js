let chart = null;
let currentLocationId = null;
let currentTab = 'timeline';

// ── タブ切り替え ────────────────────────────────────────────────

document.querySelectorAll('#main-tabs .nav-link').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('#main-tabs .nav-link').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    currentTab = btn.dataset.tab;
    if (currentLocationId) showTab(currentTab);
  });
});

function showTab(tab) {
  document.getElementById('main-placeholder').classList.add('d-none');
  document.getElementById('tab-timeline').classList.toggle('d-none', tab !== 'timeline');
  document.getElementById('tab-events').classList.toggle('d-none', tab !== 'events');
  document.getElementById('tab-sessions').classList.toggle('d-none', tab !== 'sessions');
  document.getElementById('tab-players').classList.toggle('d-none', tab !== 'players');
  document.getElementById('tab-visitors').classList.toggle('d-none', tab !== 'visitors');
  if (tab === 'timeline') loadTimeline();
  if (tab === 'events')   loadEvents();
  if (tab === 'sessions') loadSessions();
  if (tab === 'players')  loadPlayers();
  if (tab === 'visitors') loadVisitors();
}

// ── ロケーション一覧 ────────────────────────────────────────────

async function loadLocations() {
  const start = document.getElementById('loc-start').value;
  const end   = document.getElementById('loc-end').value;
  const params = new URLSearchParams();
  if (start) params.set('start', toISO(start));
  if (end)   params.set('end',   toISO(end));

  const res = await fetch('/api/locations?' + params);
  const locations = await res.json();

  const list = document.getElementById('location-list');
  if (locations.length === 0) {
    list.innerHTML = '<div class="list-group-item text-muted small">該当なし</div>';
    return;
  }

  list.innerHTML = '';
  for (const loc of locations) {
    const a = document.createElement('a');
    a.href = '#';
    a.className = 'list-group-item list-group-item-action location-item py-2';
    a.dataset.locationId = loc.location_id;
    a.innerHTML = `
      <div class="fw-semibold text-truncate">${loc.world_id}</div>
      <small class="text-muted">${loc.location_id}</small>
      <div class="d-flex justify-content-between mt-1">
        <span class="badge bg-secondary">${fmtDate(loc.first_seen)}</span>
        <span class="badge bg-primary">${fmtDate(loc.last_seen)}</span>
      </div>`;
    a.addEventListener('click', e => {
      e.preventDefault();
      selectLocation(loc.location_id);
    });
    list.appendChild(a);
  }

  if (currentLocationId) setActiveItem(currentLocationId);
}

// ── ロケーション選択 ────────────────────────────────────────────

function selectLocation(locationId) {
  currentLocationId = locationId;
  setActiveItem(locationId);
  document.getElementById('selected-label').textContent = locationId;
  showTab(currentTab);
}

// ── 人数推移 ────────────────────────────────────────────────────

async function loadTimeline() {
  if (!currentLocationId) return;
  const params = new URLSearchParams();
  const start = document.getElementById('tl-start').value;
  const end   = document.getElementById('tl-end').value;
  if (start) params.set('start', toISO(start));
  if (end)   params.set('end',   toISO(end));

  const res = await fetch(`/api/locations/${encodeURIComponent(currentLocationId)}/presence-timeline?${params}`);
  const data = await res.json();
  renderChart(data);
}

function renderChart(data) {
  if (chart) chart.destroy();
  const ctx = document.getElementById('timeline-chart').getContext('2d');
  chart = new Chart(ctx, {
    type: 'line',
    data: {
      datasets: [{
        label: '人数',
        data: data.map(d => ({ x: new Date(d.timestamp), y: d.count })),
        borderColor: 'rgb(13, 110, 253)',
        backgroundColor: 'rgba(13, 110, 253, 0.08)',
        stepped: true,
        fill: true,
        pointRadius: data.length < 200 ? 3 : 0,
        borderWidth: 2,
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          type: 'time',
          time: { displayFormats: { minute: 'HH:mm', hour: 'MM/dd HH:mm' } }
        },
        y: {
          beginAtZero: true,
          ticks: { stepSize: 1 },
          title: { display: true, text: '人数' }
        }
      },
      plugins: {
        tooltip: {
          callbacks: {
            title: items => new Date(items[0].parsed.x).toLocaleString('ja-JP'),
            label: items => ` ${items.parsed.y} 人`
          }
        },
        legend: { display: false }
      }
    }
  });
}

// ── 入退場ログ ────────────────────────────────────────────────

let evOrder = 'desc';

async function loadEvents() {
  if (!currentLocationId) return;
  const params = new URLSearchParams({ order: evOrder });
  const start = document.getElementById('ev-start').value;
  const end   = document.getElementById('ev-end').value;
  if (start) params.set('start', toISO(start));
  if (end)   params.set('end',   toISO(end));

  document.getElementById('ev-sort-indicator').textContent = evOrder === 'asc' ? ' ▲' : ' ▼';

  const res = await fetch(`/api/locations/${encodeURIComponent(currentLocationId)}/events?${params}`);
  const events = await res.json();
  renderEventTable(events);
}

function renderEventTable(events) {
  const tbody = document.getElementById('event-tbody');
  if (events.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">データなし</td></tr>';
    return;
  }
  tbody.innerHTML = events.map(ev => {
    const isJoin = ev.event_type === 'join';
    const badge = isJoin
      ? '<span class="badge bg-success" style="width:4.5em">JOIN</span>'
      : '<span class="badge bg-danger" style="width:4.5em">LEAVE</span>';
    return `<tr>
      <td class="text-nowrap small">${fmtDateFull(ev.timestamp)}</td>
      <td>${badge}</td>
      <td class="text-truncate" style="max-width:160px">${escHtml(ev.display_name)}</td>
    </tr>`;
  }).join('');
}

// ── 在室中 ──────────────────────────────────────────────────────

const plSort = { by: 'internal_id', order: 'asc' };

async function loadPlayers() {
  if (!currentLocationId) return;
  const params = new URLSearchParams({ sort_by: plSort.by, order: plSort.order });
  const res = await fetch(`/api/locations/${encodeURIComponent(currentLocationId)}/players?${params}`);
  const players = await res.json();
  const tbody = document.getElementById('player-tbody');

  document.querySelectorAll('#player-thead th[data-sort]').forEach(th => {
    th.querySelector('.sort-indicator').textContent = th.dataset.sort === plSort.by
      ? (plSort.order === 'asc' ? ' ▲' : ' ▼') : '';
  });

  if (players.length === 0) {
    tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">在室中のプレイヤーなし</td></tr>';
    return;
  }
  tbody.innerHTML = players.map(p => `<tr>
    <td class="text-end text-muted">${p.internal_id ?? '—'}</td>
    <td>${escHtml(p.display_name)}</td>
    <td class="small text-nowrap">${fmtDateFull(p.join_ts)}</td>
  </tr>`).join('');
}

// ── 訪れた人 ────────────────────────────────────────────────────

const viSort = { by: 'last_seen', order: 'desc' };

async function loadVisitors() {
  if (!currentLocationId) return;
  const params = new URLSearchParams({ sort_by: viSort.by, order: viSort.order });
  const res = await fetch(`/api/locations/${encodeURIComponent(currentLocationId)}/visitors?${params}`);
  const visitors = await res.json();
  const tbody = document.getElementById('visitor-tbody');
  document.getElementById('vi-count').textContent = `${visitors.length} 人`;

  // ヘッダーのソートインジケーター更新
  document.querySelectorAll('#visitor-thead th[data-sort]').forEach(th => {
    const col = th.dataset.sort;
    const indicator = th.querySelector('.sort-indicator');
    if (col === viSort.by) {
      indicator.textContent = viSort.order === 'asc' ? ' ▲' : ' ▼';
    } else {
      indicator.textContent = '';
    }
  });

  if (visitors.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">データなし</td></tr>';
    return;
  }
  tbody.innerHTML = visitors.map(v => `<tr>
    <td class="text-truncate" style="max-width:200px">
      <a href="#" class="visitor-name-link" data-user-id="${escHtml(v.user_id)}" data-display-name="${escHtml(v.display_name)}">${escHtml(v.display_name)}</a>
    </td>
    <td class="small text-nowrap">${fmtDateFull(v.first_seen)}</td>
    <td class="small text-nowrap">${fmtDateFull(v.last_seen)}</td>
    <td class="text-end small">${v.join_count}回</td>
    <td class="text-end small">${v.total_duration_seconds != null ? fmtDuration(v.total_duration_seconds) : '—'}</td>
  </tr>`).join('');
  tbody.querySelectorAll('.visitor-name-link').forEach(link => {
    link.addEventListener('click', e => {
      e.preventDefault();
      openPlayerSessions(link.dataset.userId, link.dataset.displayName);
    });
  });
}

// ── プレイヤーセッションモーダル ────────────────────────────────

async function openPlayerSessions(userId, displayName) {
  document.getElementById('session-modal-title').textContent = displayName + ' のセッション';
  const tbody = document.getElementById('session-modal-tbody');
  tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">読み込み中...</td></tr>';
  bootstrap.Modal.getOrCreateInstance(document.getElementById('session-modal')).show();

  const params = new URLSearchParams({ location_id: currentLocationId, order: 'desc' });
  const res = await fetch(`/api/players/${encodeURIComponent(userId)}/sessions?${params}`);
  const sessions = await res.json();

  if (sessions.length === 0) {
    tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">データなし</td></tr>';
    return;
  }
  tbody.innerHTML = sessions.map(s => {
    const duration = s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—';
    return `<tr>
      <td class="small text-nowrap">${fmtDateFull(s.join_ts)}</td>
      <td class="small text-nowrap">${leaveCellHtml(s)}</td>
      <td class="text-end small">${duration}</td>
    </tr>`;
  }).join('');
}

// ── セッション一覧 ──────────────────────────────────────────────

const ssSort = { by: 'join_ts', order: 'asc' };

async function loadSessions() {
  if (!currentLocationId) return;
  const params = new URLSearchParams({ sort_by: ssSort.by, order: ssSort.order });
  const start = document.getElementById('ss-start').value;
  const end   = document.getElementById('ss-end').value;
  if (start) params.set('start', toISO(start));
  if (end)   params.set('end',   toISO(end));

  const res = await fetch(`/api/locations/${encodeURIComponent(currentLocationId)}/sessions?${params}`);
  const sessions = await res.json();
  renderSessionTable(sessions);
}

function renderSessionTable(sessions) {
  const tbody = document.getElementById('session-tbody');

  document.querySelectorAll('#session-thead th[data-sort]').forEach(th => {
    th.querySelector('.sort-indicator').textContent = th.dataset.sort === ssSort.by
      ? (ssSort.order === 'asc' ? ' ▲' : ' ▼') : '';
  });

  if (sessions.length === 0) {
    tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">データなし</td></tr>';
    return;
  }
  tbody.innerHTML = sessions.map(s => {
    const duration = s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—';
    return `<tr>
      <td class="text-truncate" style="max-width:160px">${escHtml(s.display_name)}</td>
      <td class="small text-nowrap">${fmtDateFull(s.join_ts)}</td>
      <td class="small text-nowrap">${leaveCellHtml(s)}</td>
      <td class="text-end small">${duration}</td>
    </tr>`;
  }).join('');
}

// ── セル生成ヘルパー ────────────────────────────────────────────

function leaveCellHtml(s) {
  if (!s.leave_ts) return '<span class="badge bg-success">在室中</span>';
  const dateStr = fmtDateFull(s.leave_ts);
  if (s.is_estimated_leave) {
    return `${dateStr} <span class="badge rounded-pill bg-warning text-dark" title="退室時刻を使用した推定値です">!</span>`;
  }
  return dateStr;
}

// ── ユーティリティ ──────────────────────────────────────────────

function setActiveItem(locationId) {
  document.querySelectorAll('.location-item').forEach(el => {
    el.classList.toggle('active', el.dataset.locationId === locationId);
  });
}

function toISO(localDatetime) {
  return new Date(localDatetime).toISOString();
}

function fmtDate(iso) {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  });
}

function fmtDateFull(iso) {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  });
}

function escHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function fmtDuration(sec) {
  if (sec < 60)   return `${sec}秒`;
  if (sec < 3600) return `${Math.floor(sec / 60)}分${sec % 60}秒`;
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${h}時間${m}分`;
}

// ── イベント登録 ────────────────────────────────────────────────

document.querySelectorAll('#visitor-thead th[data-sort]').forEach(th => {
  th.addEventListener('click', () => {
    const col = th.dataset.sort;
    if (viSort.by === col) { viSort.order = viSort.order === 'asc' ? 'desc' : 'asc'; }
    else { viSort.by = col; viSort.order = 'desc'; }
    loadVisitors();
  });
});

document.querySelectorAll('#session-thead th[data-sort]').forEach(th => {
  th.addEventListener('click', () => {
    const col = th.dataset.sort;
    if (ssSort.by === col) { ssSort.order = ssSort.order === 'asc' ? 'desc' : 'asc'; }
    else { ssSort.by = col; ssSort.order = 'desc'; }
    loadSessions();
  });
});

document.querySelectorAll('#player-thead th[data-sort]').forEach(th => {
  th.addEventListener('click', () => {
    const col = th.dataset.sort;
    if (plSort.by === col) { plSort.order = plSort.order === 'asc' ? 'desc' : 'asc'; }
    else { plSort.by = col; plSort.order = 'asc'; }
    loadPlayers();
  });
});

document.getElementById('ev-ts-header').addEventListener('click', () => {
  evOrder = evOrder === 'asc' ? 'desc' : 'asc';
  loadEvents();
});

document.getElementById('loc-search-btn').addEventListener('click', loadLocations);
document.getElementById('tl-update-btn').addEventListener('click', loadTimeline);
document.getElementById('ev-update-btn').addEventListener('click', loadEvents);
document.getElementById('ss-update-btn').addEventListener('click', loadSessions);
document.getElementById('pl-reload-btn').addEventListener('click', loadPlayers);
document.getElementById('vi-reload-btn').addEventListener('click', loadVisitors);

loadLocations();
