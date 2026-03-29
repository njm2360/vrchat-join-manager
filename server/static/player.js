const urlParams   = new URLSearchParams(location.search);
const userId      = urlParams.get('user_id') || '';
const displayName = urlParams.get('display_name') || userId;
const worldId     = urlParams.get('world_id') || '';

document.getElementById('player-heading').textContent = displayName + ' のセッション履歴';
document.title = displayName + ' — セッション履歴';

if (worldId) {
  const badge = document.createElement('span');
  badge.className = 'badge bg-secondary ms-2 fw-normal text-truncate';
  badge.style.maxWidth = '320px';
  badge.style.verticalAlign = 'middle';
  badge.title = worldId;
  badge.textContent = worldId;
  document.getElementById('player-heading').appendChild(badge);
}

const now = new Date();
let curYear  = now.getFullYear();
let curMonth = now.getMonth(); // 0-based

document.getElementById('prev-btn').addEventListener('click', () => {
  if (curMonth === 0) { curYear--; curMonth = 11; }
  else curMonth--;
  loadMonth();
});

document.getElementById('next-btn').addEventListener('click', () => {
  if (curMonth === 11) { curYear++; curMonth = 0; }
  else curMonth++;
  loadMonth();
});

// ── データ取得 ──────────────────────────────────────────────────

async function loadMonth() {
  document.getElementById('month-label').textContent =
    `${curYear}年${String(curMonth + 1).padStart(2, '0')}月`;
  document.getElementById('day-list').innerHTML =
    '<div class="text-center text-muted py-4">読み込み中...</div>';

  // 前月末日から開始し、月をまたぐセッションも拾う
  const start = new Date(curYear, curMonth, 0);        // 前月最終日
  const end   = new Date(curYear, curMonth + 1, 1);    // 翌月1日

  const params = new URLSearchParams({
    start: start.toISOString(),
    end:   end.toISOString(),
    order: 'asc',
    limit: 2000,
  });
  if (worldId) params.set('world_id', worldId);

  const res = await fetch(`/api/players/${encodeURIComponent(userId)}/sessions?${params}`);
  const sessions = await res.json();
  renderMonth(sessions);
}

// ── 描画 ────────────────────────────────────────────────────────

function renderMonth(sessions) {
  const daysInMonth = new Date(curYear, curMonth + 1, 0).getDate();
  const nowMs       = Date.now();
  const DAY_MS      = 86400000;
  const DOW         = ['日', '月', '火', '水', '木', '金', '土'];

  const rows = [];

  for (let d = 1; d <= daysInMonth; d++) {
    const dayStart = new Date(curYear, curMonth, d).getTime();
    const dayEnd   = dayStart + DAY_MS;

    const segs = [];
    for (const s of sessions) {
      const sStart = new Date(s.join_ts).getTime();
      const sEnd   = s.leave_ts ? new Date(s.leave_ts).getTime() : nowMs;
      if (sStart >= dayEnd || sEnd <= dayStart) continue;

      const segStart = Math.max(sStart, dayStart);
      const segEnd   = Math.min(sEnd, dayEnd);
      const leftPct  = (segStart - dayStart) / DAY_MS * 100;
      const widthPct = Math.max(0.2, (segEnd - segStart) / DAY_MS * 100);

      segs.push({ s, leftPct, widthPct });
    }

    const dow   = new Date(curYear, curMonth, d).getDay();
    const color = dow === 0 ? 'text-danger' : dow === 6 ? 'text-primary' : '';

    const segsHtml = segs.map(({ s, leftPct, widthPct }) =>
      `<div class="seg"
         style="left:${leftPct.toFixed(3)}%;width:${widthPct.toFixed(3)}%"
         data-join="${esc(fmtFull(s.join_ts))}"
         data-leave="${esc(s.leave_ts ? fmtFull(s.leave_ts) : '在室中')}${s.is_estimated_leave ? ' (推定)' : ''}"
         data-dur="${esc(s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—')}"
       ></div>`
    ).join('');

    rows.push(`
      <div class="day-row">
        <div class="day-label ${color}">
          ${String(curMonth + 1).padStart(2, '0')}/${String(d).padStart(2, '0')} (${DOW[dow]})
        </div>
        <div class="day-bar">${segsHtml}</div>
      </div>`);
  }

  const list = document.getElementById('day-list');
  list.innerHTML = rows.join('') || '<div class="text-center text-muted py-4">データなし</div>';

  // ── ツールチップ ────────────────────────────────────────────
  const tooltip = document.getElementById('tooltip');

  list.querySelectorAll('.seg').forEach(seg => {
    seg.addEventListener('mouseenter', () => {
      tooltip.innerHTML =
        `入室: ${seg.dataset.join}<br>退室: ${seg.dataset.leave}<br>滞在: ${seg.dataset.dur}`;
      tooltip.style.display = 'block';
    });
    seg.addEventListener('mousemove', e => {
      const tx = e.clientX + 14;
      const ty = e.clientY - 10;
      // 右端にはみ出さないよう補正
      const maxX = window.innerWidth - tooltip.offsetWidth - 8;
      tooltip.style.left = Math.min(tx, maxX) + 'px';
      tooltip.style.top  = ty + 'px';
    });
    seg.addEventListener('mouseleave', () => {
      tooltip.style.display = 'none';
    });
  });
}

// ── ユーティリティ ──────────────────────────────────────────────

function fmtFull(iso) {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
}

function fmtDuration(sec) {
  if (sec < 60)   return `${sec}秒`;
  if (sec < 3600) return `${Math.floor(sec / 60)}分${sec % 60}秒`;
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${h}時間${m}分`;
}

function esc(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

loadMonth();
