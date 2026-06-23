import { useState } from "react";
import { Box, Button, Divider, Skeleton, Stack, Typography } from "@mui/material";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import { useSnackbar } from "notistack";
import { fetchInstanceDiscordMentions, useInstanceStats } from "@/api/queries";
import { fmtDuration } from "@/utils/format";
import { copyText } from "@/utils/clipboard";

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
  const { enqueueSnackbar } = useSnackbar();
  const [copying, setCopying] = useState(false);

  const copyDiscord = async () => {
    setCopying(true);
    try {
      const ids = await fetchInstanceDiscordMentions(instanceId);
      if (ids.length === 0) {
        enqueueSnackbar("Discord IDが登録されているプレイヤーがいません", {
          variant: "info",
        });
        return;
      }
      await copyText(ids.map((id) => `@${id}`).join(" ") + " ");
      enqueueSnackbar(`${ids.length}人分のDiscord IDをコピーしました`, {
        variant: "success",
      });
    } catch {
      enqueueSnackbar("コピーに失敗しました", { variant: "error" });
    } finally {
      setCopying(false);
    }
  };

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
        <Box sx={{ flexGrow: 1 }} />
        <Button
          variant="outlined"
          size="small"
          startIcon={<ContentCopyIcon />}
          onClick={copyDiscord}
          disabled={copying}
        >
          在室者のDiscord IDをコピー
        </Button>
      </Stack>
    </Box>
  );
}
