import { useState } from 'react'
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Stack,
  Tab,
  Tabs,
} from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import type { InstanceOut } from '../api/schemas'
import TimelineTab from './tabs/TimelineTab'
import EventsTab from './tabs/EventsTab'
import SessionsTab from './tabs/SessionsTab'
import PlayersTab from './tabs/PlayersTab'
import VisitorsTab from './tabs/VisitorsTab'
import CompareInstanceDialog from './dialogs/CompareInstanceDialog'
import CloseInstanceDialog from './dialogs/CloseInstanceDialog'
import DeleteInstanceDialog from './dialogs/DeleteInstanceDialog'

type TabKey = 'timeline' | 'events' | 'sessions' | 'players' | 'visitors'

interface Props {
  instanceId: number
  instance: InstanceOut | null
  onBack: () => void
  isMobile: boolean
}

export default function InstanceDetail({ instanceId, instance, onBack, isMobile }: Props) {
  const [tab, setTab] = useState<TabKey>('timeline')
  const [compareOpen, setCompareOpen] = useState(false)
  const [closeOpen, setCloseOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  return (
    <Box className="h-full overflow-auto p-3">
      <Card className="h-full flex flex-col">
        <CardHeader
          className="border-b border-neutral-200"
          title={
            <Stack spacing={1}>
              {isMobile && (
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<ArrowBackIcon />}
                  onClick={onBack}
                  className="self-start"
                >
                  一覧に戻る
                </Button>
              )}
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <Tabs
                  value={tab}
                  onChange={(_, v: TabKey) => setTab(v)}
                  variant="scrollable"
                  scrollButtons="auto"
                  className="flex-1 min-w-0"
                >
                  <Tab value="timeline" label="人数推移" />
                  <Tab value="events" label="入退場ログ" />
                  <Tab value="sessions" label="セッション一覧" />
                  <Tab value="players" label="在室中" />
                  <Tab value="visitors" label="訪れた人" />
                </Tabs>
                {instance && !instance.closed_at && (
                  <Button
                    size="small"
                    variant="contained"
                    color="warning"
                    onClick={() => setCloseOpen(true)}
                  >
                    クローズ
                  </Button>
                )}
                {instance && (
                  <Button
                    size="small"
                    variant="contained"
                    color="error"
                    onClick={() => setDeleteOpen(true)}
                  >
                    削除
                  </Button>
                )}
              </Stack>
            </Stack>
          }
        />
        <CardContent className="flex-1 min-h-0 overflow-auto">
          {tab === 'timeline' && (
            <TimelineTab
              instanceId={instanceId}
              instance={instance}
              onCompare={() => setCompareOpen(true)}
            />
          )}
          {tab === 'events' && <EventsTab instanceId={instanceId} />}
          {tab === 'sessions' && <SessionsTab instanceId={instanceId} />}
          {tab === 'players' && <PlayersTab instanceId={instanceId} />}
          {tab === 'visitors' && <VisitorsTab instanceId={instanceId} />}
        </CardContent>
      </Card>

      {instance && (
        <CompareInstanceDialog
          open={compareOpen}
          onClose={() => setCompareOpen(false)}
          current={instance}
        />
      )}
      {instance && (
        <CloseInstanceDialog
          open={closeOpen}
          onClose={() => setCloseOpen(false)}
          instance={instance}
        />
      )}
      {instance && (
        <DeleteInstanceDialog
          open={deleteOpen}
          onClose={() => setDeleteOpen(false)}
          instance={instance}
          onDeleted={onBack}
        />
      )}
    </Box>
  )
}
