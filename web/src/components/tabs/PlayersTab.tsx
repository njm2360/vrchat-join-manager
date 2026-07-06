import {
  Box,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Paper,
} from "@mui/material";
import { usePlayers } from "@/api/queries";
import { fmtDateFull } from "@/utils/format";
import PlayerLink from "@/components/PlayerLink";
import TablePlaceholderRow from "@/components/table/TablePlaceholderRow";
import SortableTableHead, { type TableColumn } from "@/components/table/SortableTableHead";
import { useSortState } from "@/hooks/useSortState";

interface Props {
  instanceId: number;
}

type SortKey = "internal_id" | "display_name" | "join_ts";

const COLUMNS: TableColumn<SortKey>[] = [
  { key: "internal_id", label: "ID", width: 64, align: "right", sortKey: "internal_id" },
  { key: "display_name", label: "名前", sortKey: "display_name" },
  { key: "discord_id", label: "Discord ID", width: 140 },
  { key: "join_ts", label: "入室日時", width: 160, sortKey: "join_ts" },
];

export default function PlayersTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("internal_id", "asc", "asc");

  const {
    data: players = [],
    isLoading,
    isError,
  } = usePlayers(instanceId, {
    sort_by: sortBy,
    order,
  });

  return (
    <Stack spacing={2}>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small" stickyHeader>
          <SortableTableHead columns={COLUMNS} sortBy={sortBy} order={order} onSort={toggleSort} />
          <TableBody>
            {players.length === 0 ? (
              <TablePlaceholderRow
                colSpan={COLUMNS.length}
                loading={isLoading}
                error={isError}
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
