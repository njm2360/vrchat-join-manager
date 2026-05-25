import { useState } from 'react'
import {
  Box,
  Button,
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
  Typography,
} from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import type { Dayjs } from 'dayjs'
import { useEvents } from '../../api/queries'
import { fmtDateFull } from '../../utils/format'

interface Props {
  instanceId: number
}

export default function EventsTab({ instanceId }: Props) {
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')
  const [start, setStart] = useState<Dayjs | null>(null)
  const [end, setEnd] = useState<Dayjs | null>(null)
  const [applied, setApplied] = useState<{ start?: string; end?: string }>({})

  const { data: events = [] } = useEvents(instanceId, { order, ...applied })

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <DateTimePicker
          label="開始"
          value={start}
          onChange={setStart}
          slotProps={{ textField: { size: 'small' } }}
        />
        <Box className="text-neutral-500 text-sm">〜</Box>
        <DateTimePicker
          label="終了"
          value={end}
          onChange={setEnd}
          slotProps={{ textField: { size: 'small' } }}
        />
        <Button
          variant="contained"
          size="small"
          onClick={() => setApplied({ start: start?.toISOString(), end: end?.toISOString() })}
        >
          更新
        </Button>
      </Stack>
      <TableContainer component={Paper} variant="outlined" className="max-h-[520px]">
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell width={160} sortDirection={order}>
                <TableSortLabel
                  active
                  direction={order}
                  onClick={() => setOrder(order === 'asc' ? 'desc' : 'asc')}
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
              <TableRow>
                <TableCell colSpan={3} align="center">
                  <Typography variant="body2" color="text.secondary">
                    データなし
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              events.map((ev) => (
                <TableRow key={ev.id} hover>
                  <TableCell>{fmtDateFull(ev.timestamp)}</TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      label={ev.event_type === 'join' ? 'JOIN' : 'LEAVE'}
                      color={ev.event_type === 'join' ? 'success' : 'error'}
                      className="w-[72px]"
                    />
                  </TableCell>
                  <TableCell>{ev.display_name}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}
