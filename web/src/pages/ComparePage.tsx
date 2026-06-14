import { useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  Badge,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Typography,
} from "@mui/material";
import PlayerLink from "@/components/PlayerLink";
import InstanceInfo from "@/components/InstanceInfo";
import { Line } from "react-chartjs-2";
import type { Chart, ChartData, ChartOptions, Plugin } from "chart.js";
import { api } from "@/api/client";
import type { InstanceOut, SessionOut, TimelinePoint } from "@/api/schemas";
import {
  buildDiffPoints,
  buildPoints,
  buildSessionMap,
  detectViolations,
  type InstColor,
  type Point,
  type Violation,
} from "@/utils/violations";
import { chartZoomOptions, visibleYRangePlugin } from "@/utils/chart";
import { fmtDateFull, fmtDuration } from "@/utils/format";
import { useSortState } from "@/hooks/useSortState";

type VSortKey = "display_name" | "join_ts" | "instance" | "diff" | "duration_seconds";

const verticalLinePlugin: Plugin<"line"> = {
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

const COMMON_X = {
  type: "time" as const,
  time: { displayFormats: { minute: "HH:mm", hour: "MM/dd HH:mm" } },
};

export default function ComparePage() {
  const { id1: id1Str, id2: id2Str } = useParams<{ id1: string; id2: string }>();
  const id1 = Number(id1Str);
  const id2 = Number(id2Str);

  const compareRef = useRef<Chart<"line"> | null>(null);
  const diffRef = useRef<Chart<"line"> | null>(null);
  const [verticalX, setVerticalX] = useState<number | null>(null);
  const [grace, setGrace] = useState(15);
  const vSort = useSortState<VSortKey>("join_ts", "asc", "asc");

  const ids = [id1, id2] as const;
  const valid = ids.every((n) => Number.isFinite(n) && n > 0);

  const queries = useQuery({
    queryKey: ["compare", id1, id2],
    enabled: valid,
    queryFn: async () => {
      const results = await Promise.all([
        api.GET("/api/instances/{instance_id}", { params: { path: { instance_id: id1 } } }),
        api.GET("/api/instances/{instance_id}", { params: { path: { instance_id: id2 } } }),
        api.GET("/api/instances/{instance_id}/presence-timeline", {
          params: { path: { instance_id: id1 } },
        }),
        api.GET("/api/instances/{instance_id}/presence-timeline", {
          params: { path: { instance_id: id2 } },
        }),
        api.GET("/api/instances/{instance_id}/sessions", {
          params: { path: { instance_id: id1 } },
        }),
        api.GET("/api/instances/{instance_id}/sessions", {
          params: { path: { instance_id: id2 } },
        }),
      ]);
      if (results.some((r) => r.error)) throw new Error("failed to load compare data");
      return {
        inst1: results[0].data as InstanceOut,
        inst2: results[1].data as InstanceOut,
        tl1: (results[2].data ?? []) as TimelinePoint[],
        tl2: (results[3].data ?? []) as TimelinePoint[],
        sess1: (results[4].data ?? []) as SessionOut[],
        sess2: (results[5].data ?? []) as SessionOut[],
      };
    },
  });

  const derived = useMemo(() => {
    if (!queries.data) return null;
    const { inst1, inst2, tl1, tl2, sess1, sess2 } = queries.data;
    const pts1 = buildPoints(tl1, inst1.closed_at);
    const pts2 = buildPoints(tl2, inst2.closed_at);
    const sm1 = buildSessionMap(sess1);
    const sm2 = buildSessionMap(sess2);
    const graceSec = grace * 60;
    const violations = [
      ...detectViolations(tl1, pts2, sm1, "blue", graceSec),
      ...detectViolations(tl2, pts1, sm2, "red", graceSec),
    ].sort((a, b) => a.join_ts.getTime() - b.join_ts.getTime());
    return { pts1, pts2, tl1Len: tl1.length, tl2Len: tl2.length, violations, inst1, inst2 };
  }, [queries.data, grace]);

  if (!valid) {
    return (
      <Alert severity="error" className="m-3">
        URLパラメータ id1, id2 が必要です。
      </Alert>
    );
  }
  if (queries.error) {
    return (
      <Alert severity="error" className="m-3">
        データの読み込みに失敗しました
      </Alert>
    );
  }
  if (!derived) {
    return (
      <Typography color="text.secondary" className="p-3">
        読み込み中...
      </Typography>
    );
  }

  const { pts1, pts2, tl1Len, tl2Len, violations, inst1, inst2 } = derived;

  return (
    <Box className="h-full overflow-auto p-3 bg-neutral-50">
      <Stack spacing={2}>
        <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
          <InstanceCard inst={inst1} color="#0d6efd" />
          <InstanceCard inst={inst2} color="#dc3545" />
        </Stack>

        <Card>
          <CardContent>
            <Stack direction="row" className="mb-2" sx={{ justifyContent: "flex-end" }}>
              <Button
                size="small"
                variant="outlined"
                onClick={() => {
                  compareRef.current?.resetZoom();
                  diffRef.current?.resetZoom();
                }}
              >
                ズームリセット
              </Button>
            </Stack>
            <Box className="relative h-[420px]">
              <CompareChart
                pts1={pts1}
                pts2={pts2}
                rawLen1={tl1Len}
                rawLen2={tl2Len}
                verticalX={verticalX}
                onReady={(c) => (compareRef.current = c)}
                otherChartRef={diffRef}
              />
            </Box>
          </CardContent>
        </Card>

        <Card>
          <CardHeader
            className="border-b border-neutral-200 py-2!"
            title={
              <Typography variant="caption" color="text.secondary" className="font-semibold">
                人数差分（
                <Box component="span" color="primary.main">
                  青
                </Box>{" "}
                −{" "}
                <Box component="span" color="error.main">
                  赤
                </Box>
                ）
              </Typography>
            }
          />
          <CardContent>
            <Box className="relative h-[220px]">
              <DiffChart
                pts1={pts1}
                pts2={pts2}
                onReady={(c) => (diffRef.current = c)}
                otherChartRef={compareRef}
              />
            </Box>
          </CardContent>
        </Card>

        <Card>
          <CardHeader
            className="border-b border-neutral-200 py-2!"
            title={
              <Stack direction="row" sx={{ alignItems: "center", justifyContent: "space-between" }}>
                <Typography variant="caption" color="text.secondary" className="font-semibold">
                  ルール違反一覧
                </Typography>
                <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
                  <Typography variant="caption" component="label">
                    猶予
                  </Typography>
                  <Select
                    size="small"
                    value={grace}
                    onChange={(e) => setGrace(Number(e.target.value))}
                  >
                    {[5, 10, 15, 20, 25, 30].map((m) => (
                      <MenuItem key={m} value={m}>
                        {m}分
                      </MenuItem>
                    ))}
                  </Select>
                  <Badge badgeContent={violations.length} color="error" showZero />
                </Stack>
              </Stack>
            }
          />
          <ViolationsTable
            violations={violations}
            sort={{ by: vSort.sortBy, dir: vSort.order }}
            onSort={vSort.toggleSort}
            highlightedTs={verticalX}
            onPickTime={(t) => setVerticalX((prev) => (prev === t ? null : t))}
            id1={id1}
            id2={id2}
          />
        </Card>
      </Stack>
    </Box>
  );
}

function InstanceCard({ inst, color }: { inst: InstanceOut; color: string }) {
  return (
    <Card
      component={Link}
      to={`/instances/${inst.id}`}
      className="flex-1 no-underline text-inherit"
      sx={{
        borderColor: color,
        borderWidth: 1,
        borderStyle: "solid",
        borderLeftWidth: 4,
      }}
    >
      <CardContent className="py-2!">
        <InstanceInfo instance={inst} dense />
      </CardContent>
    </Card>
  );
}

interface CompareChartProps {
  pts1: Point[];
  pts2: Point[];
  rawLen1: number;
  rawLen2: number;
  verticalX: number | null;
  onReady: (chart: Chart<"line">) => void;
  otherChartRef: React.RefObject<Chart<"line"> | null>;
}

function CompareChart({
  pts1,
  pts2,
  rawLen1,
  rawLen2,
  verticalX,
  onReady,
  otherChartRef,
}: CompareChartProps) {
  const data: ChartData<"line", Point[]> = {
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
  };
  const options: ChartOptions<"line"> = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    scales: {
      x: COMMON_X,
      y: { beginAtZero: true, ticks: { stepSize: 1 }, title: { display: true, text: "人数" } },
    },
    plugins: {
      // @ts-expect-error custom plugin option
      verticalLine: { x: verticalX },
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
      zoom: {
        ...chartZoomOptions,
        zoom: {
          ...chartZoomOptions.zoom,
          onZoom: ({ chart }) => syncZoom(chart, otherChartRef.current),
        },
        pan: {
          ...chartZoomOptions.pan,
          onPan: ({ chart }) => syncZoom(chart, otherChartRef.current),
        },
      },
    },
  };
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
}

interface DiffChartProps {
  pts1: Point[];
  pts2: Point[];
  onReady: (chart: Chart<"line">) => void;
  otherChartRef: React.RefObject<Chart<"line"> | null>;
}

function DiffChart({ pts1, pts2, onReady, otherChartRef }: DiffChartProps) {
  const diffPts = useMemo(() => buildDiffPoints(pts1, pts2), [pts1, pts2]);
  // compareChart と同じX軸範囲にそろえる
  const allTimes = [...pts1, ...pts2].map((p) => p.x.getTime());
  const xMin = allTimes.length ? new Date(Math.min(...allTimes)) : undefined;
  const xMax = allTimes.length ? new Date(Math.max(...allTimes)) : undefined;
  // 正値 (青が多い) と負値 (赤が多い) を別データセットに分割
  const posPts = diffPts.map((p) => ({ x: p.x, y: Math.max(0, p.y) }));
  const negPts = diffPts.map((p) => ({ x: p.x, y: Math.min(0, p.y) }));
  const r = diffPts.length < 200 ? 2 : 0;

  const data: ChartData<"line", Point[]> = {
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

  const options: ChartOptions<"line"> = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    scales: {
      x: { ...COMMON_X, min: xMin?.getTime(), max: xMax?.getTime() },
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
      zoom: {
        ...chartZoomOptions,
        zoom: {
          ...chartZoomOptions.zoom,
          onZoom: ({ chart }) => syncZoom(chart, otherChartRef.current),
        },
        pan: {
          ...chartZoomOptions.pan,
          onPan: ({ chart }) => syncZoom(chart, otherChartRef.current),
        },
      },
    },
  };

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
}

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

interface ViolationsTableProps {
  violations: Violation[];
  sort: { by: VSortKey; dir: "asc" | "desc" };
  onSort: (key: VSortKey) => void;
  highlightedTs: number | null;
  onPickTime: (t: number) => void;
  id1: number;
  id2: number;
}

function ViolationsTable({
  violations,
  sort,
  onSort,
  highlightedTs,
  onPickTime,
  id1,
  id2,
}: ViolationsTableProps) {
  const sorted = useMemo(() => {
    return [...violations].sort((a, b) => {
      let va: number | string;
      let vb: number | string;
      if (sort.by === "join_ts") {
        va = a.join_ts.getTime();
        vb = b.join_ts.getTime();
      } else if (sort.by === "duration_seconds") {
        va = a.duration_seconds ?? -1;
        vb = b.duration_seconds ?? -1;
      } else {
        va = a[sort.by] as string;
        vb = b[sort.by] as string;
      }
      if (va < vb) return sort.dir === "asc" ? -1 : 1;
      if (va > vb) return sort.dir === "asc" ? 1 : -1;
      return 0;
    });
  }, [violations, sort]);

  if (violations.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" className="text-center py-3">
        違反なし
      </Typography>
    );
  }

  const COLS: { key: VSortKey; label: string }[] = [
    { key: "display_name", label: "ユーザー名" },
    { key: "join_ts", label: "違反時刻" },
    { key: "instance", label: "参加先" },
    { key: "diff", label: "人数差" },
    { key: "duration_seconds", label: "滞在時間" },
  ];

  return (
    <TableContainer component={Paper} variant="outlined" square>
      <Table size="small">
        <TableHead>
          <TableRow>
            {COLS.map((c) => (
              <TableCell key={c.key} sortDirection={sort.by === c.key ? sort.dir : false}>
                <TableSortLabel
                  active={sort.by === c.key}
                  direction={sort.by === c.key ? sort.dir : "asc"}
                  onClick={() => onSort(c.key)}
                >
                  {c.label}
                </TableSortLabel>
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {sorted.map((v, i) => {
            const ts = v.join_ts.getTime();
            const color: InstColor = v.instance;
            return (
              <TableRow
                key={i}
                hover
                selected={highlightedTs === ts}
                onClick={() => onPickTime(ts)}
                sx={{ cursor: "pointer" }}
              >
                <TableCell>
                  <PlayerLink
                    userId={v.user_id}
                    displayName={v.display_name}
                    instanceId={v.instance === "blue" ? id1 : id2}
                    stopPropagation
                  />
                </TableCell>
                <TableCell>{fmtDateFull(v.join_ts)}</TableCell>
                <TableCell>
                  <Chip
                    size="small"
                    label={color === "blue" ? "青" : "赤"}
                    sx={{
                      backgroundColor: color === "blue" ? "#0d6efd" : "#dc3545",
                      color: "#fff",
                    }}
                  />
                </TableCell>
                <TableCell>+{v.diff}</TableCell>
                <TableCell>
                  {v.duration_seconds != null ? fmtDuration(v.duration_seconds) : "—"}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
