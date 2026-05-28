import { Chip, Stack, Typography } from '@mui/material'
import type { InstanceOut } from '../api/schemas'
import { fmtDate, extractInstanceNumber } from '../utils/format'

function accessLabel(inst: InstanceOut): string {
  if (inst.group_id) {
    const t = inst.group_access_type
    if (t === 'public') return 'Group Public'
    if (t === 'plus') return 'Group+'
    return 'Group'
  }
  if (inst.private) return 'Invite'
  if (inst.hidden) return 'Friends+'
  if (inst.friends) return 'Friends'
  return 'Public'
}

interface Props {
  instance: InstanceOut
  dense?: boolean
}

export default function InstanceInfo({ instance, dense }: Props) {
  const ongoing = !instance.closed_at
  const rangeLabel = ongoing
    ? `${fmtDate(instance.opened_at)} 〜`
    : `${fmtDate(instance.opened_at)} 〜 ${fmtDate(instance.closed_at!)}`
  const instanceNumber =
    instance.instance_id || extractInstanceNumber(instance.location_id)

  return (
    <Stack spacing={0.75}>
      <Stack
        direction="row"
        spacing={1.5}
        useFlexGap
        sx={{ alignItems: 'baseline', flexWrap: 'wrap' }}
      >
        <Typography
          variant={dense ? 'subtitle1' : 'h6'}
          className="font-mono"
          sx={{ fontWeight: 700, lineHeight: 1.2 }}
        >
          {instanceNumber || '—'}
        </Typography>
        {instance.group_id && (
          <Typography
            variant="body2"
            className="font-mono"
            sx={{ fontWeight: 600, wordBreak: 'break-all' }}
          >
            {instance.group_id}
            {instance.group_name ? ` (${instance.group_name})` : ''}
          </Typography>
        )}
      </Stack>
      <Stack
        direction="row"
        spacing={0.5}
        useFlexGap
        sx={{ alignItems: 'center', flexWrap: 'wrap' }}
      >
        <Chip
          size="small"
          label={rangeLabel}
          color={ongoing ? 'success' : 'default'}
        />
        {instance.region && (
          <Chip
            size="small"
            variant="outlined"
            label={instance.region.toUpperCase()}
          />
        )}
        <Chip size="small" variant="outlined" label={accessLabel(instance)} />
        {ongoing && instance.user_count > 0 && (
          <Chip
            size="small"
            color="warning"
            label={`${instance.user_count}人`}
          />
        )}
      </Stack>
      <Typography
        variant="caption"
        color="text.secondary"
        className="font-mono"
        title={instance.location_id}
        sx={{
          whiteSpace: 'nowrap',
          overflowX: 'auto',
          display: 'block',
        }}
      >
        {instance.location_id}
      </Typography>
    </Stack>
  )
}
