import type { Chart, Plugin } from "chart.js";

export const chartZoomOptions = {
  pan: { enabled: true, mode: "x" as const },
  zoom: {
    wheel: { enabled: true, modifierKey: "ctrl" as const },
    pinch: { enabled: true },
    mode: "x" as const,
  },
};

// 表示中のX軸範囲に応じてY軸を自動スケール
export const visibleYRangePlugin: Plugin<"line"> = {
  id: "visibleYRange",
  afterDataLimits(chart: Chart, args) {
    if (args.scale.axis !== "y") return;
    const xScale = chart.scales.x;
    if (!xScale) return;
    const xMin = xScale.min;
    const xMax = xScale.max;

    let yMin = Infinity;
    let yMax = -Infinity;
    for (const ds of chart.data.datasets) {
      let carry: number | null = null;
      for (const raw of ds.data as { x: Date | number; y: number }[]) {
        const xv = raw.x instanceof Date ? raw.x.getTime() : +new Date(raw.x);
        if (xv < xMin) {
          carry = raw.y;
          continue;
        }
        if (xv > xMax) break;
        if (carry !== null) {
          yMin = Math.min(yMin, carry);
          yMax = Math.max(yMax, carry);
          carry = null;
        }
        yMin = Math.min(yMin, raw.y);
        yMax = Math.max(yMax, raw.y);
      }
      if (carry !== null) {
        yMin = Math.min(yMin, carry);
        yMax = Math.max(yMax, carry);
      }
    }
    if (!isFinite(yMin) || !isFinite(yMax)) return;

    const beginAtZero =
      chart.options.scales?.y && "beginAtZero" in chart.options.scales.y
        ? (chart.options.scales.y as { beginAtZero?: boolean }).beginAtZero
        : undefined;
    if (beginAtZero) yMin = Math.min(0, yMin);

    const lo = Math.floor(yMin);
    const hi = Math.ceil(yMax);
    args.scale.min = lo;
    args.scale.max = hi === lo ? hi + 1 : hi;
  },
};
