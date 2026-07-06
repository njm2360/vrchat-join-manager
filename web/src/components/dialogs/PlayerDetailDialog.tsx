import { useState } from "react";
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  Tab,
  Tabs,
  Typography,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import { useSnackbar } from "notistack";
import { useInstance, usePlayerDetail } from "@/api/queries";
import { copyText } from "@/utils/clipboard";
import IdRow from "@/components/dialogs/playerDetail/IdRow";
import OverviewTab from "@/components/dialogs/playerDetail/OverviewTab";
import SessionsTab from "@/components/dialogs/playerDetail/SessionsTab";
import InstanceTab from "@/components/dialogs/playerDetail/InstanceTab";

export interface PlayerDetailCtx {
  userId: string;
  displayName: string;
  instanceId?: number;
}

interface Props {
  open: boolean;
  onClose: () => void;
  ctx: PlayerDetailCtx;
}

type TabKey = "overview" | "sessions" | "instance";

export default function PlayerDetailDialog({ open, onClose, ctx }: Props) {
  const { userId, displayName: fallbackName, instanceId } = ctx;
  const hasInstance = instanceId != null;
  const [tab, setTab] = useState<TabKey>("overview");
  const { enqueueSnackbar } = useSnackbar();

  const { data: detail, isError: detailError } = usePlayerDetail(userId, { enabled: open });
  const { data: instance } = useInstance(instanceId ?? null, { enabled: hasInstance && open });

  const displayName = detail?.display_name ?? fallbackName;
  const discordId = detail?.discord_id ?? null;

  const copy = async (label: string, value: string) => {
    try {
      await copyText(value);
      enqueueSnackbar(`${label}をコピーしました`, { variant: "success" });
    } catch {
      enqueueSnackbar("クリップボードへのコピーに失敗しました", { variant: "error" });
    }
  };

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
              onCopy={() => copy("ユーザーID", userId)}
              externalHref={`https://vrchat.com/home/user/${encodeURIComponent(userId)}`}
              externalTitle="VRChat のプロフィールを開く"
            />
            <IdRow
              label="Discord ID"
              value={discordId}
              onCopy={discordId ? () => copy("Discord ID", discordId) : undefined}
              edit={{
                userId,
                onSaved: (next) =>
                  enqueueSnackbar(next ? "Discord IDを更新しました" : "Discord IDを削除しました", {
                    variant: "success",
                  }),
                onError: (msg) => enqueueSnackbar(msg, { variant: "error" }),
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
          <Tab value="sessions" label="セッション" />
          {hasInstance && <Tab value="instance" label="このインスタンス" />}
        </Tabs>
      </Box>

      <DialogContent dividers className="p-0!">
        {tab === "overview" && (
          <OverviewTab userId={userId} detail={detail ?? null} error={detailError} />
        )}
        {tab === "sessions" && <SessionsTab userId={userId} />}
        {tab === "instance" && hasInstance && (
          <InstanceTab userId={userId} instance={instance ?? null} />
        )}
      </DialogContent>
    </Dialog>
  );
}
