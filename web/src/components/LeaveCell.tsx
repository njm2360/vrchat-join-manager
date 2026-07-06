import { Box, Chip, Tooltip } from "@mui/material";
import type { SessionOut } from "@/api/schemas";
import { fmtDateFull } from "@/utils/format";

interface Props {
  s: Pick<SessionOut, "leave_ts" | "is_estimated_leave">;
}

export default function LeaveCell({ s }: Props) {
  if (!s.leave_ts) return <Chip size="small" color="success" label="在室中" />;
  return (
    <Box className="flex items-center gap-1">
      <span>{fmtDateFull(s.leave_ts)}</span>
      {s.is_estimated_leave && (
        <Tooltip title="退室時刻を使用した推定値です">
          <Chip size="small" color="warning" label="!" className="h-[18px]!" />
        </Tooltip>
      )}
    </Box>
  );
}
