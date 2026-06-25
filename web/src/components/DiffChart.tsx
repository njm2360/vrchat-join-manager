import { memo, useMemo } from "react";
import { Line } from "react-chartjs-2";
import type { Chart, ChartData, ChartOptions } from "chart.js";
import { buildDiffPoints, type Point } from "@/utils/violations";
import { COMMON_X, syncedZoomOptions, visibleYRangePlugin } from "@/utils/chart";

interface DiffChartProps {
  pts1: Point[];
  pts2: Point[];
  onReady: (chart: Chart<"line">) => void;
  otherChartRef: React.RefObject<Chart<"line"> | null>;
}

const DiffChart = memo(function DiffChart({ pts1, pts2, onReady, otherChartRef }: DiffChartProps) {
  const diffPts = useMemo(() => buildDiffPoints(pts1, pts2), [pts1, pts2]);
  const [xMin, xMax] = useMemo(() => {
    let min = Infinity;
    let max = -Infinity;
    for (const p of pts1) {
      const t = p.x.getTime();
      if (t < min) min = t;
      if (t > max) max = t;
    }
    for (const p of pts2) {
      const t = p.x.getTime();
      if (t < min) min = t;
      if (t > max) max = t;
    }
    return Number.isFinite(min) ? [min, max] : [undefined, undefined];
  }, [pts1, pts2]);

  const data = useMemo<ChartData<"line", Point[]>>(() => {
    const posPts = diffPts.map((p) => ({ x: p.x, y: Math.max(0, p.y) }));
    const negPts = diffPts.map((p) => ({ x: p.x, y: Math.min(0, p.y) }));
    const r = diffPts.length < 200 ? 2 : 0;
    return {
      datasets: [
        {
          data: posPts,
          borderColor: "rgb(13, 110, 253)",
          backgroundColor: "rgba(13, 110, 253, 0.18)",
          stepped: true,
          fill: "origin",
          pointRadius: r,
          borderWidth: 1.5,
        },
        {
          data: negPts,
          borderColor: "rgb(220, 53, 69)",
          backgroundColor: "rgba(220, 53, 69, 0.18)",
          stepped: true,
          fill: "origin",
          pointRadius: r,
          borderWidth: 1.5,
        },
      ],
    };
  }, [diffPts]);

  const options = useMemo<ChartOptions<"line">>(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      scales: {
        x: { ...COMMON_X, min: xMin, max: xMax },
        y: { ticks: { stepSize: 1 }, title: { display: true, text: "差分 (人)" } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          mode: "index",
          intersect: false,
          filter: (item) => item.datasetIndex === 0,
          callbacks: {
            title: (items) => new Date(items[0].parsed.x ?? 0).toLocaleString("ja-JP"),
            label: (item) => {
              const v = diffPts[item.dataIndex]?.y ?? 0;
              if (v > 0) return ` 青が ${v} 人多い`;
              if (v < 0) return ` 赤が ${-v} 人多い`;
              return " 同数";
            },
          },
        },
        zoom: syncedZoomOptions(otherChartRef),
      },
    }),
    [xMin, xMax, diffPts, otherChartRef],
  );

  return (
    <Line
      ref={(c) => {
        if (c) onReady(c as unknown as Chart<"line">);
      }}
      data={data}
      options={options}
      plugins={[visibleYRangePlugin]}
    />
  );
});

export default DiffChart;
