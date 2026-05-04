// ── 共通ユーティリティ ──────────────────────────────────────────

function fmtDate(iso) {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  });
}

function fmtDateFull(iso) {
  return new Date(iso).toLocaleString('ja-JP', {
    month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
}

function fmtDuration(sec) {
  if (sec < 60) return `${sec}秒`;
  if (sec < 3600) return `${Math.floor(sec / 60)}分${sec % 60}秒`;
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${h}時間${m}分`;
}

function chartZoomPlugin() {
  return {
    pan: { enabled: true, mode: 'x' },
    zoom: {
      wheel: { enabled: true, modifierKey: 'ctrl' },
      pinch: { enabled: true },
      mode: 'x',
    },
  };
}

const visibleYRangePlugin = {
  id: 'visibleYRange',
  afterDataLimits(chart, args) {
    if (args.scale.axis !== 'y') return;
    const xScale = chart.scales.x;
    if (!xScale) return;
    const xMin = xScale.min, xMax = xScale.max;

    let yMin = Infinity, yMax = -Infinity;
    for (const ds of chart.data.datasets) {
      let carry = null;
      for (const p of ds.data) {
        const xv = p.x instanceof Date ? p.x.getTime() : +new Date(p.x);
        if (xv < xMin) { carry = p.y; continue; }
        if (xv > xMax) break;
        if (carry !== null) {
          yMin = Math.min(yMin, carry);
          yMax = Math.max(yMax, carry);
          carry = null;
        }
        yMin = Math.min(yMin, p.y);
        yMax = Math.max(yMax, p.y);
      }
      if (carry !== null) {
        yMin = Math.min(yMin, carry);
        yMax = Math.max(yMax, carry);
      }
    }
    if (!isFinite(yMin) || !isFinite(yMax)) return;

    const beginAtZero = chart.options.scales?.y?.beginAtZero;
    if (beginAtZero) yMin = Math.min(0, yMin);

    const lo = Math.floor(yMin);
    const hi = Math.ceil(yMax);
    args.scale.min = lo;
    args.scale.max = hi === lo ? hi + 1 : hi;
  }
};

function escHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
