import { useState } from "react";
import {
  Box,
  CircularProgress,
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
import { useSessionsInfinite } from "@/api/queries";
import { fmtDateFull, fmtDuration } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import DateRangeFilter from "@/components/DateRangeFilter";
import PlayerLink from "@/components/PlayerLink";
import LeaveCell from "@/components/LeaveCell";
import TablePlaceholderRow from "@/components/TablePlaceholderRow";
import VirtualPaddingRow from "@/components/VirtualPaddingRow";
import { useSortState } from "@/hooks/useSortState";

interface Props {
  instanceId: number;
}

type SortKey = "internal_id" | "display_name" | "join_ts" | "leave_ts" | "duration_seconds";

const COLUMNS: { key: SortKey; label: string; width?: number; align?: "right" }[] = [
  { key: "internal_id", label: "ID", width: 64, align: "right" },
  { key: "display_name", label: "名前" },
  { key: "join_ts", label: "入室", width: 160 },
  { key: "leave_ts", label: "退室", width: 160 },
  { key: "duration_seconds", label: "滞在時間", width: 120, align: "right" },
];

export default function SessionsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("leave_ts", "asc");
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});

  const query = useSessionsInfinite(instanceId, { sort_by: sortBy, order, ...applied });
  const {
    items: sessions,
    scrollRef,
    virtualItems,
    paddingTop,
    paddingBottom,
    measureElement,
  } = useInfiniteTable(query);

  return (
    <Stack spacing={2}>
      <DateRangeFilter onApply={setApplied} />
      <TableContainer
        ref={scrollRef}
        component={Paper}
        variant="outlined"
        className="max-h-[520px]"
      >
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {COLUMNS.map((c) => (
                <TableCell
                  key={c.key}
                  width={c.width}
                  align={c.align}
                  sortDirection={sortBy === c.key ? order : false}
                >
                  <TableSortLabel
                    active={sortBy === c.key}
                    direction={sortBy === c.key ? order : "asc"}
                    onClick={() => toggleSort(c.key)}
                  >
                    {c.label}
                  </TableSortLabel>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {sessions.length === 0 ? (
              <TablePlaceholderRow
                colSpan={COLUMNS.length}
                loading={query.isLoading}
                emptyText="データなし"
              />
            ) : (
              <>
                <VirtualPaddingRow height={paddingTop} colSpan={COLUMNS.length} />
                {virtualItems.map((vi) => {
                  const s = sessions[vi.index];
                  return (
                    <TableRow key={s.id} hover ref={measureElement} data-index={vi.index}>
                      <TableCell align="right" className="text-neutral-500">
                        {s.internal_id}
                      </TableCell>
                      <TableCell className="truncate max-w-[200px]">
                        <PlayerLink
                          userId={s.user_id}
                          displayName={s.display_name}
                          instanceId={instanceId}
                        />
                      </TableCell>
                      <TableCell>{fmtDateFull(s.join_ts)}</TableCell>
                      <TableCell>
                        <LeaveCell s={s} />
                      </TableCell>
                      <TableCell align="right">
                        {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : "—"}
                      </TableCell>
                    </TableRow>
                  );
                })}
                <VirtualPaddingRow height={paddingBottom} colSpan={COLUMNS.length} />
              </>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      {query.isFetchingNextPage && (
        <Box sx={{ display: "flex", justifyContent: "center", py: 1 }}>
          <CircularProgress size={20} />
        </Box>
      )}
    </Stack>
  );
}
