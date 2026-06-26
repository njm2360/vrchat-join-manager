import { Box, Button, Stack, Typography } from "@mui/material";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import { Link as RouterLink } from "react-router-dom";
import type { PlayerDetailOut } from "@/api/schemas";
import { fmtDate, fmtDuration } from "@/utils/format";

interface Props {
  userId: string;
  detail: PlayerDetailOut | null;
}

export default function OverviewTab({ userId, detail }: Props) {
  if (!detail) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        読み込み中...
      </Typography>
    );
  }

  return (
    <Stack spacing={2.5} className="p-3">
      <Box className="grid grid-cols-2 md:grid-cols-3 gap-2">
        <StatCard label="通算訪問" value={`${detail.total_visits}回`} />
        <StatCard label="通算滞在" value={fmtDuration(detail.total_duration_seconds)} />
        <StatCard
          label="現在の状態"
          value={detail.in_room ? "在室中" : "未在室"}
          accent={detail.in_room ? "success" : undefined}
        />
        <StatCard
          label="初回訪問"
          value={detail.first_seen ? fmtDate(detail.first_seen) : "—"}
          small
        />
        <StatCard
          label="最終訪問"
          value={detail.last_seen ? fmtDate(detail.last_seen) : "—"}
          small
        />
      </Box>

      <Box className="flex justify-end">
        <Button
          variant="outlined"
          size="small"
          endIcon={<OpenInNewIcon />}
          component={RouterLink}
          to={`/players/${encodeURIComponent(userId)}`}
          target="_blank"
        >
          月別カレンダーを開く
        </Button>
      </Box>
    </Stack>
  );
}

function StatCard({
  label,
  value,
  small,
  accent,
}: {
  label: string;
  value: string;
  small?: boolean;
  accent?: "success";
}) {
  return (
    <Box className="border border-neutral-200 rounded-md p-2 bg-white">
      <Typography variant="caption" color="text.secondary" className="block">
        {label}
      </Typography>
      <Typography
        variant={small ? "body2" : "subtitle1"}
        className={`font-semibold ${accent === "success" ? "text-green-600" : ""}`}
      >
        {value}
      </Typography>
    </Box>
  );
}
