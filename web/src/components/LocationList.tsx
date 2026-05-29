import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Box,
  Button,
  Chip,
  List,
  ListItemButton,
  Stack,
  Typography,
} from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import type { Dayjs } from 'dayjs'
import { api } from '@/api/client'
import type { InstanceOut } from '@/api/schemas'
import { fmtDate, extractInstanceNumber } from '@/utils/format'

type Range = { start: Dayjs | null; end: Dayjs | null }

interface Props {
  selectedId: number | null
  onSelect: (instance: InstanceOut) => void
}

export default function LocationList({ selectedId, onSelect }: Props) {
  const [draft, setDraft] = useState<Range>({ start: null, end: null })
  const [applied, setApplied] = useState<Range>({ start: null, end: null })

  const { data, isLoading } = useQuery({
    queryKey: ['instances', applied.start?.toISOString() ?? null, applied.end?.toISOString() ?? null],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/instances', {
        params: {
          query: {
            start: applied.start?.toISOString(),
            end: applied.end?.toISOString(),
          },
        },
      })
      if (error) throw new Error('failed to load instances')
      return data ?? []
    },
  })

  return (
    <Box className="flex flex-col h-full bg-neutral-50 border-r border-neutral-200">
      <Stack spacing={1.5} className="p-3 border-b border-neutral-200">
        <Stack direction="row" spacing={1}>
          <DateTimePicker
            label="開始"
            value={draft.start}
            onChange={(v) => setDraft((d) => ({ ...d, start: v }))}
            slotProps={{ textField: { size: 'small', fullWidth: true } }}
          />
          <DateTimePicker
            label="終了"
            value={draft.end}
            onChange={(v) => setDraft((d) => ({ ...d, end: v }))}
            slotProps={{ textField: { size: 'small', fullWidth: true } }}
          />
        </Stack>
        <Button
          variant="contained"
          color="inherit"
          size="small"
          onClick={() => setApplied(draft)}
        >
          絞り込み
        </Button>
      </Stack>

      <Box className="flex-1 min-h-0 overflow-y-auto">
        {isLoading ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            読み込み中...
          </Typography>
        ) : (data?.length ?? 0) === 0 ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            該当なし
          </Typography>
        ) : (
          <List disablePadding>
            {data!.map((inst) => (
              <LocationItem
                key={inst.id}
                inst={inst}
                selected={inst.id === selectedId}
                onClick={() => onSelect(inst)}
              />
            ))}
          </List>
        )}
      </Box>
    </Box>
  )
}

interface ItemProps {
  inst: InstanceOut
  selected: boolean
  onClick: () => void
}

function LocationItem({ inst, selected, onClick }: ItemProps) {
  const ongoing = !inst.closed_at
  const rangeLabel = ongoing
    ? `${fmtDate(inst.opened_at)} 〜`
    : `${fmtDate(inst.opened_at)} 〜 ${fmtDate(inst.closed_at!)}`

  return (
    <ListItemButton selected={selected} onClick={onClick} className="block! py-2!">
      <Typography variant="body2" color="text.secondary" className="block font-mono">
        {extractInstanceNumber(inst.location_id) || '—'}
      </Typography>
      <Typography variant="caption" color="text.secondary" className="block truncate">
        {inst.location_id}
      </Typography>
      <Stack direction="row" spacing={0.5} className="mt-1" sx={{ alignItems: 'center' }}>
        <Chip
          size="small"
          label={rangeLabel}
          color={ongoing ? 'success' : 'default'}
          variant="filled"
        />
        {ongoing && inst.user_count > 0 && (
          <Chip size="small" label={`${inst.user_count}人`} color="warning" />
        )}
      </Stack>
    </ListItemButton>
  )
}

