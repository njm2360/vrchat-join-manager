import { Box, Chip } from "@mui/material";
import type { SessionOut } from "@/api/schemas";
import { fmtDateFull } from "@/utils/format";

interface Props {
  s: Pick<SessionOut, "join_ts" | "is_estimated_join">;
}

export default function JoinCell({ s }: Props) {
  return (
    <Box className="flex items-center gap-1">
      <span>{fmtDateFull(s.join_ts)}</span>
      {s.is_estimated_join && (
        <Chip
          size="small"
          color="warning"
          label="!"
          title="観測開始時刻を使用した推定値です"
          className="h-[18px]!"
        />
      )}
    </Box>
  );
}
