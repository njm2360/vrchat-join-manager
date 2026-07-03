import {
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import type { PlayerSessionOut } from "@/api/schemas";
import { fmtDuration } from "@/utils/format";
import JoinCell from "@/components/JoinCell";
import LeaveCell from "@/components/LeaveCell";

interface Props {
  sessions: PlayerSessionOut[];
  isLoading?: boolean;
  emptyText?: string;
  highlight?: number | null;
  onHover?: (idx: number | null) => void;
}

export default function PlayerSessionTable({
  sessions,
  isLoading = false,
  emptyText = "セッション履歴なし",
  highlight = null,
  onHover,
}: Props) {
  return (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell width={160}>入室</TableCell>
            <TableCell width={160}>退室</TableCell>
            <TableCell width={120} align="right">
              滞在時間
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {isLoading ? (
            <MessageRow text="読み込み中..." />
          ) : sessions.length === 0 ? (
            <MessageRow text={emptyText} />
          ) : (
            sessions.map((s, i) => (
              <TableRow
                key={s.id}
                hover
                selected={onHover ? highlight === i : undefined}
                onMouseEnter={onHover ? () => onHover(i) : undefined}
                onMouseLeave={onHover ? () => onHover(null) : undefined}
              >
                <TableCell>
                  <JoinCell s={s} />
                </TableCell>
                <TableCell>
                  <LeaveCell s={s} />
                </TableCell>
                <TableCell align="right">
                  {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : "—"}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function MessageRow({ text }: { text: string }) {
  return (
    <TableRow>
      <TableCell colSpan={3} align="center">
        <Typography variant="body2" color="text.secondary">
          {text}
        </Typography>
      </TableCell>
    </TableRow>
  );
}
