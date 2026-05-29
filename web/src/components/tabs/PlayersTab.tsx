import { useMemo } from 'react'
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
import { useSnackbar } from 'notistack'
import { usePlayers } from '@/api/queries'
import { fmtDateFull } from '@/utils/format'
import { copyText } from '@/utils/clipboard'
import PlayerLink from '@/components/PlayerLink'
import { useSortState } from '@/hooks/useSortState'

interface Props {
  instanceId: number
}

type SortKey = 'internal_id' | 'display_name' | 'join_ts'

const COLUMNS: { key: SortKey; label: string; width?: number; align?: 'right' }[] = [
  { key: 'internal_id', label: 'ID', width: 64, align: 'right' },
  { key: 'display_name', label: '名前' },
]

export default function PlayersTab({ instanceId }: Props) {
  const { sortBy, order, toggleSort } = useSortState<SortKey>('internal_id', 'asc', 'asc')
  const { enqueueSnackbar } = useSnackbar()

  const { data: players = [], refetch } = usePlayers(instanceId, {
    sort_by: sortBy,
    order,
  })

  const mentionText = useMemo(
    () =>
      players
        .filter((p) => p.discord_id)
        .map((p) => `@${p.discord_id}`)
        .join(' ') + ' ',
    [players],
  )

  const copyDiscord = async () => {
    const count = players.filter((p) => p.discord_id).length
    if (count === 0) {
      enqueueSnackbar('Discord IDが登録されているプレイヤーがいません', { variant: 'info' })
      return
    }
    try {
      await copyText(mentionText)
      enqueueSnackbar(`${count}人分のDiscord IDをコピーしました`, { variant: 'success' })
    } catch {
      enqueueSnackbar('クリップボードへのコピーに失敗しました', { variant: 'error' })
    }
  }

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Button variant="contained" size="small" onClick={() => refetch()}>
          更新
        </Button>
        <Button variant="outlined" size="small" onClick={copyDiscord}>
          全員のDiscord IDをコピー
        </Button>
        <Typography variant="h6" className="font-bold">
          {players.length} 人
        </Typography>
      </Stack>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {COLUMNS.map((c) => (
                <TableCell key={c.key} width={c.width} align={c.align}>
                  <TableSortLabel
                    active={sortBy === c.key}
                    direction={sortBy === c.key ? order : 'asc'}
                    onClick={() => toggleSort(c.key)}
                  >
                    {c.label}
                  </TableSortLabel>
                </TableCell>
              ))}
              <TableCell width={140}>Discord ID</TableCell>
              <TableCell width={160}>
                <TableSortLabel
                  active={sortBy === 'join_ts'}
                  direction={sortBy === 'join_ts' ? order : 'asc'}
                  onClick={() => toggleSort('join_ts')}
                >
                  入室日時
                </TableSortLabel>
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {players.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} align="center">
                  <Typography variant="body2" color="text.secondary">
                    在室中のプレイヤーなし
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              players.map((p) => (
                <TableRow key={p.user_id} hover>
                  <TableCell align="right" className="text-neutral-500">
                    {p.internal_id ?? '—'}
                  </TableCell>
                  <TableCell>
                    <PlayerLink userId={p.user_id} displayName={p.display_name} instanceId={instanceId} />
                  </TableCell>
                  <TableCell>
                    {p.discord_id ? (
                      p.discord_id
                    ) : (
                      <Box component="span" className="text-neutral-400">
                        未登録
                      </Box>
                    )}
                  </TableCell>
                  <TableCell>{fmtDateFull(p.join_ts)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}
