import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  Typography,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import { Link as RouterLink } from "react-router-dom";
import { api } from "@/api/client";
import type { InstanceOut } from "@/api/schemas";
import InstanceListItem from "@/components/InstanceListItem";

interface Props {
  open: boolean;
  onClose: () => void;
  current: InstanceOut;
  linkTarget?: string;
}

export default function CompareInstanceDialog({
  open,
  onClose,
  current,
  linkTarget = "_blank",
}: Props) {
  const start = current.opened_at;
  const end = useMemo(
    () => current.closed_at ?? new Date().toISOString(),
    [current.id, current.closed_at, open],
  );

  const { data: instances = [], isLoading, isError } = useQuery<InstanceOut[]>({
    queryKey: ["instances", "overlap", current.id, start, end],
    enabled: open,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/instances", {
        params: { query: { start, end } },
      });
      if (error) throw new Error("failed to load instances");
      return data ?? [];
    },
  });

  const overlapping = instances.filter((inst) => inst.id !== current.id);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth scroll="paper">
      <DialogTitle className="flex items-center">
        <Box className="flex-1">比較するインスタンスを選択</Box>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers className="p-0!">
        {isLoading ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            読み込み中...
          </Typography>
        ) : isError ? (
          <Typography variant="body2" color="error" className="p-3">
            インスタンスの読み込みに失敗しました
          </Typography>
        ) : overlapping.length === 0 ? (
          <Typography variant="body2" color="text.secondary" className="p-3">
            比較できる他のインスタンスがありません
          </Typography>
        ) : (
          <List disablePadding>
            {overlapping.map((inst) => (
              <InstanceListItem
                key={inst.id}
                inst={inst}
                component={RouterLink}
                to={`/violations?ids=${[current.id, inst.id].sort((a, b) => a - b).join(",")}`}
                target={linkTarget}
                onClick={onClose}
              />
            ))}
          </List>
        )}
      </DialogContent>
    </Dialog>
  );
}
