let chart = null;
let currentInstanceId = null;
let currentInstanceData = null;
let currentTab = 'timeline';

// ── タブ切り替え ────────────────────────────────────────────────

document.querySelectorAll('#main-tabs .nav-link').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('#main-tabs .nav-link').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    currentTab = btn.dataset.tab;
    if (currentInstanceId) showTab(currentTab);
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
  if (tab === 'events') loadEvents();
  if (tab === 'sessions') loadSessions();
  if (tab === 'players') loadPlayers();
  if (tab === 'visitors') loadVisitors();
}

// ── インスタンス一覧 ─────────────────────────────────────────────

async function loadLocations() {
  const start = document.getElementById('loc-start').value;
  const end = document.getElementById('loc-end').value;
  const params = new URLSearchParams();
  if (start) params.set('start', toISO(start));
  if (end) params.set('end', toISO(end));

  const res = await fetch('/api/instances?' + params);
  const instances = await res.json();

  const list = document.getElementById('location-list');
  if (instances.length === 0) {
    list.innerHTML = '<div class="list-group-item text-muted small">該当なし</div>';
    return;
  }

  list.innerHTML = '';
  for (const inst of instances) {
    const a = document.createElement('a');
    a.href = '#';
    a.className = 'list-group-item list-group-item-action location-item py-2';
    a.dataset.instanceId = inst.id;
    const rangeBadge = inst.closed_at
      ? `<span class="badge bg-secondary">${fmtDate(inst.opened_at)} 〜 ${fmtDate(inst.closed_at)}</span>`
      : `<span class="badge bg-success">${fmtDate(inst.opened_at)} 〜</span>`;
    const countBadge = !inst.closed_at && inst.user_count > 0
      ? `<span class="badge bg-warning text-dark">${inst.user_count}人</span>`
      : '';
    a.innerHTML = `
      <div class="fw-semibold text-truncate">${escHtml(inst.world_id)}</div>
      <small class="text-muted">${escHtml(inst.location_id)}</small>
      <div class="d-flex align-items-center gap-1 mt-1 small">
        ${rangeBadge}${countBadge}
      </div>`;
    a.addEventListener('click', e => {
      e.preventDefault();
      selectInstance(inst);
    });
    list.appendChild(a);
  }

  if (currentInstanceId) {
    setActiveItem(currentInstanceId);
  } else {
    const autoId = Number(new URLSearchParams(location.search).get('instance'));
    if (autoId) {
      const inst = instances.find(i => i.id === autoId);
      if (inst) selectInstance(inst);
    }
  }
}

// ── インスタンス選択 ─────────────────────────────────────────────

function selectInstance(inst) {
  currentInstanceId = inst.id;
  currentInstanceData = inst;
  setActiveItem(inst.id);
  const url = new URL(location.href);
  url.searchParams.set('instance', inst.id);
  history.replaceState(null, '', url);
  showTab(currentTab);
  // モバイル: 詳細画面へ切り替え
  if (window.innerWidth < 768) {
    document.getElementById('col-sidebar').classList.add('d-none');
    document.getElementById('col-main').classList.remove('d-none');
  }
}

document.getElementById('back-btn').addEventListener('click', () => {
  document.getElementById('col-main').classList.add('d-none');
  document.getElementById('col-sidebar').classList.remove('d-none');
});

// ── 人数推移 ────────────────────────────────────────────────────

async function loadTimeline() {
  if (!currentInstanceId) return;
  const params = new URLSearchParams();
  const start = document.getElementById('tl-start').value;
  const end = document.getElementById('tl-end').value;
  if (start) params.set('start', toISO(start));
  if (end) params.set('end', toISO(end));

  const res = await fetch(`/api/instances/${currentInstanceId}/presence-timeline?${params}`);
  const data = await res.json();
  renderChart(data);
}

function renderChart(data) {
  if (chart) chart.destroy();
  const ctx = document.getElementById('timeline-chart').getContext('2d');
  const isOngoing = currentInstanceData && !currentInstanceData.closed_at;
  const xMax = isOngoing ? new Date() : undefined;
  const points = data.map(d => ({ x: new Date(d.timestamp), y: d.count, displayName: d.display_name }));
  if (isOngoing && points.length > 0) {
    points.push({ x: xMax, y: points[points.length - 1].y, displayName: null });
  }
  chart = new Chart(ctx, {
    type: 'line',
    data: {
      datasets: [{
        label: '人数',
        data: points,
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
          time: { displayFormats: { minute: 'HH:mm', hour: 'MM/dd HH:mm' } },
          max: xMax,
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
            label: items => ` ${items.parsed.y} 人`,
            afterLabel: items => {
              const name = items.dataset.data[items.dataIndex].displayName;
              return name ? ` ${name}` : '';
            }
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
  if (!currentInstanceId) return;
  const params = new URLSearchParams({ order: evOrder });
  const start = document.getElementById('ev-start').value;
  const end = document.getElementById('ev-end').value;
  if (start) params.set('start', toISO(start));
  if (end) params.set('end', toISO(end));

  document.getElementById('ev-sort-indicator').textContent = evOrder === 'asc' ? ' ▲' : ' ▼';

  const res = await fetch(`/api/instances/${currentInstanceId}/events?${params}`);
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
let currentPlayers = [];

async function loadPlayers() {
  if (!currentInstanceId) return;
  const params = new URLSearchParams({ sort_by: plSort.by, order: plSort.order });
  const res = await fetch(`/api/instances/${currentInstanceId}/players?${params}`);
  currentPlayers = await res.json();
  const tbody = document.getElementById('player-tbody');

  document.querySelectorAll('#player-thead th[data-sort]').forEach(th => {
    th.querySelector('.sort-indicator').textContent = th.dataset.sort === plSort.by
      ? (plSort.order === 'asc' ? ' ▲' : ' ▼') : '';
  });

  document.getElementById('pl-count').textContent = `${currentPlayers.length} 人`;
  if (currentPlayers.length === 0) {
    tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">在室中のプレイヤーなし</td></tr>';
    return;
  }
  tbody.innerHTML = currentPlayers.map(p => `<tr>
    <td class="text-end text-muted">${p.internal_id ?? '—'}</td>
    <td>${escHtml(p.display_name)}</td>
    <td class="small">${p.discord_id ? escHtml(p.discord_id) : '<span class="text-muted">未登録</span>'}</td>
    <td class="small text-nowrap">${fmtDateFull(p.join_ts)}</td>
  </tr>`).join('');
}

// ── 訪れた人 ────────────────────────────────────────────────────

const viSort = { by: 'last_seen', order: 'desc' };

async function loadVisitors() {
  if (!currentInstanceId) return;
  const params = new URLSearchParams({ sort_by: viSort.by, order: viSort.order });
  const res = await fetch(`/api/instances/${currentInstanceId}/visitors?${params}`);
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
  const worldId = currentInstanceData?.world_id ?? '';
  document.getElementById('session-modal-player-link').href =
    `/player.html?user_id=${encodeURIComponent(userId)}&display_name=${encodeURIComponent(displayName)}` +
    (worldId ? `&world_id=${encodeURIComponent(worldId)}` : '');
  const tbody = document.getElementById('session-modal-tbody');
  const tlContainer = document.getElementById('session-modal-timeline');
  tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">読み込み中...</td></tr>';
  tlContainer.innerHTML = '';
  bootstrap.Modal.getOrCreateInstance(document.getElementById('session-modal')).show();

  const params = new URLSearchParams({ instance_id: currentInstanceId, order: 'asc' });
  const res = await fetch(`/api/players/${encodeURIComponent(userId)}/sessions?${params}`);
  const sessions = await res.json();

  if (sessions.length === 0) {
    tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">データなし</td></tr>';
    return;
  }

  // タイムラインを描画
  if (currentInstanceData) renderPlayerTimeline(sessions, currentInstanceData);

  // テーブルを描画 (data-idx 付き)
  tbody.innerHTML = sessions.map((s, i) => {
    const duration = s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—';
    return `<tr data-idx="${i}">
      <td class="small text-nowrap">${fmtDateFull(s.join_ts)}</td>
      <td class="small text-nowrap">${leaveCellHtml(s)}</td>
      <td class="text-end small">${duration}</td>
    </tr>`;
  }).join('');

  // ホバー連動
  const svg = document.getElementById('player-tl-svg');
  if (!svg) return;

  const highlight = (idx, on) => {
    svg.querySelectorAll(`.tl-bar[data-idx="${idx}"]`).forEach(el => {
      el.setAttribute('fill', on ? 'rgba(13,110,253,0.92)' : 'rgba(13,110,253,0.55)');
      el.setAttribute('stroke', on ? 'rgba(13,110,253,1)' : 'none');
    });
    tbody.querySelectorAll(`tr[data-idx="${idx}"]`).forEach(el => {
      el.classList.toggle('table-primary', on);
    });
  };

  tbody.querySelectorAll('tr[data-idx]').forEach(tr => {
    tr.addEventListener('mouseenter', () => highlight(tr.dataset.idx, true));
    tr.addEventListener('mouseleave', () => highlight(tr.dataset.idx, false));
  });

  svg.querySelectorAll('.tl-bar').forEach(bar => {
    bar.style.cursor = 'pointer';
    bar.addEventListener('mouseenter', () => highlight(bar.dataset.idx, true));
    bar.addEventListener('mouseleave', () => highlight(bar.dataset.idx, false));
  });
}

function renderPlayerTimeline(sessions, instanceData) {
  const container = document.getElementById('session-modal-timeline');

  const instStart = new Date(instanceData.opened_at).getTime();
  const instEnd = instanceData.closed_at ? new Date(instanceData.closed_at).getTime() : Date.now();
  const total = instEnd - instStart;
  if (total <= 0) { container.innerHTML = ''; return; }

  const VW = 1000;
  const BAR_H = 26;

  const toX = ts => ((new Date(ts).getTime() - instStart) / total) * VW;

  const bars = sessions.map((s, i) => {
    const x1 = toX(s.join_ts);
    const x2 = s.leave_ts ? toX(s.leave_ts) : VW;
    const w = Math.max(3, x2 - x1);
    return `<rect class="tl-bar" data-idx="${i}" x="${x1.toFixed(1)}" y="0" width="${w.toFixed(1)}" height="${BAR_H}" fill="rgba(13,110,253,0.55)" stroke="none" rx="2"/>`;
  }).join('');

  const fmt = ts => new Date(ts).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  });

  container.innerHTML = `
    <svg id="player-tl-svg" viewBox="0 0 ${VW} ${BAR_H}" preserveAspectRatio="none"
         style="width:100%;height:${BAR_H}px;display:block">
      <rect x="0" y="0" width="${VW}" height="${BAR_H}" fill="#dee2e6" rx="3"/>
      ${bars}
    </svg>
    <div class="d-flex justify-content-between mt-1" style="font-size:11px;color:#6c757d">
      <span>${fmt(instanceData.opened_at)}</span>
      <span>${fmt(instanceData.closed_at ?? new Date(instEnd).toISOString())}</span>
    </div>`;
}

// ── セッション一覧 ──────────────────────────────────────────────

const ssSort = { by: 'leave_ts', order: 'asc' };

async function loadSessions() {
  if (!currentInstanceId) return;
  const params = new URLSearchParams({ sort_by: ssSort.by, order: ssSort.order });
  const start = document.getElementById('ss-start').value;
  const end = document.getElementById('ss-end').value;
  if (start) params.set('start', toISO(start));
  if (end) params.set('end', toISO(end));

  const res = await fetch(`/api/instances/${currentInstanceId}/sessions?${params}`);
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

function setActiveItem(instanceId) {
  document.querySelectorAll('.location-item').forEach(el => {
    el.classList.toggle('active', Number(el.dataset.instanceId) === instanceId);
  });
}

function toISO(localDatetime) {
  return new Date(localDatetime).toISOString();
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

// ── インスタンス比較 ────────────────────────────────────────────

document.getElementById('tl-compare-btn').addEventListener('click', async () => {
  if (!currentInstanceId) return;
  const list = document.getElementById('compare-instance-list');
  list.innerHTML = '<div class="list-group-item text-muted small">読み込み中...</div>';
  bootstrap.Modal.getOrCreateInstance(document.getElementById('compare-modal')).show();

  const res = await fetch('/api/instances');
  const instances = await res.json();
  const curStart = new Date(currentInstanceData.opened_at);
  const curEnd = currentInstanceData.closed_at ? new Date(currentInstanceData.closed_at) : new Date();
  const others = instances.filter(inst => {
    if (inst.id === currentInstanceId) return false;
    const s = new Date(inst.opened_at);
    const e = inst.closed_at ? new Date(inst.closed_at) : new Date();
    return curStart < e && s < curEnd;  // 期間が1秒でもかぶっていれば表示
  });

  if (others.length === 0) {
    list.innerHTML = '<div class="list-group-item text-muted small">比較できる他のインスタンスがありません</div>';
    return;
  }

  list.innerHTML = '';
  for (const inst of others) {
    const a = document.createElement('a');
    a.href = '#';
    a.className = 'list-group-item list-group-item-action py-2';
    const rangeBadge = inst.closed_at
      ? `<span class="badge bg-secondary">${fmtDate(inst.opened_at)} 〜 ${fmtDate(inst.closed_at)}</span>`
      : `<span class="badge bg-success">${fmtDate(inst.opened_at)} 〜</span>`;
    a.innerHTML = `
      <div class="fw-semibold text-truncate">${escHtml(inst.world_id)}</div>
      <small class="text-muted">${escHtml(inst.location_id)}</small>
      <div class="mt-1 small">${rangeBadge}</div>`;
    a.addEventListener('click', e => {
      e.preventDefault();
      bootstrap.Modal.getInstance(document.getElementById('compare-modal')).hide();
      window.open(`/compare.html?id1=${currentInstanceId}&id2=${inst.id}`, '_blank');
    });
    list.appendChild(a);
  }
});

document.getElementById('loc-search-btn').addEventListener('click', loadLocations);
document.getElementById('tl-update-btn').addEventListener('click', loadTimeline);
document.getElementById('ev-update-btn').addEventListener('click', loadEvents);
document.getElementById('ss-update-btn').addEventListener('click', loadSessions);
document.getElementById('pl-reload-btn').addEventListener('click', loadPlayers);
document.getElementById('vi-reload-btn').addEventListener('click', loadVisitors);

document.getElementById('pl-copy-discord-btn').addEventListener('click', () => {
  const mentions = currentPlayers
    .filter(p => p.discord_id)
    .map(p => `@${p.discord_id}`);
  const btn = document.getElementById('pl-copy-discord-btn');
  if (mentions.length === 0) {
    btn.textContent = 'IDなし';
    setTimeout(() => { btn.textContent = '全員のDiscordIDをコピー'; }, 2000);
    return;
  }
  const text = mentions.join(' ') + ' ';
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).then(() => {
      btn.textContent = `コピー済み (${mentions.length}人)`;
      setTimeout(() => { btn.textContent = '全員のDiscordIDをコピー'; }, 2000);
    });
  } else {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
    btn.textContent = `コピー済み (${mentions.length}人)`;
    setTimeout(() => { btn.textContent = '全員のDiscordIDをコピー'; }, 2000);
  }
});

loadLocations();
