import { AppBar, Box, Toolbar, Typography } from '@mui/material'
import { Outlet, Link, useLocation } from 'react-router-dom'

export default function AppLayout() {
  const { pathname } = useLocation()
  const isHome = pathname === '/' || pathname.startsWith('/instances')

  return (
    <Box className="flex flex-col h-full">
      <AppBar position="static" color="default" elevation={0} className="bg-neutral-900! text-white!">
        <Toolbar variant="dense" className="gap-3">
          <Typography
            component={Link}
            to="/"
            variant="h6"
            className="text-inherit! no-underline! font-medium!"
          >
            VRChat Join Manager
          </Typography>
          {!isHome && (
            <Typography
              component={Link}
              to="/"
              variant="body2"
              className="ml-auto! text-inherit! no-underline! opacity-80"
            >
              ← 一覧に戻る
            </Typography>
          )}
        </Toolbar>
      </AppBar>
      <Box className="flex-1 min-h-0 overflow-hidden">
        <Outlet />
      </Box>
    </Box>
  )
}
