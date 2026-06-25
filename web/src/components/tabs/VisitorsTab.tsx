import { Stack, TableCell } from "@mui/material";
import { useVisitorsInfinite } from "@/api/queries";
import { fmtDateFull, fmtDuration } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import { useSortState } from "@/hooks/useSortState";
import PlayerLink from "@/components/PlayerLink";
import VirtualTable from "@/components/table/VirtualTable";
import type { TableColumn } from "@/components/table/SortableTableHead";

interface Props {
  instanceId: number;
}

type SortKey =
  | "display_name"
  | "first_seen"
  | "last_seen"
  | "join_count"
  | "total_duration_seconds";

const COLUMNS: TableColumn<SortKey>[] = [
  { key: "display_name", label: "名前", sortKey: "display_name" },
  { key: "first_seen", label: "初訪問", width: 150, sortKey: "first_seen" },
  { key: "last_seen", label: "最終訪問", width: 150, sortKey: "last_seen" },
  { key: "join_count", label: "訪問回数", width: 120, align: "right", sortKey: "join_count" },
  {
    key: "total_duration_seconds",
    label: "合計滞在",
    width: 120,
    align: "right",
    sortKey: "total_duration_seconds",
  },
];

export default function VisitorsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("last_seen", "desc");

  const query = useVisitorsInfinite(instanceId, { sort_by: sortBy, order });
  const table = useInfiniteTable(query);

  return (
    <Stack spacing={2}>
      <VirtualTable
        columns={COLUMNS}
        sortBy={sortBy}
        order={order}
        onSort={toggleSort}
        table={table}
        isLoading={query.isLoading}
        isFetchingNextPage={query.isFetchingNextPage}
        emptyText="データなし"
        rowKey={(v) => v.user_id}
        renderCells={(v) => (
          <>
            <TableCell className="truncate max-w-[240px]">
              <PlayerLink
                userId={v.user_id}
                displayName={v.display_name}
                instanceId={instanceId}
              />
            </TableCell>
            <TableCell>{fmtDateFull(v.first_seen)}</TableCell>
            <TableCell>{fmtDateFull(v.last_seen)}</TableCell>
            <TableCell align="right">{v.join_count}回</TableCell>
            <TableCell align="right">
              {v.total_duration_seconds != null ? fmtDuration(v.total_duration_seconds) : "—"}
            </TableCell>
          </>
        )}
      />
    </Stack>
  );
}
