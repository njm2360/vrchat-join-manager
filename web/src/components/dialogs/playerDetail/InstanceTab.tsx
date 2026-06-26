import { useMemo, useState } from "react";
import { Box, Stack, Typography } from "@mui/material";
import { usePlayerSessions } from "@/api/queries";
import type { InstanceOut, PlayerSessionOut } from "@/api/schemas";
import { fmtDate } from "@/utils/format";
import PlayerSessionTable from "@/components/dialogs/playerDetail/PlayerSessionTable";

interface Props {
  userId: string;
  instance: InstanceOut | null;
}

export default function InstanceTab({ userId, instance }: Props) {
  const [highlight, setHighlight] = useState<number | null>(null);

  const { data: sessions = [], isLoading } = usePlayerSessions(
    userId,
    { instance_id: instance?.id, order: "asc" },
    { enabled: !!instance },
  );

  if (!instance) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        インスタンス情報を読み込み中...
      </Typography>
    );
  }

  if (isLoading) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        読み込み中...
      </Typography>
    );
  }

  if (sessions.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        このインスタンスでのセッションなし
      </Typography>
    );
  }

  return (
    <Stack spacing={1.5} className="p-3">
      <PlayerTimelineBar
        sessions={sessions}
        instance={instance}
        highlight={highlight}
        onHover={setHighlight}
      />
      <PlayerSessionTable sessions={sessions} highlight={highlight} onHover={setHighlight} />
    </Stack>
  );
}

interface BarProps {
  sessions: PlayerSessionOut[];
  instance: InstanceOut;
  highlight: number | null;
  onHover: (idx: number | null) => void;
}

function PlayerTimelineBar({ sessions, instance, highlight, onHover }: BarProps) {
  const [nowMs] = useState(() => Date.now());
  const { instStart, instEnd, bars } = useMemo(() => {
    const instStart = new Date(instance.opened_at).getTime();
    const instEnd = instance.closed_at ? new Date(instance.closed_at).getTime() : nowMs;
    const total = Math.max(1, instEnd - instStart);
    const VW = 1000;
    const toX = (ts: string) => ((new Date(ts).getTime() - instStart) / total) * VW;
    const bars = sessions.map((s, i) => {
      const x1 = toX(s.join_ts);
      const x2 = s.leave_ts ? toX(s.leave_ts) : VW;
      return { i, x1, w: Math.max(3, x2 - x1) };
    });
    return { instStart, instEnd, bars };
  }, [sessions, instance, nowMs]);

  const BAR_H = 26;
  return (
    <Box>
      <Box
        component="svg"
        viewBox={`0 0 1000 ${BAR_H}`}
        preserveAspectRatio="none"
        className="w-full block"
        sx={{ height: BAR_H }}
      >
        <rect x={0} y={0} width={1000} height={BAR_H} fill="#dee2e6" rx={3} />
        {bars.map((b) => (
          <rect
            key={b.i}
            x={b.x1}
            y={0}
            width={b.w}
            height={BAR_H}
            rx={2}
            fill={highlight === b.i ? "rgba(13,110,253,0.92)" : "rgba(13,110,253,0.55)"}
            stroke={highlight === b.i ? "rgba(13,110,253,1)" : "none"}
            onMouseEnter={() => onHover(b.i)}
            onMouseLeave={() => onHover(null)}
            style={{ cursor: "pointer" }}
          />
        ))}
      </Box>
      <Box className="flex justify-between text-xs text-neutral-500 mt-1">
        <span>{fmtDate(instStart)}</span>
        <span>{fmtDate(instEnd)}</span>
      </Box>
    </Box>
  );
}
