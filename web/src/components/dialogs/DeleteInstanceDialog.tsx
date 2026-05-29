import { useRef, useState } from 'react'
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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSnackbar } from 'notistack'
import { api } from '@/api/client'
import type { InstanceOut } from '@/api/schemas'
import { extractInstanceNumber } from '@/utils/format'

interface Props {
  open: boolean
  onClose: () => void
  instance: InstanceOut
  onDeleted: () => void
}

export default function DeleteInstanceDialog({ open, onClose, instance, onDeleted }: Props) {
  const expected = extractInstanceNumber(instance.location_id)
  const [confirmText, setConfirmText] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const qc = useQueryClient()
  const { enqueueSnackbar } = useSnackbar()

  const deleteMut = useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/api/instances/{instance_id}', {
        params: { path: { instance_id: instance.id } },
      })
      if (error) throw new Error('削除に失敗しました')
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['instances'] })
      enqueueSnackbar('インスタンスを削除しました', { variant: 'success' })
      onClose()
      onDeleted()
    },
    onError: (e: Error) => enqueueSnackbar(e.message, { variant: 'error' }),
  })

  const canConfirm = !!expected && confirmText.trim() === expected && !deleteMut.isPending

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      slotProps={{
        transition: {
          onEnter: () => setConfirmText(''),
          onEntered: () => inputRef.current?.focus(),
        },
      }}
    >
      <DialogTitle>インスタンスの削除</DialogTitle>
      <DialogContent>
        <Stack spacing={2} className="mt-1">
          <Typography variant="body2">
            この操作は取り消せません。インスタンスとそれに紐づくセッション・イベントなどがすべて削除されます。
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {instance.world_name || instance.world_id} / {instance.location_id}
          </Typography>
          <TextField
            inputRef={inputRef}
            size="small"
            label={`確認のため、インスタンス番号 ${expected} を入力してください`}
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && canConfirm) deleteMut.mutate()
            }}
            slotProps={{ htmlInput: { inputMode: 'numeric', autoComplete: 'off' } }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>キャンセル</Button>
        <Button
          variant="contained"
          color="error"
          disabled={!canConfirm}
          onClick={() => deleteMut.mutate()}
        >
          {deleteMut.isPending ? '削除中...' : '削除'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
