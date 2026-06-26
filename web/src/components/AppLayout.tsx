import { AppBar, Box, Divider, Toolbar, Typography } from "@mui/material";
import { Outlet, Link, useLocation } from "react-router-dom";

const NAV: { label: string; to: string; match: (pathname: string) => boolean }[] = [
  {
    label: "インスタンス",
    to: "/",
    match: (p) => p === "/" || p.startsWith("/instances"),
  },
  {
    label: "プレイヤー",
    to: "/players",
    match: (p) => p.startsWith("/players"),
  },
  {
    label: "違反検知",
    to: "/violations",
    match: (p) => p.startsWith("/violations") || p.startsWith("/compare"),
  },
];

export default function AppLayout() {
  const { pathname } = useLocation();

  return (
    <Box className="flex flex-col h-full">
      <AppBar
        position="static"
        color="default"
        elevation={0}
        className="bg-neutral-900! text-white!"
      >
        <Toolbar variant="dense" className="gap-3">
          <Typography
            component={Link}
            to="/"
            variant="h6"
            className="text-inherit! no-underline! font-medium!"
          >
            VRChat Join Manager
          </Typography>
          <Divider
            orientation="vertical"
            flexItem
            className="my-2!"
            sx={{ borderColor: "rgba(255,255,255,0.2)" }}
          />
          <Box component="nav" className="flex items-stretch self-stretch">
            {NAV.map((item) => {
              const active = item.match(pathname);
              return (
                <Typography
                  key={item.to}
                  component={Link}
                  to={item.to}
                  variant="body2"
                  className={`flex items-center px-3 text-inherit! no-underline! border-b-2 transition-opacity ${
                    active
                      ? "opacity-100 font-medium! border-white"
                      : "opacity-70 hover:opacity-100 border-transparent"
                  }`}
                >
                  {item.label}
                </Typography>
              );
            })}
          </Box>
        </Toolbar>
      </AppBar>
      <Box className="flex-1 min-h-0 overflow-hidden">
        <Outlet />
      </Box>
    </Box>
  );
}
