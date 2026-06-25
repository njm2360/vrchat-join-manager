import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Box, Popover, Stack, Tooltip, Typography, useMediaQuery, useTheme } from "@mui/material";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import type { PlayerSessionOut } from "@/api/schemas";
import { fmtDateFull, fmtDuration } from "@/utils/format";

const DOW = ["日", "月", "火", "水", "木", "金", "土"];
const DAY_MS = 86_400_000;
const LABEL_CLS = "basis-[50px] sm:basis-[76px] shrink-0";

interface Props {
  year: number;
  month: number;
  sessions: PlayerSessionOut[];
}

export default function SessionCalendar({ year, month, sessions }: Props) {
  const [nowMs] = useState(() => Date.now());
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const [popover, setPopover] = useState<{ anchorEl: HTMLElement; s: PlayerSessionOut } | null>(
    null,
  );

  const rows = useMemo(() => {
    const days = new Date(year, month + 1, 0).getDate();
    const parsed = sessions.map((s) => ({
      s,
      start: new Date(s.join_ts).getTime(),
      end: s.leave_ts ? new Date(s.leave_ts).getTime() : nowMs,
    }));
    return Array.from({ length: days }, (_, i) => {
      const d = i + 1;
      const dayStart = new Date(year, month, d).getTime();
      const dayEnd = dayStart + DAY_MS;
      const dow = new Date(year, month, d).getDay();
      const segs = parsed
        .map(({ s, start, end }) => {
          if (start >= dayEnd || end <= dayStart) return null;
          const segStart = Math.max(start, dayStart);
          const segEnd = Math.min(end, dayEnd);
          return {
            s,
            key: s.join_ts,
            leftPct: ((segStart - dayStart) / DAY_MS) * 100,
            widthPct: Math.max(0.2, ((segEnd - segStart) / DAY_MS) * 100),
          };
        })
        .filter((x): x is NonNullable<typeof x> => x !== null);
      return { d, dow, segs };
    });
  }, [sessions, year, month, nowMs]);

  return (
    <>
      <HourScale />
      <Box className="text-[13px]">
        {rows.map(({ d, dow, segs }) => {
          const color = dow === 0 ? "text-red-600" : dow === 6 ? "text-blue-600" : "text-inherit";

          return (
            <Box key={d} className="flex items-center h-[30px] border-b border-neutral-100">
              <Box
                className={`${LABEL_CLS} text-right pr-2 text-xs tabular-nums select-none ${color}`}
              >
                <Box component="span" className="inline-block w-[2ch] text-right">
                  {d}
                </Box>
                ({DOW[dow]})
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
                {segs.map(({ s, key, leftPct, widthPct }) => (
                  <SessionBar
                    key={key}
                    s={s}
                    leftPct={leftPct}
                    widthPct={widthPct}
                    isMobile={isMobile}
                    onTap={(el) => setPopover({ anchorEl: el, s })}
                  />
                ))}
              </Box>
            </Box>
          );
        })}
        <Popover
          open={!!popover}
          anchorEl={popover?.anchorEl ?? null}
          onClose={() => setPopover(null)}
          anchorOrigin={{ vertical: "top", horizontal: "center" }}
          transformOrigin={{ vertical: "bottom", horizontal: "center" }}
        >
          {popover && (
            <Box className="p-2.5 min-w-[200px]">
              <SessionDetail s={popover.s} />
              <Typography
                component={Link}
                to={`/instances/${popover.s.instance_id}`}
                onClick={() => setPopover(null)}
                variant="body2"
                className="mt-2 inline-flex items-center gap-1 no-underline font-medium"
                color="primary"
              >
                <OpenInNewIcon fontSize="inherit" />
                インスタンスを開く
              </Typography>
            </Box>
          )}
        </Popover>
      </Box>
    </>
  );
}

function HourScale() {
  return (
    <Box className="flex items-center h-5 mb-1">
      <Box className={LABEL_CLS} />
      <Box className="flex-1 relative text-[10px] text-neutral-500 tabular-nums">
        <span className="absolute left-0">0</span>
        <span className="absolute left-1/4 -translate-x-1/2">6</span>
        <span className="absolute left-1/2 -translate-x-1/2">12</span>
        <span className="absolute left-3/4 -translate-x-1/2">18</span>
        <span className="absolute right-0">24</span>
      </Box>
    </Box>
  );
}

interface SessionBarProps {
  s: PlayerSessionOut;
  leftPct: number;
  widthPct: number;
  isMobile: boolean;
  onTap: (anchorEl: HTMLElement) => void;
}

function SessionBar({ s, leftPct, widthPct, isMobile, onTap }: SessionBarProps) {
  const sx = {
    left: `${leftPct.toFixed(3)}%`,
    width: `${widthPct.toFixed(3)}%`,
    backgroundColor: "rgba(13,110,253,0.6)",
    "&:hover": { backgroundColor: "rgba(13,110,253,0.95)" },
  };
  const cls = `absolute top-[2px] bottom-[2px] rounded-sm transition-colors cursor-pointer block ${
    isMobile ? "min-w-[10px]" : "min-w-[2px]"
  }`;

  if (isMobile) {
    return <Box className={cls} sx={sx} onClick={(e) => onTap(e.currentTarget)} />;
  }

  return (
    <Tooltip arrow placement="top" title={<SessionDetail s={s} />}>
      <Box component={Link} to={`/instances/${s.instance_id}`} className={cls} sx={sx} />
    </Tooltip>
  );
}

function SessionDetail({ s }: { s: PlayerSessionOut }) {
  return (
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
  );
}
