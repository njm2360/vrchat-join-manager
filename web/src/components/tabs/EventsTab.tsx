import { useState } from "react";
import {
  Chip,
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
import { useEvents } from "@/api/queries";
import { fmtDateFull } from "@/utils/format";
import DateRangeFilter from "@/components/DateRangeFilter";
import PlayerLink from "@/components/PlayerLink";
import TablePlaceholderRow from "@/components/TablePlaceholderRow";

interface Props {
  instanceId: number;
}

export default function EventsTab({ instanceId }: Props) {
  const [order, setOrder] = useState<"asc" | "desc">("desc");
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({});

  const { data: events = [], isLoading } = useEvents(instanceId, { order, ...applied });

  return (
    <Stack spacing={2}>
      <DateRangeFilter onApply={setApplied} />
      <TableContainer component={Paper} variant="outlined" className="max-h-[520px]">
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
              <TablePlaceholderRow colSpan={3} loading={isLoading} emptyText="データなし" />
            ) : (
              events.map((ev) => (
                <TableRow key={ev.id} hover>
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
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  );
}
