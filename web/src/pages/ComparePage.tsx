import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  Badge,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  MenuItem,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import InstanceInfo from "@/components/InstanceInfo";
import CompareChart from "@/components/CompareChart";
import DiffChart from "@/components/DiffChart";
import ViolationsTable, { type VSortKey } from "@/components/ViolationsTable";
import type { Chart } from "chart.js";
import { api } from "@/api/client";
import type { InstanceOut, SessionOut, TimelinePoint } from "@/api/schemas";
import { buildPoints, buildSessionMap, detectViolations } from "@/utils/violations";
import { useSortState } from "@/hooks/useSortState";

export default function ComparePage() {
  const [searchParams] = useSearchParams();
  const ids = useMemo(() => {
    const parsed = (searchParams.get("ids") ?? "")
      .split(",")
      .map((s) => Number(s.trim()))
      .filter((n) => Number.isFinite(n) && n > 0);
    return [...new Set(parsed)].sort((a, b) => a - b);
  }, [searchParams]);
  const valid = ids.length === 2;
  const [id1, id2] = ids;

  const compareRef = useRef<Chart<"line"> | null>(null);
  const diffRef = useRef<Chart<"line"> | null>(null);
  const [verticalX, setVerticalX] = useState<number | null>(null);
  const [grace, setGrace] = useState(15);
  const vSort = useSortState<VSortKey>("join_ts", "asc", "asc");

  const handleCompareReady = useCallback((c: Chart<"line">) => {
    compareRef.current = c;
  }, []);
  const handleDiffReady = useCallback((c: Chart<"line">) => {
    diffRef.current = c;
  }, []);

  useEffect(() => {
    const ch = compareRef.current;
    if (!ch) return;
    const p = ch.options.plugins as { verticalLine?: { x: number | null } };
    if (p.verticalLine) p.verticalLine.x = verticalX;
    ch.draw();
  }, [verticalX]);

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
        比較するインスタンスID 2件を ?ids=... で指定してください。
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
                onReady={handleCompareReady}
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
                onReady={handleDiffReady}
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
