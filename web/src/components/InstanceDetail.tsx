import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tab,
  Tabs,
} from "@mui/material";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import LockIcon from "@mui/icons-material/Lock";
import DeleteIcon from "@mui/icons-material/Delete";
import type { InstanceOut } from "@/api/schemas";
import InstanceInfo from "@/components/InstanceInfo";
import InstanceStatsPanel from "@/components/InstanceStatsPanel";
import TimelineTab from "@/components/tabs/TimelineTab";
import EventsTab from "@/components/tabs/EventsTab";
import SessionsTab from "@/components/tabs/SessionsTab";
import PlayersTab from "@/components/tabs/PlayersTab";
import VisitorsTab from "@/components/tabs/VisitorsTab";
import CompareInstanceDialog from "@/components/dialogs/CompareInstanceDialog";
import CloseInstanceDialog from "@/components/dialogs/CloseInstanceDialog";
import DeleteInstanceDialog from "@/components/dialogs/DeleteInstanceDialog";

type TabKey = "timeline" | "events" | "sessions" | "players" | "visitors";
const TAB_KEYS: readonly TabKey[] = ["timeline", "events", "sessions", "players", "visitors"];
const DEFAULT_TAB: TabKey = "timeline";

interface Props {
  instanceId: number;
  instance: InstanceOut | null;
  onBack: () => void;
  isMobile: boolean;
}

export default function InstanceDetail({ instanceId, instance, onBack, isMobile }: Props) {
  const [params, setParams] = useSearchParams();
  const tabParam = params.get("tab");
  const tab: TabKey = (TAB_KEYS as readonly string[]).includes(tabParam ?? "")
    ? (tabParam as TabKey)
    : DEFAULT_TAB;
  const setTab = (next: TabKey) =>
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set("tab", next);
        return p;
      },
      { replace: true },
    );

  useEffect(() => {
    if (params.get("tab")) return;
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set("tab", DEFAULT_TAB);
        return p;
      },
      { replace: true },
    );
  }, []);
  const [compareOpen, setCompareOpen] = useState(false);
  const [closeOpen, setCloseOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const menuOpen = Boolean(menuAnchor);
  const closeMenu = () => setMenuAnchor(null);

  const actionMenu = instance && (
    <>
      <IconButton
        size="small"
        onClick={(e) => setMenuAnchor(e.currentTarget)}
        aria-label="操作メニュー"
      >
        <MoreVertIcon />
      </IconButton>
      <Menu anchorEl={menuAnchor} open={menuOpen} onClose={closeMenu}>
        {!instance.closed_at && (
          <MenuItem
            onClick={() => {
              closeMenu();
              setCloseOpen(true);
            }}
          >
            <ListItemIcon>
              <LockIcon fontSize="small" color="warning" />
            </ListItemIcon>
            <ListItemText>クローズ</ListItemText>
          </MenuItem>
        )}
        <MenuItem
          onClick={() => {
            closeMenu();
            setDeleteOpen(true);
          }}
        >
          <ListItemIcon>
            <DeleteIcon fontSize="small" color="error" />
          </ListItemIcon>
          <ListItemText sx={{ color: "error.main" }}>削除</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );

  return (
    <Box className="h-full overflow-auto p-3">
      <Card className="h-full flex flex-col">
        <CardHeader
          avatar={
            isMobile ? (
              <IconButton onClick={onBack} size="small" aria-label="戻る">
                <ArrowBackIcon />
              </IconButton>
            ) : undefined
          }
          title={instance && <InstanceInfo instance={instance} />}
          action={actionMenu}
          className="border-b border-neutral-200"
          sx={{
            alignItems: "flex-start",
            "& .MuiCardHeader-action": { alignSelf: "flex-start", m: 0 },
            "& .MuiCardHeader-content": { minWidth: 0, overflow: "hidden" },
          }}
        />
        <InstanceStatsPanel instanceId={instanceId} />
        <Box className="border-b border-neutral-200" sx={{ px: 2 }}>
          <Tabs
            value={tab}
            onChange={(_, v: TabKey) => setTab(v)}
            variant="scrollable"
            scrollButtons="auto"
          >
            <Tab value="timeline" label="人数推移" />
            <Tab value="events" label="入退場ログ" />
            <Tab value="sessions" label="セッション一覧" />
            <Tab value="players" label="在室中" />
            <Tab value="visitors" label="訪れた人" />
          </Tabs>
        </Box>
        <CardContent className="flex-1 min-h-0 overflow-auto">
          {tab === "timeline" && (
            <TimelineTab
              instanceId={instanceId}
              instance={instance}
              onCompare={() => setCompareOpen(true)}
            />
          )}
          {tab === "events" && <EventsTab instanceId={instanceId} />}
          {tab === "sessions" && <SessionsTab instanceId={instanceId} />}
          {tab === "players" && <PlayersTab instanceId={instanceId} />}
          {tab === "visitors" && <VisitorsTab instanceId={instanceId} />}
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
  );
}
