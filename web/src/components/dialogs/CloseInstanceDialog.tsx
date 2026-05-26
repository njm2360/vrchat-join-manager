import { useEffect, useState } from 'react'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import dayjs, { type Dayjs } from 'dayjs'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSnackbar } from 'notistack'
import { api } from '../../api/client'
import type { InstanceOut } from '../../api/schemas'
import { extractInstanceNumber } from '../../utils/format'

interface Props {
  open: boolean
  onClose: () => void
  instance: InstanceOut
}

export default function CloseInstanceDialog({ open, onClose, instance }: Props) {
  const expected = extractInstanceNumber(instance.location_id)
  const [confirmText, setConfirmText] = useState('')
  const [at, setAt] = useState<Dayjs | null>(dayjs())
  const qc = useQueryClient()
  const { enqueueSnackbar } = useSnackbar()

  useEffect(() => {
    if (open) {
      setConfirmText('')
      setAt(dayjs())
    }
  }, [open])

  const closeMut = useMutation({
    mutationFn: async () => {
      if (!at) throw new Error('クローズ時刻を入力してください')
      const { error } = await api.POST('/api/locations/{location_id}/close', {
        params: { path: { location_id: instance.location_id } },
        body: { at: at.toISOString() },
      })
      if (error) throw new Error('クローズに失敗しました')
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['instance', instance.id] })
      qc.invalidateQueries({ queryKey: ['instances'] })
      enqueueSnackbar('インスタンスをクローズしました', { variant: 'success' })
      onClose()
    },
    onError: (e: Error) => enqueueSnackbar(e.message, { variant: 'error' }),
  })

  const canConfirm = !!expected && confirmText.trim() === expected && !closeMut.isPending

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>インスタンスのクローズ</DialogTitle>
      <DialogContent>
        <Stack spacing={2} className="mt-1">
          <Typography variant="body2">
            進行中のままになっているインスタンスを手動でクローズします。在室中のセッションは指定時刻で退室扱いになります。
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {instance.world_name || instance.world_id} / {instance.location_id}
          </Typography>
          <DateTimePicker
            label="クローズ時刻"
            value={at}
            onChange={setAt}
            slotProps={{ textField: { size: 'small' } }}
          />
          <TextField
            size="small"
            label={`確認のため、インスタンス番号 ${expected} を入力してください`}
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && canConfirm) closeMut.mutate()
            }}
            slotProps={{ htmlInput: { inputMode: 'numeric', autoComplete: 'off' } }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>キャンセル</Button>
        <Button
          variant="contained"
          color="warning"
          disabled={!canConfirm}
          onClick={() => closeMut.mutate()}
        >
          {closeMut.isPending ? 'クローズ中...' : 'クローズ'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
