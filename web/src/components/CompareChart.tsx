import { memo, useMemo } from "react";
import { Line } from "react-chartjs-2";
import type { Chart, ChartData, ChartOptions } from "chart.js";
import type { Point } from "@/utils/violations";
import {
  COMMON_X,
  syncedZoomOptions,
  verticalLinePlugin,
  visibleYRangePlugin,
} from "@/utils/chart";

interface CompareChartProps {
  pts1: Point[];
  pts2: Point[];
  rawLen1: number;
  rawLen2: number;
  onReady: (chart: Chart<"line">) => void;
  otherChartRef: React.RefObject<Chart<"line"> | null>;
}

const CompareChart = memo(function CompareChart({
  pts1,
  pts2,
  rawLen1,
  rawLen2,
  onReady,
  otherChartRef,
}: CompareChartProps) {
  const data = useMemo<ChartData<"line", Point[]>>(
    () => ({
      datasets: [
        {
          label: "青",
          data: pts1,
          borderColor: "rgb(13, 110, 253)",
          backgroundColor: "rgba(13, 110, 253, 0.08)",
          stepped: true,
          fill: true,
          pointRadius: rawLen1 < 200 ? 3 : 0,
          borderWidth: 2,
        },
        {
          label: "赤",
          data: pts2,
          borderColor: "rgb(220, 53, 69)",
          backgroundColor: "rgba(220, 53, 69, 0.08)",
          stepped: true,
          fill: true,
          pointRadius: rawLen2 < 200 ? 3 : 0,
          borderWidth: 2,
        },
      ],
    }),
    [pts1, pts2, rawLen1, rawLen2],
  );
  const options = useMemo<ChartOptions<"line">>(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      scales: {
        x: COMMON_X,
        y: { beginAtZero: true, ticks: { stepSize: 1 }, title: { display: true, text: "人数" } },
      },
      plugins: {
        verticalLine: { x: null },
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: (items) => new Date(items[0].parsed.x ?? 0).toLocaleString("ja-JP"),
            label: (item) => ` ${item.datasetIndex === 0 ? "青" : "赤"}: ${item.parsed.y} 人`,
            afterLabel: (item) => {
              const raw = item.dataset.data[item.dataIndex] as unknown as Point;
              return raw?.displayName ? ` ${raw.displayName}` : "";
            },
          },
        },
        zoom: syncedZoomOptions(otherChartRef),
      },
    }),
    [otherChartRef],
  );
  return (
    <Line
      ref={(c) => {
        if (c) onReady(c as unknown as Chart<"line">);
      }}
      data={data}
      options={options}
      plugins={[verticalLinePlugin, visibleYRangePlugin]}
    />
  );
});

export default CompareChart;
