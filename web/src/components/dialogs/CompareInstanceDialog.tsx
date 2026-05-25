import { useQuery } from '@tanstack/react-query'
import {
  Box,
  Chip,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItemButton,
  Stack,
  Typography,
} from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import { api } from '../../api/client'
import type { InstanceOut } from '../../api/schemas'
import { fmtDate } from '../../utils/format'

interface Props {
  open: boolean
  onClose: () => void
  current: InstanceOut
}

export default function CompareInstanceDialog({ open, onClose, current }: Props) {
  const { data: instances = [], isLoading } = useQuery<InstanceOut[]>({
    queryKey: ['instances', 'all'],
    enabled: open,
    queryFn: async () => {
      const { data, error } = await api.GET('/api/instances', {})
      if (error) throw new Error('failed to load instances')
      return data ?? []
    },
  })

  const curStart = new Date(current.opened_at).getTime()
  const curEnd = current.closed_at ? new Date(current.closed_at).getTime() : Date.now()
  const overlapping = instances.filter((inst) => {
    if (inst.id === current.id) return false
    const s = new Date(inst.opened_at).getTime()
    const e = inst.closed_at ? new Date(inst.closed_at).getTime() : Date.now()
    return curStart < e && s < curEnd
  })

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth scroll="paper">
      <DialogTitle className="flex items-center">
        <Box className="flex-1">比較するインスタンスを選択</Box>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers className="p-0!">
        {isLoading ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            読み込み中...
          </Typography>
        ) : overlapping.length === 0 ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            比較できる他のインスタンスがありません
          </Typography>
        ) : (
          <List disablePadding>
            {overlapping.map((inst) => {
              const range = inst.closed_at
                ? `${fmtDate(inst.opened_at)} 〜 ${fmtDate(inst.closed_at)}`
                : `${fmtDate(inst.opened_at)} 〜`
              return (
                <ListItemButton
                  key={inst.id}
                  className="block!"
                  onClick={() => {
                    onClose()
                    window.open(`/compare/${current.id}/${inst.id}`, '_blank')
                  }}
                >
                  <Typography variant="body2" className="font-medium truncate">
                    {inst.world_name || inst.world_id}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" className="block truncate">
                    {inst.location_id}
                  </Typography>
                  <Stack direction="row" className="mt-1">
                    <Chip
                      size="small"
                      label={range}
                      color={inst.closed_at ? 'default' : 'success'}
                    />
                  </Stack>
                </ListItemButton>
              )
            })}
          </List>
        )}
      </DialogContent>
    </Dialog>
  )
}
