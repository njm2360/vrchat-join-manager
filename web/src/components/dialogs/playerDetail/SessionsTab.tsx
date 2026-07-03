import { Box, TableCell } from "@mui/material";
import { usePlayerSessionsInfinite } from "@/api/queries";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import { fmtDuration } from "@/utils/format";
import JoinCell from "@/components/JoinCell";
import LeaveCell from "@/components/LeaveCell";
import VirtualTable from "@/components/table/VirtualTable";
import type { TableColumn } from "@/components/table/SortableTableHead";

interface Props {
  userId: string;
}

const COLUMNS: TableColumn<never>[] = [
  { key: "join_ts", label: "入室", width: 160 },
  { key: "leave_ts", label: "退室", width: 160 },
  { key: "duration_seconds", label: "滞在時間", width: 120, align: "right" },
];

export default function SessionsTab({ userId }: Props) {
  const query = usePlayerSessionsInfinite(userId, { order: "desc" });
  const table = useInfiniteTable(query);

  return (
    <Box className="p-3">
      <VirtualTable
        columns={COLUMNS}
        sortBy={"" as never}
        order="desc"
        onSort={() => {}}
        table={table}
        isLoading={query.isLoading}
        isFetchingNextPage={query.isFetchingNextPage}
        emptyText="セッション履歴なし"
        rowKey={(s) => s.id}
        renderCells={(s) => (
          <>
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
    </Box>
  );
}
