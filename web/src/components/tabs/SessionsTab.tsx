import { useState } from 'react'
import {
  Box,
  Chip,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Paper,
  Typography,
} from '@mui/material'
import { useSessions } from '../../api/queries'
import { fmtDateFull, fmtDuration } from '../../utils/format'
import type { SessionOut } from '../../api/schemas'
import { usePlayerDetailDialog } from '../usePlayerDetailDialog'
import DateRangeFilter from '../DateRangeFilter'

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
  const [sortBy, setSortBy] = useState<SortKey>('leave_ts')
  const [order, setOrder] = useState<'asc' | 'desc'>('asc')
  const { open: openPlayer } = usePlayerDetailDialog()
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({})

  const { data: sessions = [] } = useSessions(instanceId, {
    sort_by: sortBy,
    order,
    ...applied,
  })

  const toggleSort = (key: SortKey) => {
    if (key === sortBy) setOrder(order === 'asc' ? 'desc' : 'asc')
    else {
      setSortBy(key)
      setOrder('desc')
    }
  }

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
              <TableRow>
                <TableCell colSpan={COLUMNS.length} align="center">
                  <Typography variant="body2" color="text.secondary">
                    データなし
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              sessions.map((s) => (
                <TableRow key={s.id} hover>
                  <TableCell className="truncate max-w-[200px]">
                    <Link
                      component="button"
                      underline="hover"
                      onClick={() =>
                        openPlayer({
                          userId: s.user_id,
                          displayName: s.display_name,
                          instanceId,
                        })
                      }
                    >
                      {s.display_name}
                    </Link>
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

export function LeaveCell({ s }: { s: Pick<SessionOut, 'leave_ts' | 'is_estimated_leave'> }) {
  if (!s.leave_ts) return <Chip size="small" color="success" label="在室中" />
  return (
    <Box className="flex items-center gap-1">
      <span>{fmtDateFull(s.leave_ts)}</span>
      {s.is_estimated_leave && (
        <Chip
          size="small"
          color="warning"
          label="!"
          title="退室時刻を使用した推定値です"
          className="h-[18px]!"
        />
      )}
    </Box>
  )
}
