import {
  Box,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Paper,
} from "@mui/material";
import { usePlayers } from "@/api/queries";
import { fmtDateFull } from "@/utils/format";
import PlayerLink from "@/components/PlayerLink";
import TablePlaceholderRow from "@/components/TablePlaceholderRow";
import { useSortState } from "@/hooks/useSortState";

interface Props {
  instanceId: number;
}

type SortKey = "internal_id" | "display_name" | "join_ts";

const COLUMNS: { key: SortKey; label: string; width?: number; align?: "right" }[] = [
  { key: "internal_id", label: "ID", width: 64, align: "right" },
  { key: "display_name", label: "名前" },
];

export default function PlayersTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("internal_id", "asc", "asc");

  const { data: players = [], isLoading } = usePlayers(instanceId, { sort_by: sortBy, order });

  return (
    <Stack spacing={2}>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {COLUMNS.map((c) => (
                <TableCell key={c.key} width={c.width} align={c.align}>
                  <TableSortLabel
                    active={sortBy === c.key}
                    direction={sortBy === c.key ? order : "asc"}
                    onClick={() => toggleSort(c.key)}
                  >
                    {c.label}
                  </TableSortLabel>
                </TableCell>
              ))}
              <TableCell width={140}>Discord ID</TableCell>
              <TableCell width={160}>
                <TableSortLabel
                  active={sortBy === "join_ts"}
                  direction={sortBy === "join_ts" ? order : "asc"}
                  onClick={() => toggleSort("join_ts")}
                >
                  入室日時
                </TableSortLabel>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {players.length === 0 ? (
              <TablePlaceholderRow
                colSpan={4}
                loading={isLoading}
                emptyText="在室中のプレイヤーなし"
              />
            ) : (
              players.map((p) => (
                <TableRow key={p.user_id} hover>
                  <TableCell align="right" className="text-neutral-500">
                    {p.internal_id ?? "—"}
                  </TableCell>
                  <TableCell>
                    <PlayerLink
                      userId={p.user_id}
                      displayName={p.display_name}
                      instanceId={instanceId}
                    />
                  </TableCell>
                  <TableCell>
                    {p.discord_id ? (
                      p.discord_id
                    ) : (
                      <Box component="span" className="text-neutral-400">
                        未登録
                      </Box>
                    )}
                  </TableCell>
                  <TableCell>{fmtDateFull(p.join_ts)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  );
}
