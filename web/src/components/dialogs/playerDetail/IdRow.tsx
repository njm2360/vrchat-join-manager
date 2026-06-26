import { useState } from "react";
import { IconButton, Stack, TextField, Typography } from "@mui/material";
import CheckIcon from "@mui/icons-material/Check";
import CloseIcon from "@mui/icons-material/Close";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import EditIcon from "@mui/icons-material/Edit";
import LaunchIcon from "@mui/icons-material/Launch";
import { useSetPlayerDiscord } from "@/api/queries";

interface EditOptions {
  userId: string;
  onSaved?: (next: string | null) => void;
  onError?: (message: string) => void;
}

interface Props {
  label: string;
  value: string | null | undefined;
  onCopy?: () => void;
  externalHref?: string;
  externalTitle?: string;
  edit?: EditOptions;
}

export default function IdRow({ label, value, onCopy, externalHref, externalTitle, edit }: Props) {
  const hasValue = !!value;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const mutation = useSetPlayerDiscord(edit?.userId ?? "");

  const startEdit = () => {
    setDraft(value ?? "");
    setEditing(true);
  };

  const save = () => {
    if (!edit) return;
    const trimmed = draft.trim();
    const next = trimmed === "" ? null : trimmed;
    if ((value ?? null) === next) {
      setEditing(false);
      return;
    }
    mutation.mutate(next, {
      onSuccess: () => {
        edit.onSaved?.(next);
        setEditing(false);
      },
      onError: (e) => edit.onError?.((e as Error).message),
    });
  };

  return (
    <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: "center", flexWrap: "wrap" }}>
      <Typography variant="caption" color="text.secondary" className="min-w-[80px] shrink-0">
        {label}
      </Typography>

      {editing ? (
        <>
          <TextField
            size="small"
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                save();
              } else if (e.key === "Escape") {
                e.preventDefault();
                setEditing(false);
              }
            }}
            placeholder="空欄で削除"
            disabled={mutation.isPending}
            slotProps={{
              htmlInput: {
                autoComplete: "off",
                style: {
                  fontFamily: "ui-monospace, monospace",
                  fontSize: "0.85rem",
                  padding: "4px 8px",
                },
              },
            }}
          />
          <IconButton
            size="small"
            onClick={save}
            disabled={mutation.isPending}
            title="保存 (Enter)"
          >
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
              title={externalTitle ?? "外部リンクを開く"}
            >
              <LaunchIcon fontSize="inherit" />
            </IconButton>
          )}
          {edit && (
            <IconButton size="small" onClick={startEdit} title={`${label}を編集`}>
              <EditIcon fontSize="inherit" />
            </IconButton>
          )}
        </>
      )}
    </Stack>
  );
}
