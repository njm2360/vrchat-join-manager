import { useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams, Link } from "react-router-dom";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import ChevronLeftIcon from "@mui/icons-material/ChevronLeft";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import { usePlayerDetail, usePlayerSessions } from "@/api/queries";
import type { PlayerSessionOut } from "@/api/schemas";
import { fmtDateFull, fmtDuration } from "@/utils/format";

const DOW = ["日", "月", "火", "水", "木", "金", "土"];

export default function PlayerPage() {
  const { userId = "" } = useParams<{ userId: string }>();
  const [params, setParams] = useSearchParams();
  const worldId = params.get("world_id") || "";
  const { data: player } = usePlayerDetail(userId);
  const displayName = player?.display_name || userId;

  const now = useMemo(() => new Date(), []);
  const yearParam = Number(params.get("year"));
  const monthParam = Number(params.get("month"));
  const year = Number.isInteger(yearParam) && yearParam >= 1970 ? yearParam : now.getFullYear();
  const month =
    Number.isInteger(monthParam) && monthParam >= 1 && monthParam <= 12
      ? monthParam - 1
      : now.getMonth();

  const setYM = (next: { year: number; month: number }) => {
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set("year", String(next.year));
        p.set("month", String(next.month + 1));
        return p;
      },
      { replace: true },
    );
  };

  useEffect(() => {
    if (params.get("year") && params.get("month")) return;
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (!p.get("year")) p.set("year", String(year));
        if (!p.get("month")) p.set("month", String(month + 1));
        return p;
      },
      { replace: true },
    );
  }, []);

  // 前月末日から翌月1日まで取得 (月をまたぐセッションも拾う)
  const start = new Date(year, month, 0).toISOString();
  const end = new Date(year, month + 1, 1).toISOString();

  const { data: sessions = [] } = usePlayerSessions(userId, {
    start,
    end,
    order: "asc",
    limit: 2000,
    world_id: worldId || undefined,
  });

  const prev = () =>
    setYM(month === 0 ? { year: year - 1, month: 11 } : { year, month: month - 1 });
  const next = () =>
    setYM(month === 11 ? { year: year + 1, month: 0 } : { year, month: month + 1 });

  return (
    <Box className="h-full overflow-auto p-3">
      <title>{`${displayName} — セッション履歴`}</title>
      <Card className="max-w-[960px] mx-auto">
        <CardHeader
          title={
            <Stack direction="row" spacing={1.5} sx={{ alignItems: "center", flexWrap: "wrap" }}>
              <Typography
                component={Link}
                to="/"
                variant="subtitle1"
                className="font-medium no-underline text-inherit hover:underline"
              >
                {displayName}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                のセッション履歴
              </Typography>
              {worldId && (
                <Tooltip title={worldId} arrow>
                  <Chip size="small" label={worldId} className="max-w-[320px]" />
                </Tooltip>
              )}
              <Box className="flex-1" />
              <IconButton size="small" onClick={prev}>
                <ChevronLeftIcon />
              </IconButton>
              <Typography variant="subtitle2" className="min-w-[6em] text-center font-semibold">
                {year}年{String(month + 1).padStart(2, "0")}月
              </Typography>
              <IconButton size="small" onClick={next}>
                <ChevronRightIcon />
              </IconButton>
            </Stack>
          }
        />
        <CardContent>
          <HourScale />
          <MonthCalendar year={year} month={month} sessions={sessions} />
        </CardContent>
      </Card>
    </Box>
  );
}

function HourScale() {
  return (
    <Box className="flex items-center h-5 mb-1">
      <Box className="basis-[84px] shrink-0" />
      <Box className="flex-1 relative text-[10px] text-neutral-500">
        <span className="absolute left-0">0:00</span>
        <span className="absolute left-1/4 -translate-x-1/2">6:00</span>
        <span className="absolute left-1/2 -translate-x-1/2">12:00</span>
        <span className="absolute left-3/4 -translate-x-1/2">18:00</span>
        <span className="absolute right-0 translate-x-0">24:00</span>
      </Box>
    </Box>
  );
}

interface MonthProps {
  year: number;
  month: number;
  sessions: PlayerSessionOut[];
}

function MonthCalendar({ year, month, sessions }: MonthProps) {
  const days = new Date(year, month + 1, 0).getDate();
  const [nowMs] = useState(() => Date.now());
  const DAY_MS = 86_400_000;
  const [hoveredKey, setHoveredKey] = useState<string | null>(null);

  return (
    <Box className="text-[13px]">
      {Array.from({ length: days }, (_, i) => i + 1).map((d) => {
        const dayStart = new Date(year, month, d).getTime();
        const dayEnd = dayStart + DAY_MS;
        const dow = new Date(year, month, d).getDay();
        const color = dow === 0 ? "text-red-600" : dow === 6 ? "text-blue-600" : "text-inherit";

        const segs = sessions
          .map((s) => {
            const sStart = new Date(s.join_ts).getTime();
            const sEnd = s.leave_ts ? new Date(s.leave_ts).getTime() : nowMs;
            if (sStart >= dayEnd || sEnd <= dayStart) return null;
            const segStart = Math.max(sStart, dayStart);
            const segEnd = Math.min(sEnd, dayEnd);
            return {
              s,
              key: s.join_ts,
              leftPct: ((segStart - dayStart) / DAY_MS) * 100,
              widthPct: Math.max(0.2, ((segEnd - segStart) / DAY_MS) * 100),
            };
          })
          .filter((x): x is NonNullable<typeof x> => x !== null);

        return (
          <Box key={d} className="flex items-center h-[30px] border-b border-neutral-100">
            <Box
              className={`basis-[84px] shrink-0 text-right pr-2 text-xs tabular-nums select-none ${color}`}
            >
              {String(month + 1).padStart(2, "0")}/{String(d).padStart(2, "0")} ({DOW[dow]})
            </Box>
            <Box
              className="flex-1 relative h-[18px] rounded-sm"
              sx={{
                background:
                  "linear-gradient(#c8cdd2,#c8cdd2) no-repeat 25%/1px 100%, " +
                  "linear-gradient(#c8cdd2,#c8cdd2) no-repeat 50%/1px 100%, " +
                  "linear-gradient(#c8cdd2,#c8cdd2) no-repeat 75%/1px 100%, #dee2e6",
              }}
            >
              {segs.map(({ s, key, leftPct, widthPct }, idx) => {
                const active = hoveredKey === key;
                return (
                  <Tooltip
                    key={idx}
                    arrow
                    placement="top"
                    title={
                      <Stack spacing={0.25} className="text-[12px] leading-snug">
                        <Box>
                          <Box component="span" className="opacity-70 mr-1">
                            入室
                          </Box>
                          <Box component="span" className="tabular-nums">
                            {fmtDateFull(s.join_ts)}
                          </Box>
                        </Box>
                        <Box>
                          <Box component="span" className="opacity-70 mr-1">
                            退室
                          </Box>
                          <Box component="span" className="tabular-nums">
                            {s.leave_ts ? fmtDateFull(s.leave_ts) : "在室中"}
                            {s.is_estimated_leave ? " (推定)" : ""}
                          </Box>
                        </Box>
                        <Box>
                          <Box component="span" className="opacity-70 mr-1">
                            滞在
                          </Box>
                          <Box component="span" className="tabular-nums">
                            {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : "—"}
                          </Box>
                        </Box>
                      </Stack>
                    }
                  >
                    <Box
                      component={Link}
                      to={`/instances/${s.instance_id}`}
                      onMouseEnter={() => setHoveredKey(key)}
                      onMouseLeave={() => setHoveredKey((k) => (k === key ? null : k))}
                      className="absolute top-[2px] bottom-[2px] rounded-sm min-w-[2px] transition-colors cursor-pointer block"
                      sx={{
                        left: `${leftPct.toFixed(3)}%`,
                        width: `${widthPct.toFixed(3)}%`,
                        backgroundColor: active ? "rgba(13,110,253,0.95)" : "rgba(13,110,253,0.6)",
                      }}
                    />
                  </Tooltip>
                );
              })}
            </Box>
          </Box>
        );
      })}
    </Box>
  );
}
