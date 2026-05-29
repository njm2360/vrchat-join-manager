import {
  Box,
  Button,
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
import { useVisitors } from '../../api/queries'
import { fmtDateFull, fmtDuration } from '../../utils/format'
import PlayerLink from '../PlayerLink'
import { useSortState } from '../../hooks/useSortState'

interface Props {
  instanceId: number
}

type SortKey = 'display_name' | 'first_seen' | 'last_seen' | 'join_count' | 'total_duration_seconds'

const COLUMNS: { key: SortKey; label: string; width?: number; align?: 'right' }[] = [
  { key: 'display_name', label: '名前' },
  { key: 'first_seen', label: '初訪問', width: 150 },
  { key: 'last_seen', label: '最終訪問', width: 150 },
  { key: 'join_count', label: '訪問回数', width: 120, align: 'right' },
  { key: 'total_duration_seconds', label: '合計滞在', width: 120, align: 'right' },
]

export default function VisitorsTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>('last_seen', 'desc')

  const { data: visitors = [], refetch } = useVisitors(instanceId, {
    sort_by: sortBy,
    order,
  })

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Button variant="contained" size="small" onClick={() => refetch()}>
          更新
        </Button>
        <Typography variant="h6" className="font-bold">
          {visitors.length} 人
        </Typography>
      </Stack>
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
            {visitors.length === 0 ? (
              <TableRow>
                <TableCell colSpan={COLUMNS.length} align="center">
                  <Typography variant="body2" color="text.secondary">
                    データなし
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              visitors.map((v) => (
                <TableRow key={v.user_id} hover>
                  <TableCell className="truncate max-w-[240px]">
                    <PlayerLink userId={v.user_id} displayName={v.display_name} instanceId={instanceId} />
                  </TableCell>
                  <TableCell>{fmtDateFull(v.first_seen)}</TableCell>
                  <TableCell>{fmtDateFull(v.last_seen)}</TableCell>
                  <TableCell align="right">{v.join_count}回</TableCell>
                  <TableCell align="right">
                    {v.total_duration_seconds != null
                      ? fmtDuration(v.total_duration_seconds)
                      : '—'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <Box />
    </Stack>
  )
}
