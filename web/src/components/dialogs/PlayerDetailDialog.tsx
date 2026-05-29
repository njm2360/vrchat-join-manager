import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Box,
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Paper,
  Typography,
} from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import EditIcon from '@mui/icons-material/Edit'
import OpenInNewIcon from '@mui/icons-material/OpenInNew'
import LaunchIcon from '@mui/icons-material/Launch'
import { useSnackbar } from 'notistack'
import { useInstance, usePlayerDetail, usePlayerSessions, useSetPlayerDiscord } from '../../api/queries'
import type { InstanceOut, PlayerDetailOut, PlayerSessionOut } from '../../api/schemas'
import { fmtDate, fmtDateFull, fmtDuration } from '../../utils/format'
import { copyText } from '../../utils/clipboard'
import { LeaveCell } from '../tabs/SessionsTab'

export interface PlayerDetailCtx {
  userId: string
  displayName: string
  instanceId?: number
}

interface Props {
  open: boolean
  onClose: () => void
  ctx: PlayerDetailCtx
}

type TabKey = 'overview' | 'instance'

export default function PlayerDetailDialog({ open, onClose, ctx }: Props) {
  const { userId, displayName: fallbackName, instanceId } = ctx
  const hasInstance = instanceId != null
  const [tab, setTab] = useState<TabKey>(hasInstance ? 'instance' : 'overview')
  const { enqueueSnackbar } = useSnackbar()

  const { data: detail } = usePlayerDetail(userId, { enabled: open })
  const { data: instance } = useInstance(instanceId ?? null, { enabled: hasInstance && open })

  const displayName = detail?.display_name ?? fallbackName
  const discordId = detail?.discord_id ?? null

  const copy = async (label: string, value: string) => {
    try {
      await copyText(value)
      enqueueSnackbar(`${label}をコピーしました`, { variant: 'success' })
    } catch {
      enqueueSnackbar('クリップボードへのコピーに失敗しました', { variant: 'error' })
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth scroll="paper">
      <DialogTitle className="flex items-start gap-2">
        <Box className="flex-1 min-w-0">
          <Typography variant="h6" className="font-medium truncate">
            {displayName}
          </Typography>
          <Stack spacing={0.5} className="mt-1">
            <IdRow
              label="ユーザーID"
              value={userId}
              onCopy={() => copy('ユーザーID', userId)}
              externalHref={`https://vrchat.com/home/user/${encodeURIComponent(userId)}`}
              externalTitle="VRChat のプロフィールを開く"
            />
            <IdRow
              label="Discord ID"
              value={discordId}
              onCopy={discordId ? () => copy('Discord ID', discordId) : undefined}
              edit={{
                userId,
                onSaved: (next) =>
                  enqueueSnackbar(
                    next ? 'Discord IDを更新しました' : 'Discord IDを削除しました',
                    { variant: 'success' },
                  ),
                onError: (msg) => enqueueSnackbar(msg, { variant: 'error' }),
              }}
            />
          </Stack>
        </Box>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </DialogTitle>

      <Box className="px-3 border-b border-neutral-200">
        <Tabs value={tab} onChange={(_, v: TabKey) => setTab(v)}>
          <Tab value="overview" label="概要" />
          {hasInstance && <Tab value="instance" label="このインスタンス" />}
        </Tabs>
      </Box>

      <DialogContent dividers className="p-0!">
        {tab === 'overview' && <OverviewTab userId={userId} detail={detail ?? null} />}
        {tab === 'instance' && hasInstance && (
          <InstanceTab userId={userId} instance={instance ?? null} />
        )}
      </DialogContent>
    </Dialog>
  )
}

// ── ID 行 (ラベル + 値 + コピー/外部リンクボタン) ────────────────

interface EditOptions {
  userId: string
  onSaved?: (next: string | null) => void
  onError?: (message: string) => void
}

interface IdRowProps {
  label: string
  value: string | null | undefined
  onCopy?: () => void
  externalHref?: string
  externalTitle?: string
  edit?: EditOptions
}

function IdRow({ label, value, onCopy, externalHref, externalTitle, edit }: IdRowProps) {
  const hasValue = !!value
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const mutation = useSetPlayerDiscord(edit?.userId ?? '')

  useEffect(() => {
    if (editing) {
      setDraft(value ?? '')
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [editing, value])

  const save = () => {
    if (!edit) return
    const trimmed = draft.trim()
    const next = trimmed === '' ? null : trimmed
    if ((value ?? null) === next) {
      setEditing(false)
      return
    }
    mutation.mutate(next, {
      onSuccess: () => {
        edit.onSaved?.(next)
        setEditing(false)
      },
      onError: (e) => edit.onError?.((e as Error).message),
    })
  }

  return (
    <Stack
      direction="row"
      spacing={1}
      useFlexGap
      sx={{ alignItems: 'center', flexWrap: 'wrap' }}
    >
      <Typography
        variant="caption"
        color="text.secondary"
        className="min-w-[80px] shrink-0"
      >
        {label}
      </Typography>

      {editing ? (
        <>
          <TextField
            size="small"
            value={draft}
            inputRef={inputRef}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                save()
              } else if (e.key === 'Escape') {
                e.preventDefault()
                setEditing(false)
              }
            }}
            placeholder="空欄で削除"
            disabled={mutation.isPending}
            slotProps={{
              htmlInput: {
                autoComplete: 'off',
                style: { fontFamily: 'ui-monospace, monospace', fontSize: '0.85rem', padding: '4px 8px' },
              },
            }}
          />
          <IconButton size="small" onClick={save} disabled={mutation.isPending} title="保存 (Enter)">
            <CheckIcon fontSize="inherit" />
          </IconButton>
          <IconButton
            size="small"
            onClick={() => setEditing(false)}
            disabled={mutation.isPending}
            title="キャンセル (Esc)"
          >
            <CloseIcon fontSize="inherit" />
          </IconButton>
        </>
      ) : (
        <>
          {hasValue ? (
            <Typography variant="caption" className="font-mono break-all">
              {value}
            </Typography>
          ) : (
            <Typography variant="caption" color="text.disabled">
              未登録
            </Typography>
          )}
          {hasValue && onCopy && (
            <IconButton size="small" onClick={onCopy} title={`${label}をコピー`}>
              <ContentCopyIcon fontSize="inherit" />
            </IconButton>
          )}
          {hasValue && externalHref && (
            <IconButton
              size="small"
              component="a"
              href={externalHref}
              target="_blank"
              rel="noopener noreferrer"
              title={externalTitle ?? '外部リンクを開く'}
            >
              <LaunchIcon fontSize="inherit" />
            </IconButton>
          )}
          {edit && (
            <IconButton size="small" onClick={() => setEditing(true)} title={`${label}を編集`}>
              <EditIcon fontSize="inherit" />
            </IconButton>
          )}
        </>
      )}
    </Stack>
  )
}

// ── 概要タブ ──────────────────────────────────────────────────────

function OverviewTab({ userId, detail }: { userId: string; detail: PlayerDetailOut | null }) {
  // 直近セッションは別途取得 (limit 10)
  const { data: recent = [], isLoading: recentLoading } = usePlayerSessions(
    userId,
    { order: 'desc', limit: 10 },
  )

  if (!detail) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        読み込み中...
      </Typography>
    )
  }

  return (
    <Stack spacing={2.5} className="p-3">
      {/* サマリーカード */}
      <Box className="grid grid-cols-2 md:grid-cols-3 gap-2">
        <StatCard label="通算訪問" value={`${detail.total_visits}回`} />
        <StatCard label="通算滞在" value={fmtDuration(detail.total_duration_seconds)} />
        <StatCard
          label="現在の状態"
          value={detail.in_room ? '在室中' : '未在室'}
          accent={detail.in_room ? 'success' : undefined}
        />
        <StatCard
          label="初回訪問"
          value={detail.first_seen ? fmtDate(detail.first_seen) : '—'}
          small
        />
        <StatCard
          label="最終訪問"
          value={detail.last_seen ? fmtDate(detail.last_seen) : '—'}
          small
        />
      </Box>

      {/* 直近セッション */}
      <Box>
        <Typography variant="subtitle2" color="text.secondary" className="mb-1!">
          直近のセッション (最大10件)
        </Typography>
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
              {recentLoading ? (
                <TableRow>
                  <TableCell colSpan={3} align="center">
                    <Typography variant="body2" color="text.secondary">読み込み中...</Typography>
                  </TableCell>
                </TableRow>
              ) : recent.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} align="center">
                    <Typography variant="body2" color="text.secondary">セッション履歴なし</Typography>
                  </TableCell>
                </TableRow>
              ) : (
                recent.map((s) => (
                  <TableRow key={s.id} hover>
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
      </Box>

      <Box className="flex justify-end">
        <Button
          variant="outlined"
          size="small"
          endIcon={<OpenInNewIcon />}
          component="a"
          href={`/players/${encodeURIComponent(userId)}`}
          target="_blank"
        >
          月別カレンダーを開く
        </Button>
      </Box>
    </Stack>
  )
}

function StatCard({
  label,
  value,
  small,
  accent,
}: {
  label: string
  value: string
  small?: boolean
  accent?: 'success'
}) {
  return (
    <Box className="border border-neutral-200 rounded-md p-2 bg-white">
      <Typography variant="caption" color="text.secondary" className="block">
        {label}
      </Typography>
      <Typography
        variant={small ? 'body2' : 'subtitle1'}
        className={`font-semibold ${accent === 'success' ? 'text-green-600' : ''}`}
      >
        {value}
      </Typography>
    </Box>
  )
}

// ── このインスタンスタブ (旧 SessionDialog の中身) ────────────────

function InstanceTab({ userId, instance }: { userId: string; instance: InstanceOut | null }) {
  const [highlight, setHighlight] = useState<number | null>(null)

  const { data: sessions = [], isLoading } = usePlayerSessions(
    userId,
    { instance_id: instance?.id, order: 'asc' },
    { enabled: !!instance },
  )

  if (!instance) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        インスタンス情報を読み込み中...
      </Typography>
    )
  }

  if (isLoading) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        読み込み中...
      </Typography>
    )
  }

  if (sessions.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" className="p-3 text-center">
        このインスタンスでのセッションなし
      </Typography>
    )
  }

  return (
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
  )
}

interface BarProps {
  sessions: PlayerSessionOut[]
  instance: InstanceOut
  highlight: number | null
  onHover: (idx: number | null) => void
}

function PlayerTimelineBar({ sessions, instance, highlight, onHover }: BarProps) {
  const [nowMs] = useState(() => Date.now())
  const { instStart, instEnd, bars } = useMemo(() => {
    const instStart = new Date(instance.opened_at).getTime()
    const instEnd = instance.closed_at ? new Date(instance.closed_at).getTime() : nowMs
    const total = Math.max(1, instEnd - instStart)
    const VW = 1000
    const toX = (ts: string) => ((new Date(ts).getTime() - instStart) / total) * VW
    const bars = sessions.map((s, i) => {
      const x1 = toX(s.join_ts)
      const x2 = s.leave_ts ? toX(s.leave_ts) : VW
      return { i, x1, w: Math.max(3, x2 - x1) }
    })
    return { instStart, instEnd, bars }
  }, [sessions, instance, nowMs])

  const fmt = (ts: number | string) =>
    new Date(ts).toLocaleString('ja-JP', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })

  const BAR_H = 26
  return (
    <Box>
      <Box
        component="svg"
        viewBox={`0 0 1000 ${BAR_H}`}
        preserveAspectRatio="none"
        className="w-full block"
        sx={{ height: BAR_H }}
      >
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
