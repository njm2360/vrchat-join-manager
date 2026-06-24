import { useState } from "react";
import {
  Box,
  Chip,
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
import { useEventsInfinite } from "@/api/queries";
import { fmtDateFull } from "@/utils/format";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import DateRangeFilter from "@/components/DateRangeFilter";
import PlayerLink from "@/components/PlayerLink";
import TablePlaceholderRow from "@/components/TablePlaceholderRow";
import VirtualPaddingRow from "@/components/VirtualPaddingRow";

interface Props {
  instanceId: number;
}

export default function EventsTab({ instanceId }: Props) {
  const [order, setOrder] = useState<"asc" | "desc">("desc");
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});

  const query = useEventsInfinite(instanceId, { order, ...applied });
  const {
    items: events,
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
              <TableCell width={160} sortDirection={order}>
                <TableSortLabel
                  active
                  direction={order}
                  onClick={() => setOrder(order === "asc" ? "desc" : "asc")}
                >
                  日時
                </TableSortLabel>
              </TableCell>
              <TableCell width={120}>種別</TableCell>
              <TableCell>名前</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.length === 0 ? (
              <TablePlaceholderRow colSpan={3} loading={query.isLoading} emptyText="データなし" />
            ) : (
              <>
                <VirtualPaddingRow height={paddingTop} colSpan={3} />
                {virtualItems.map((vi) => {
                  const ev = events[vi.index];
                  return (
                    <TableRow key={ev.id} hover ref={measureElement} data-index={vi.index}>
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
                    </TableRow>
                  );
                })}
                <VirtualPaddingRow height={paddingBottom} colSpan={3} />
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
