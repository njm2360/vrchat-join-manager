import { useState } from 'react'
import {
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Paper,
} from '@mui/material'
import { useSessions } from '@/api/queries'
import { fmtDateFull, fmtDuration } from '@/utils/format'
import DateRangeFilter from '@/components/DateRangeFilter'
import PlayerLink from '@/components/PlayerLink'
import LeaveCell from '@/components/LeaveCell'
import TablePlaceholderRow from '@/components/TablePlaceholderRow'
import { useSortState } from '@/hooks/useSortState'

interface Props {
  instanceId: number
}

type SortKey = 'display_name' | 'join_ts' | 'leave_ts' | 'duration_seconds'

const COLUMNS: { key: SortKey; label: string; width?: number; align?: 'right' }[] = [
  { key: 'display_name', label: '名前' },
  { key: 'join_ts', label: '入室', width: 160 },
  { key: 'leave_ts', label: '退室', width: 160 },
  { key: 'duration_seconds', label: '滞在時間', width: 120, align: 'right' },
]

export default function SessionsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>('leave_ts', 'asc')
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({})

  const { data: sessions = [], isLoading } = useSessions(instanceId, {
    sort_by: sortBy,
    order,
    ...applied,
  })

  return (
    <Stack spacing={2}>
      <DateRangeFilter onApply={setApplied} />
      <TableContainer component={Paper} variant="outlined" className="max-h-[520px]">
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
                    direction={sortBy === c.key ? order : 'asc'}
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
              <TablePlaceholderRow colSpan={COLUMNS.length} loading={isLoading} emptyText="データなし" />
            ) : (
              sessions.map((s) => (
                <TableRow key={s.id} hover>
                  <TableCell className="truncate max-w-[200px]">
                    <PlayerLink userId={s.user_id} displayName={s.display_name} instanceId={instanceId} />
                  </TableCell>
                  <TableCell>{fmtDateFull(s.join_ts)}</TableCell>
                  <TableCell><LeaveCell s={s} /></TableCell>
                  <TableCell align="right">
                    {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}
