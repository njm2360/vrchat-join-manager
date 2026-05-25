import { useMemo, useState } from 'react'
import {
  Box,
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
} from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import { usePlayerSessions } from '../../api/queries'
import type { InstanceOut, PlayerSessionOut } from '../../api/schemas'
import { fmtDateFull, fmtDuration } from '../../utils/format'
import { LeaveCell } from '../tabs/SessionsTab'

interface Props {
  open: boolean
  onClose: () => void
  userId: string
  displayName: string
  instance: InstanceOut
}

export default function SessionDialog({ open, onClose, userId, displayName, instance }: Props) {
  const [highlight, setHighlight] = useState<number | null>(null)

  const { data: sessions = [], isLoading } = usePlayerSessions(
    userId,
    { instance_id: instance.id, order: 'asc' },
    { enabled: open },
  )

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth scroll="paper">
      <DialogTitle className="flex items-center gap-2">
        <Box className="flex-1">{displayName} のセッション</Box>
        <Button
          size="small"
          variant="outlined"
          component="a"
          href={`/players/${encodeURIComponent(userId)}?display_name=${encodeURIComponent(displayName)}&world_id=${encodeURIComponent(instance.world_id)}`}
          target="_blank"
        >
          このプレイヤーのセッション
        </Button>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers className="p-0!">
        {isLoading ? (
          <Typography variant="body2" color="text.secondary" className="p-3 text-center">
            読み込み中...
          </Typography>
        ) : sessions.length === 0 ? (
          <Typography variant="body2" color="text.secondary" className="p-3 text-center">
            データなし
          </Typography>
        ) : (
          <Stack spacing={1.5} className="p-3">
            <PlayerTimelineBar
              sessions={sessions}
              instance={instance}
              highlight={highlight}
              onHover={setHighlight}
            />
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell width={160}>入室</TableCell>
                    <TableCell width={160}>退室</TableCell>
                    <TableCell width={120} align="right">滞在時間</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {sessions.map((s, i) => (
                    <TableRow
                      key={s.id}
                      hover
                      selected={highlight === i}
                      onMouseEnter={() => setHighlight(i)}
                      onMouseLeave={() => setHighlight(null)}
                    >
                      <TableCell>{fmtDateFull(s.join_ts)}</TableCell>
                      <TableCell><LeaveCell s={s} /></TableCell>
                      <TableCell align="right">
                        {s.duration_seconds != null ? fmtDuration(s.duration_seconds) : '—'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Stack>
        )}
      </DialogContent>
    </Dialog>
  )
}

interface BarProps {
  sessions: PlayerSessionOut[]
  instance: InstanceOut
  highlight: number | null
  onHover: (idx: number | null) => void
}

function PlayerTimelineBar({ sessions, instance, highlight, onHover }: BarProps) {
  const { instStart, instEnd, bars } = useMemo(() => {
    const instStart = new Date(instance.opened_at).getTime()
    const instEnd = instance.closed_at ? new Date(instance.closed_at).getTime() : Date.now()
    const total = instEnd - instStart
    const VW = 1000
    const toX = (ts: string) => ((new Date(ts).getTime() - instStart) / total) * VW
    const bars = sessions.map((s, i) => {
      const x1 = toX(s.join_ts)
      const x2 = s.leave_ts ? toX(s.leave_ts) : VW
      return { i, x1, w: Math.max(3, x2 - x1) }
    })
    return { instStart, instEnd, bars }
  }, [sessions, instance])

  const fmt = (ts: number | string) =>
    new Date(ts).toLocaleString('ja-JP', {
      month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
    })

  const BAR_H = 26
  return (
    <Box>
      <Box component="svg" viewBox={`0 0 1000 ${BAR_H}`} preserveAspectRatio="none"
        className="w-full block" sx={{ height: BAR_H }}>
        <rect x={0} y={0} width={1000} height={BAR_H} fill="#dee2e6" rx={3} />
        {bars.map((b) => (
          <rect
            key={b.i}
            x={b.x1}
            y={0}
            width={b.w}
            height={BAR_H}
            rx={2}
            fill={highlight === b.i ? 'rgba(13,110,253,0.92)' : 'rgba(13,110,253,0.55)'}
            stroke={highlight === b.i ? 'rgba(13,110,253,1)' : 'none'}
            onMouseEnter={() => onHover(b.i)}
            onMouseLeave={() => onHover(null)}
            style={{ cursor: 'pointer' }}
          />
        ))}
      </Box>
      <Box className="flex justify-between text-xs text-neutral-500 mt-1">
        <span>{fmt(instStart)}</span>
        <span>{fmt(instEnd)}</span>
      </Box>
    </Box>
  )
}
