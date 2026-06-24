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
import { useVisitorsInfinite } from "@/api/queries";
import { fmtDateFull, fmtDuration } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import PlayerLink from "@/components/PlayerLink";
import TablePlaceholderRow from "@/components/TablePlaceholderRow";
import VirtualPaddingRow from "@/components/VirtualPaddingRow";
import { useSortState } from "@/hooks/useSortState";

interface Props {
  instanceId: number;
}

type SortKey =
  | "display_name"
  | "first_seen"
  | "last_seen"
  | "join_count"
  | "total_duration_seconds";

const COLUMNS: { key: SortKey; label: string; width?: number; align?: "right" }[] = [
  { key: "display_name", label: "名前" },
  { key: "first_seen", label: "初訪問", width: 150 },
  { key: "last_seen", label: "最終訪問", width: 150 },
  { key: "join_count", label: "訪問回数", width: 120, align: "right" },
  { key: "total_duration_seconds", label: "合計滞在", width: 120, align: "right" },
];

export default function VisitorsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>("last_seen", "desc");

  const query = useVisitorsInfinite(instanceId, { sort_by: sortBy, order });
  const {
    items: visitors,
    scrollRef,
    virtualItems,
    paddingTop,
    paddingBottom,
    measureElement,
  } = useInfiniteTable(query);

  return (
    <Stack spacing={2}>
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
            {visitors.length === 0 ? (
              <TablePlaceholderRow
                colSpan={COLUMNS.length}
                loading={query.isLoading}
                emptyText="データなし"
              />
            ) : (
              <>
                <VirtualPaddingRow height={paddingTop} colSpan={COLUMNS.length} />
                {virtualItems.map((vi) => {
                  const v = visitors[vi.index];
                  return (
                    <TableRow key={v.user_id} hover ref={measureElement} data-index={vi.index}>
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
                        {v.total_duration_seconds != null
                          ? fmtDuration(v.total_duration_seconds)
                          : "—"}
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
