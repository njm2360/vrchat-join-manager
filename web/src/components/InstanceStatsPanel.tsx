import { Box, Divider, Skeleton, Stack, Typography } from "@mui/material";
import { useInstanceStats } from "@/api/queries";
import { fmtDuration } from "@/utils/format";

interface Props {
  instanceId: number;
}

function StatItem({ label, value }: { label: string; value: string }) {
  return (
    <Box sx={{ minWidth: 72 }}>
      <Typography variant="caption" color="text.secondary" className="block leading-none">
        {label}
      </Typography>
      <Typography variant="h6" className="font-bold leading-tight">
        {value}
      </Typography>
    </Box>
  );
}

export default function InstanceStatsPanel({ instanceId }: Props) {
  const { data: stats, isLoading } = useInstanceStats(instanceId);

  if (isLoading || !stats) {
    return (
      <Box sx={{ px: 2, py: 1.5 }}>
        <Skeleton variant="rounded" height={48} />
      </Box>
    );
  }

  return (
    <Box className="border-b border-neutral-200" sx={{ px: 2, py: 1.5 }}>
      <Stack
        direction="row"
        spacing={3}
        useFlexGap
        sx={{ flexWrap: "wrap", alignItems: "center", rowGap: 1.5 }}
        divider={<Divider orientation="vertical" flexItem />}
      >
        <StatItem label="在室中" value={`${stats.present_count} 人`} />
        <StatItem label="最大同時" value={`${stats.peak_concurrent} 人`} />
        <StatItem label="ユニーク訪問" value={`${stats.visitor_count.toLocaleString()} 人`} />
        <StatItem label="リピーター" value={`${stats.repeat_visitor_count.toLocaleString()} 人`} />
        <StatItem label="総セッション" value={stats.session_count.toLocaleString()} />
        <StatItem label="総入退場" value={stats.event_count.toLocaleString()} />
        <StatItem label="合計滞在" value={fmtDuration(stats.total_duration_seconds)} />
        <StatItem label="平均滞在" value={fmtDuration(stats.avg_session_seconds)} />
      </Stack>
    </Box>
  );
}
