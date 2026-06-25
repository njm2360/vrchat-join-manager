import { useEffect, useMemo } from "react";
import { useParams, useSearchParams, Link } from "react-router-dom";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import ChevronLeftIcon from "@mui/icons-material/ChevronLeft";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import SessionCalendar from "@/components/SessionCalendar";
import { usePlayerDetail, usePlayerSessions } from "@/api/queries";

export default function PlayerPage() {
  const { userId = "" } = useParams<{ userId: string }>();
  const [params, setParams] = useSearchParams();
  const worldId = params.get("world_id") || "";
  const { data: player } = usePlayerDetail(userId);
  const displayName = player?.display_name || userId;

  const now = useMemo(() => new Date(), []);
  const yearParam = Number(params.get("year"));
  const monthParam = Number(params.get("month"));
  const year = Number.isInteger(yearParam) && yearParam >= 1970 ? yearParam : now.getFullYear();
  const month =
    Number.isInteger(monthParam) && monthParam >= 1 && monthParam <= 12
      ? monthParam - 1
      : now.getMonth();

  const setYM = (next: { year: number; month: number }) => {
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set("year", String(next.year));
        p.set("month", String(next.month + 1));
        return p;
      },
      { replace: true },
    );
  };

  useEffect(() => {
    if (params.get("year") && params.get("month")) return;
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (!p.get("year")) p.set("year", String(year));
        if (!p.get("month")) p.set("month", String(month + 1));
        return p;
      },
      { replace: true },
    );
  }, []);

  // 前月末日から翌月1日まで取得 (月をまたぐセッションも拾う)
  const start = new Date(year, month, 0).toISOString();
  const end = new Date(year, month + 1, 1).toISOString();

  const { data: sessions = [] } = usePlayerSessions(userId, {
    start,
    end,
    order: "asc",
    limit: 2000,
    world_id: worldId || undefined,
  });

  const prev = () =>
    setYM(month === 0 ? { year: year - 1, month: 11 } : { year, month: month - 1 });
  const next = () =>
    setYM(month === 11 ? { year: year + 1, month: 0 } : { year, month: month + 1 });

  return (
    <Box className="h-full overflow-auto p-3">
      <title>{`${displayName} — セッション履歴`}</title>
      <Card className="max-w-[960px] mx-auto">
        <CardHeader
          sx={{ pb: 1 }}
          title={
            <Stack
              direction={{ xs: "column", sm: "row" }}
              spacing={1}
              sx={{ alignItems: { xs: "stretch", sm: "center" } }}
            >
              <Stack
                direction="row"
                spacing={1}
                useFlexGap
                sx={{ alignItems: "center", flexWrap: "wrap", flex: 1, minWidth: 0 }}
              >
                <Typography
                  component={Link}
                  to="/"
                  variant="subtitle1"
                  noWrap
                  title={displayName}
                  className="font-medium no-underline text-inherit hover:underline"
                  sx={{ minWidth: 0, flexShrink: 1 }}
                >
                  {displayName}
                </Typography>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ whiteSpace: "nowrap", flexShrink: 0 }}
                >
                  のセッション履歴
                </Typography>
                {worldId && (
                  <Tooltip title={worldId} arrow>
                    <Chip size="small" label={worldId} sx={{ maxWidth: { xs: 180, sm: 320 } }} />
                  </Tooltip>
                )}
              </Stack>
              <Stack
                direction="row"
                spacing={0.5}
                sx={{
                  alignItems: "center",
                  flexShrink: 0,
                  justifyContent: { xs: "space-between", sm: "flex-end" },
                }}
              >
                <IconButton size="small" onClick={prev}>
                  <ChevronLeftIcon />
                </IconButton>
                <Typography variant="subtitle2" className="min-w-[6em] text-center font-semibold">
                  {year}年{String(month + 1).padStart(2, "0")}月
                </Typography>
                <IconButton size="small" onClick={next}>
                  <ChevronRightIcon />
                </IconButton>
              </Stack>
            </Stack>
          }
        />
        <CardContent sx={{ pt: 0 }}>
          <SessionCalendar year={year} month={month} sessions={sessions} />
        </CardContent>
      </Card>
    </Box>
  );
}
