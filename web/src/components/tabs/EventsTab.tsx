import { useState } from "react";
import { Chip, Stack, TableCell } from "@mui/material";
import { useEventsInfinite } from "@/api/queries";
import { fmtDateFull } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import { useSortState } from "@/hooks/useSortState";
import DateRangeFilter from "@/components/DateRangeFilter";
import PlayerLink from "@/components/PlayerLink";
import VirtualTable from "@/components/table/VirtualTable";
import type { TableColumn } from "@/components/table/SortableTableHead";

interface Props {
  instanceId: number;
}

type SortKey = "timestamp";

const COLUMNS: TableColumn<SortKey>[] = [
  { key: "timestamp", label: "日時", width: 160, sortKey: "timestamp" },
  { key: "type", label: "種別", width: 120 },
  { key: "name", label: "名前" },
];

export default function EventsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("timestamp", "desc", "desc");
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});

  const query = useEventsInfinite(instanceId, { order, ...applied });
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
        isError={query.isError}
        isFetchingNextPage={query.isFetchingNextPage}
        emptyText="データなし"
        rowKey={(ev) => ev.id}
        renderCells={(ev) => (
          <>
            <TableCell>{fmtDateFull(ev.timestamp)}</TableCell>
            <TableCell>
              <Chip
                size="small"
                label={ev.event_type === "join" ? "JOIN" : "LEAVE"}
                color={ev.event_type === "join" ? "success" : "error"}
                className="w-[72px]"
              />
            </TableCell>
            <TableCell>
              <PlayerLink
                userId={ev.user_id}
                displayName={ev.display_name}
                instanceId={instanceId}
              />
            </TableCell>
          </>
        )}
      />
    </Stack>
  );
}
