import { useMemo } from "react";
import {
  Chip,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Typography,
} from "@mui/material";
import PlayerLink from "@/components/PlayerLink";
import type { InstColor, Violation } from "@/utils/violations";
import { fmtDateFull, fmtDuration } from "@/utils/format";

export type VSortKey = "display_name" | "join_ts" | "instance" | "diff" | "duration_seconds";

interface ViolationsTableProps {
  violations: Violation[];
  sort: { by: VSortKey; dir: "asc" | "desc" };
  onSort: (key: VSortKey) => void;
  highlightedTs: number | null;
  onPickTime: (t: number) => void;
  id1: number;
  id2: number;
}

const COLS: { key: VSortKey; label: string }[] = [
  { key: "display_name", label: "ユーザー名" },
  { key: "join_ts", label: "違反時刻" },
  { key: "instance", label: "参加先" },
  { key: "diff", label: "人数差" },
  { key: "duration_seconds", label: "滞在時間" },
];

export default function ViolationsTable({
  violations,
  sort,
  onSort,
  highlightedTs,
  onPickTime,
  id1,
  id2,
}: ViolationsTableProps) {
  const sorted = useMemo(() => {
    return [...violations].sort((a, b) => {
      let va: number | string;
      let vb: number | string;
      if (sort.by === "join_ts") {
        va = a.join_ts.getTime();
        vb = b.join_ts.getTime();
      } else if (sort.by === "duration_seconds") {
        va = a.duration_seconds ?? -1;
        vb = b.duration_seconds ?? -1;
      } else {
        va = a[sort.by] as string;
        vb = b[sort.by] as string;
      }
      if (va < vb) return sort.dir === "asc" ? -1 : 1;
      if (va > vb) return sort.dir === "asc" ? 1 : -1;
      return 0;
    });
  }, [violations, sort]);

  if (violations.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" className="text-center py-3">
        違反なし
      </Typography>
    );
  }

  return (
    <TableContainer component={Paper} variant="outlined" square>
      <Table size="small">
        <TableHead>
          <TableRow>
            {COLS.map((c) => (
              <TableCell key={c.key} sortDirection={sort.by === c.key ? sort.dir : false}>
                <TableSortLabel
                  active={sort.by === c.key}
                  direction={sort.by === c.key ? sort.dir : "asc"}
                  onClick={() => onSort(c.key)}
                >
                  {c.label}
                </TableSortLabel>
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {sorted.map((v, i) => {
            const ts = v.join_ts.getTime();
            const color: InstColor = v.instance;
            return (
              <TableRow
                key={i}
                hover
                selected={highlightedTs === ts}
                onClick={() => onPickTime(ts)}
                sx={{ cursor: "pointer" }}
              >
                <TableCell>
                  <PlayerLink
                    userId={v.user_id}
                    displayName={v.display_name}
                    instanceId={v.instance === "blue" ? id1 : id2}
                    stopPropagation
                  />
                </TableCell>
                <TableCell>{fmtDateFull(v.join_ts)}</TableCell>
                <TableCell>
                  <Chip
                    size="small"
                    label={color === "blue" ? "青" : "赤"}
                    sx={{
                      backgroundColor: color === "blue" ? "#0d6efd" : "#dc3545",
                      color: "#fff",
                    }}
                  />
                </TableCell>
                <TableCell>+{v.diff}</TableCell>
                <TableCell>
                  {v.duration_seconds != null ? fmtDuration(v.duration_seconds) : "—"}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
