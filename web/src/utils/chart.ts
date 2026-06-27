import type { RefObject } from "react";
import type { Chart, Plugin } from "chart.js";

export const chartZoomOptions = {
  pan: { enabled: true, mode: "x" as const },
  zoom: {
    wheel: { enabled: true, modifierKey: "ctrl" as const },
    pinch: { enabled: true },
    mode: "x" as const,
  },
};

export const COMMON_X = {
  type: "time" as const,
  time: { displayFormats: { minute: "HH:mm", hour: "MM/dd HH:mm" } },
};

export const verticalLinePlugin: Plugin<"line"> = {
  id: "verticalLine",
  afterDraw(chart) {
    const x = (chart.options.plugins as { verticalLine?: { x: number | null } })?.verticalLine?.x;
    if (x == null) return;
    const { ctx, scales } = chart;
    const xPos = scales.x.getPixelForValue(x);
    if (xPos < scales.x.left || xPos > scales.x.right) return;
    ctx.save();
    ctx.beginPath();
    ctx.moveTo(xPos, scales.y.top);
    ctx.lineTo(xPos, scales.y.bottom);
    ctx.strokeStyle = "rgba(255, 165, 0, 0.9)";
    ctx.lineWidth = 2;
    ctx.setLineDash([5, 3]);
    ctx.stroke();
    ctx.restore();
  },
};

let _syncingZoom = false;
function syncZoom(src: Chart, dst: Chart<"line"> | null) {
  if (_syncingZoom || !src || !dst) return;
  _syncingZoom = true;
  try {
    const sx = src.scales.x;
    dst.zoomScale("x", { min: sx.min, max: sx.max }, "none");
  } finally {
    _syncingZoom = false;
  }
}

export function syncedZoomOptions(otherChartRef: RefObject<Chart<"line"> | null>) {
  return {
    ...chartZoomOptions,
    zoom: {
      ...chartZoomOptions.zoom,
      onZoom: ({ chart }: { chart: Chart }) => syncZoom(chart, otherChartRef.current),
    },
    pan: {
      ...chartZoomOptions.pan,
      onPan: ({ chart }: { chart: Chart }) => syncZoom(chart, otherChartRef.current),
    },
  };
}

const toMs = (v: unknown): number | null => {
  if (v == null) return null;
  if (typeof v === "number") return v;
  if (v instanceof Date) return v.getTime();
  const t = +new Date(v as string);
  return Number.isNaN(t) ? null : t;
};

// 表示中のX軸範囲に応じてY軸を自動スケール
export const visibleYRangePlugin: Plugin<"line"> = {
  id: "visibleYRange",
  afterDataLimits(chart: Chart, args) {
    if (args.scale.axis !== "y") return;
    const xScale = chart.scales.x;
    if (!xScale) return;
    const xOpts = xScale.options as { min?: unknown; max?: unknown };
    const xMin = toMs(xOpts.min) ?? -Infinity;
    const xMax = toMs(xOpts.max) ?? Infinity;

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
