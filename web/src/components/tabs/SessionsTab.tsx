import { useState } from "react";
import { Stack, TableCell } from "@mui/material";
import { useSessionsInfinite } from "@/api/queries";
import { fmtDuration } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import { useSortState } from "@/hooks/useSortState";
import DateRangeFilter from "@/components/DateRangeFilter";
import PlayerLink from "@/components/PlayerLink";
import JoinCell from "@/components/JoinCell";
import LeaveCell from "@/components/LeaveCell";
import VirtualTable from "@/components/table/VirtualTable";
import type { TableColumn } from "@/components/table/SortableTableHead";

interface Props {
  instanceId: number;
}

type SortKey = "internal_id" | "display_name" | "join_ts" | "leave_ts" | "duration_seconds";

const COLUMNS: TableColumn<SortKey>[] = [
  { key: "internal_id", label: "ID", width: 64, align: "right", sortKey: "internal_id" },
  { key: "display_name", label: "名前", sortKey: "display_name" },
  { key: "join_ts", label: "入室", width: 160, sortKey: "join_ts" },
  { key: "leave_ts", label: "退室", width: 160, sortKey: "leave_ts" },
  {
    key: "duration_seconds",
    label: "滞在時間",
    width: 120,
    align: "right",
    sortKey: "duration_seconds",
  },
];

export default function SessionsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("leave_ts", "asc");
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});

  const query = useSessionsInfinite(instanceId, { sort_by: sortBy, order, ...applied });
  const table = useInfiniteTable(query);

  return (
    <Stack spacing={2}>
      <DateRangeFilter onApply={setApplied} />
      <VirtualTable
        columns={COLUMNS}
        sortBy={sortBy}
        order={order}
        onSort={toggleSort}
        table={table}
        isLoading={query.isLoading}
        isFetchingNextPage={query.isFetchingNextPage}
        emptyText="データなし"
        rowKey={(s) => s.id}
        renderCells={(s) => (
          <>
            <TableCell align="right" className="text-neutral-500">
              {s.internal_id}
            </TableCell>
            <TableCell className="truncate max-w-[200px]">
              <PlayerLink userId={s.user_id} displayName={s.display_name} instanceId={instanceId} />
            </TableCell>
            <TableCell>
              <JoinCell s={s} />
            </TableCell>
            <TableCell>
              <LeaveCell s={s} />
            </TableCell>
            <TableCell align="right">
              {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : "—"}
            </TableCell>
          </>
        )}
      />
    </Stack>
  );
}
