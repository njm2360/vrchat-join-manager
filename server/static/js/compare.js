const params = new URLSearchParams(location.search);
const id1 = params.get('id1');
const id2 = params.get('id2');

if (!id1 || !id2) {
  const el = document.getElementById('error-msg');
  el.textContent = 'URLパラメータ id1, id2 が必要です。';
  el.classList.remove('d-none');
} else {
  loadCompare();
}

let allViolations = [];
let vSortCol = 'join_ts';
let vSortDir = 'asc';

async function loadCompare() {
  try {
    const [inst1, inst2, tl1, tl2, sess1, sess2] = await Promise.all([
      fetch(`/api/instances/${id1}`).then(r => r.json()),
      fetch(`/api/instances/${id2}`).then(r => r.json()),
      fetch(`/api/instances/${id1}/presence-timeline`).then(r => r.json()),
      fetch(`/api/instances/${id2}/presence-timeline`).then(r => r.json()),
      fetch(`/api/instances/${id1}/sessions`).then(r => r.json()),
      fetch(`/api/instances/${id2}/sessions`).then(r => r.json()),
    ]);

    renderInfo('inst1', inst1);
    renderInfo('inst2', inst2);

    const pts1 = buildPoints(tl1, inst1);
    const pts2 = buildPoints(tl2, inst2);
    renderCompareChart(pts1, tl1.length, pts2, tl2.length);
    renderDiffChart(pts1, pts2);

    const sessMap1 = buildSessionMap(sess1);
    const sessMap2 = buildSessionMap(sess2);
    allViolations = [
      ...detectViolations(tl1, pts2, sessMap1, 'blue'),
      ...detectViolations(tl2, pts1, sessMap2, 'red'),
    ].sort((a, b) => a.join_ts - b.join_ts);

    initViolationsTable();
    renderViolations();
  } catch (e) {
    const el = document.getElementById('error-msg');
    el.textContent = 'データの読み込みに失敗しました: ' + e.message;
    el.classList.remove('d-none');
  }
}

function renderInfo(prefix, inst) {
  document.getElementById(prefix + '-world').textContent = inst.world_id ?? '—';
  document.getElementById(prefix + '-location').textContent = inst.location_id ?? '';
  const openedAt = fmtDate(inst.opened_at);
  const closedAt = inst.closed_at ? fmtDate(inst.closed_at) : '進行中';
  document.getElementById(prefix + '-range').innerHTML = inst.closed_at
    ? `<span class="badge bg-secondary">${openedAt} 〜 ${closedAt}</span>`
    : `<span class="badge bg-success">${openedAt} 〜 ${closedAt}</span>`;
  document.getElementById(prefix + '-link').href = `/?instance=${inst.id}`;
}

function buildPoints(data, inst) {
  const pts = data.map(d => ({ x: new Date(d.timestamp), y: d.count }));
  if (!inst.closed_at && pts.length > 0) {
    pts.push({ x: new Date(), y: pts[pts.length - 1].y });
  }
  return pts;
}

// ステップ補間: 時刻 t における pts の値を返す
function stepValue(pts, t) {
  let val = 0;
  for (const p of pts) {
    if (p.x.getTime() <= t) val = p.y;
    else break;
  }
  return val;
}

function buildDiffPoints(pts1, pts2) {
  const times = [...new Set([
    ...pts1.map(p => p.x.getTime()),
    ...pts2.map(p => p.x.getTime()),
  ])].sort((a, b) => a - b);

  return times.map(t => ({
    x: new Date(t),
    y: stepValue(pts1, t) - stepValue(pts2, t),
  }));
}

const COMMON_X_OPTIONS = {
  type: 'time',
  time: { displayFormats: { minute: 'HH:mm', hour: 'MM/dd HH:mm' } },
};

function renderCompareChart(pts1, rawLen1, pts2, rawLen2) {
  const ctx = document.getElementById('compare-chart').getContext('2d');
  new Chart(ctx, {
    type: 'line',
    data: {
      datasets: [
        {
          label: '青',
          data: pts1,
          borderColor: 'rgb(13, 110, 253)',
          backgroundColor: 'rgba(13, 110, 253, 0.08)',
          stepped: true,
          fill: true,
          pointRadius: rawLen1 < 200 ? 3 : 0,
          borderWidth: 2,
        },
        {
          label: '赤',
          data: pts2,
          borderColor: 'rgb(220, 53, 69)',
          backgroundColor: 'rgba(220, 53, 69, 0.08)',
          stepped: true,
          fill: true,
          pointRadius: rawLen2 < 200 ? 3 : 0,
          borderWidth: 2,
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: COMMON_X_OPTIONS,
        y: {
          beginAtZero: true,
          ticks: { stepSize: 1 },
          title: { display: true, text: '人数' }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: items => new Date(items[0].parsed.x).toLocaleString('ja-JP'),
            label: items => {
              const color = items.datasetIndex === 0 ? '青' : '赤';
              return ` ${color}: ${items.parsed.y} 人`;
            }
          }
        }
      }
    }
  });
}

// セッションマップ: user_id -> [{join_ms, duration_seconds}]
function buildSessionMap(sessions) {
  const map = new Map();
  for (const s of sessions) {
    const ms = new Date(s.join_ts).getTime();
    if (!map.has(s.user_id)) map.set(s.user_id, []);
    map.get(s.user_id).push({ join_ms: ms, duration_seconds: s.duration_seconds });
  }
  return map;
}

function lookupDuration(sessionsMap, user_id, join_ms) {
  const arr = sessionsMap.get(user_id);
  if (!arr) return null;
  return arr.find(s => s.join_ms === join_ms)?.duration_seconds ?? null;
}

// instColor ('blue'|'red') のインスタンスへの違反Joinを検出
// 違反 = 自インスタンスの方が相手より人が多い状態でJoinした
function detectViolations(tl, otherPts, sessionsMap, instColor) {
  const violations = [];
  for (let i = 1; i < tl.length; i++) {
    const pt = tl[i];
    if (!pt.user_id) continue;
    const countBefore = tl[i - 1].count; // このイベント直前の自インスタンス人数
    if (pt.count <= countBefore) continue; // Joinでない (Leave)

    const t = new Date(pt.timestamp).getTime();
    const otherCount = stepValue(otherPts, t);
    const diff = countBefore - otherCount;
    if (diff <= 0) continue; // 相手の方が多い or 同数 → 違反なし

    violations.push({
      display_name: pt.display_name,
      join_ts: new Date(pt.timestamp),
      instance: instColor,
      diff,
      duration_seconds: lookupDuration(sessionsMap, pt.user_id, t),
    });
  }
  return violations;
}

function initViolationsTable() {
  document.querySelectorAll('#violations-table th[data-col]').forEach(th => {
    th.addEventListener('click', () => {
      const col = th.dataset.col;
      if (vSortCol === col) {
        vSortDir = vSortDir === 'asc' ? 'desc' : 'asc';
      } else {
        vSortCol = col;
        vSortDir = 'asc';
      }
      renderViolations();
    });
  });
}

function renderViolations() {
  const tbody = document.getElementById('violations-tbody');
  const empty = document.getElementById('violations-empty');
  const countBadge = document.getElementById('violation-count');
  countBadge.textContent = allViolations.length;

  // ソートアイコン更新
  document.querySelectorAll('#violations-table th[data-col]').forEach(th => {
    const icon = th.querySelector('.v-sort-icon');
    if (th.dataset.col === vSortCol) {
      icon.textContent = vSortDir === 'asc' ? ' ↑' : ' ↓';
    } else {
      icon.textContent = ' ⇅';
    }
  });

  if (allViolations.length === 0) {
    tbody.innerHTML = '';
    empty.classList.remove('d-none');
    return;
  }
  empty.classList.add('d-none');

  const sorted = [...allViolations].sort((a, b) => {
    let va = a[vSortCol];
    let vb = b[vSortCol];
    if (vSortCol === 'join_ts') { va = va.getTime(); vb = vb.getTime(); }
    if (vSortCol === 'duration_seconds') { va = va ?? -1; vb = vb ?? -1; }
    if (va < vb) return vSortDir === 'asc' ? -1 : 1;
    if (va > vb) return vSortDir === 'asc' ? 1 : -1;
    return 0;
  });

  tbody.innerHTML = sorted.map(v => {
    const color = v.instance === 'blue' ? '#0d6efd' : '#dc3545';
    const label = v.instance === 'blue' ? '青' : '赤';
    const dur = v.duration_seconds != null ? fmtDuration(v.duration_seconds) : '—';
    return `<tr>
      <td>${escHtml(v.display_name)}</td>
      <td class="text-nowrap">${fmtDate(v.join_ts.toISOString())}</td>
      <td><span class="badge" style="background:${color}">${label}</span></td>
      <td>+${v.diff}</td>
      <td class="text-nowrap">${dur}</td>
    </tr>`;
  }).join('');
}

function renderDiffChart(pts1, pts2) {
  const ctx = document.getElementById('diff-chart').getContext('2d');
  const diffPts = buildDiffPoints(pts1, pts2);
  // 正値（青が多い）と負値（赤が多い）を別データセットに分割
  const posPts = diffPts.map(p => ({ x: p.x, y: Math.max(0, p.y) }));
  const negPts = diffPts.map(p => ({ x: p.x, y: Math.min(0, p.y) }));
  const r = diffPts.length < 200 ? 2 : 0;

  new Chart(ctx, {
    type: 'line',
    data: {
      datasets: [
        {
          data: posPts,
          borderColor: 'rgb(13, 110, 253)',
          backgroundColor: 'rgba(13, 110, 253, 0.18)',
          stepped: true,
          fill: 'origin',
          pointRadius: r,
          borderWidth: 1.5,
        },
        {
          data: negPts,
          borderColor: 'rgb(220, 53, 69)',
          backgroundColor: 'rgba(220, 53, 69, 0.18)',
          stepped: true,
          fill: 'origin',
          pointRadius: r,
          borderWidth: 1.5,
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: COMMON_X_OPTIONS,
        y: {
          ticks: { stepSize: 1 },
          title: { display: true, text: '差分 (人)' }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          mode: 'index',
          intersect: false,
          filter: item => item.datasetIndex === 0,
          callbacks: {
            title: items => new Date(items[0].parsed.x).toLocaleString('ja-JP'),
            label: items => {
              const v = diffPts[items.dataIndex].y;
              if (v > 0) return ` 青が ${v} 人多い`;
              if (v < 0) return ` 赤が ${-v} 人多い`;
              return ' 同数';
            }
          }
        }
      }
    }
  });
}
