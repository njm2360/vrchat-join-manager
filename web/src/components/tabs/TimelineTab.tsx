import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Box, Button, Stack } from "@mui/material";
import { Line } from "react-chartjs-2";
import type { ChartOptions, ChartData } from "chart.js";
import type { Chart } from "chart.js";
import { useTimeline } from "@/api/queries";
import type { InstanceOut } from "@/api/schemas";
import { chartZoomOptions, visibleYRangePlugin } from "@/utils/chart";
import DateRangeFilter from "@/components/DateRangeFilter";

interface Props {
  instanceId: number;
  instance: InstanceOut | null;
  onCompare: () => void;
}

type Pt = { x: Date; y: number; displayName?: string | null };

export default function TimelineTab({ instanceId, instance, onCompare }: Props) {
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});
  const [nowMs] = useState(() => Date.now());
  const chartRef = useRef<Chart<"line"> | null>(null);

  const { data: timeline = [] } = useTimeline(instanceId, applied);

  useEffect(() => {
    chartRef.current?.resetZoom();
  }, [instanceId]);

  const isOngoing = instance && !instance.closed_at;
  const points: Pt[] = useMemo(() => {
    const pts: Pt[] = timeline.map((d) => ({
      x: new Date(d.timestamp),
      y: d.count,
      displayName: d.display_name,
    }));
    if (isOngoing && pts.length > 0) {
      pts.push({ x: new Date(nowMs), y: pts[pts.length - 1].y, displayName: null });
    }
    return pts;
  }, [timeline, isOngoing, nowMs]);

  const data: ChartData<"line", Pt[]> = {
    datasets: [
      {
        label: "人数",
        data: points,
        borderColor: "rgb(13, 110, 253)",
        backgroundColor: "rgba(13, 110, 253, 0.08)",
        stepped: true,
        fill: true,
        pointRadius: points.length < 200 ? 3 : 0,
        borderWidth: 2,
      },
    ],
  };

  const options: ChartOptions<"line"> = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    scales: {
      x: {
        type: "time",
        time: { displayFormats: { minute: "HH:mm", hour: "MM/dd HH:mm" } },
        max: isOngoing ? nowMs : undefined,
      },
      y: {
        beginAtZero: true,
        ticks: { stepSize: 1 },
        title: { display: true, text: "人数" },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        callbacks: {
          title: (items) => new Date(items[0].parsed.x ?? 0).toLocaleString("ja-JP"),
          label: (item) => ` ${item.parsed.y} 人`,
          afterLabel: (item) => {
            const raw = item.dataset.data[item.dataIndex] as unknown as Pt;
            return raw?.displayName ? ` ${raw.displayName}` : "";
          },
        },
      },
      zoom: chartZoomOptions,
    },
  };

  const setChartRef = useCallback((c: unknown) => {
    chartRef.current = (c as Chart<"line"> | null) ?? null;
  }, []);

  return (
    <Stack spacing={2}>
      <DateRangeFilter onApply={setApplied}>
        <Button variant="outlined" size="small" onClick={() => chartRef.current?.resetZoom()}>
          ズームリセット
        </Button>
        <Box className="ml-auto">
          <Button variant="outlined" size="small" onClick={onCompare}>
            他のインスタンスと比較
          </Button>
        </Box>
      </DateRangeFilter>
      <Box className="relative h-[420px]">
        <Line ref={setChartRef} data={data} options={options} plugins={[visibleYRangePlugin]} />
      </Box>
    </Stack>
  );
}
