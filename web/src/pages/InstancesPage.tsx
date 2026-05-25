import { useEffect, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Box, Typography, useMediaQuery, useTheme } from '@mui/material'
import LocationList from '../components/LocationList'
import InstanceDetail from '../components/InstanceDetail'
import { useInstance } from '../api/queries'

export default function InstancesPage() {
  const { instanceId: idStr } = useParams<{ instanceId?: string }>()
  const navigate = useNavigate()
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))

  const instanceId = useMemo(() => {
    const n = Number(idStr)
    return Number.isFinite(n) && n > 0 ? n : null
  }, [idStr])

  const { data: instance } = useInstance(instanceId)

  useEffect(() => {
    if (idStr && !instanceId) {
      navigate('/', { replace: true })
    }
  }, [idStr, instanceId, navigate])

  const showSidebar = !isMobile || instanceId == null
  const showDetail = !isMobile || instanceId != null

  return (
    <Box className="flex h-full">
      {showSidebar && (
        <Box className="w-full md:w-[320px] lg:w-[360px] md:shrink-0 h-full">
          <LocationList
            selectedId={instanceId}
            onSelect={(inst) => navigate(`/instances/${inst.id}`)}
          />
        </Box>
      )}
      {showDetail && (
        <Box className="flex-1 min-w-0 h-full">
          {instanceId == null ? (
            <Box className="h-full flex items-center justify-center p-4">
              <Typography color="text.secondary">
                左のリストからロケーションを選択してください
              </Typography>
            </Box>
          ) : (
            <InstanceDetail
              instanceId={instanceId}
              instance={instance ?? null}
              onBack={() => navigate('/')}
              isMobile={isMobile}
            />
          )}
        </Box>
      )}
    </Box>
  )
}
