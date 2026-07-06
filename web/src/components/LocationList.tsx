import { useState } from "react";
import { Box, Button, Checkbox, FormControlLabel, Stack, Typography } from "@mui/material";
import { DateTimePicker } from "@mui/x-date-pickers/DateTimePicker";
import type { Dayjs } from "dayjs";
import { useInstancesInfinite } from "@/api/queries";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import type { InstanceOut } from "@/api/schemas";
import InstanceListItem from "@/components/InstanceListItem";
import InfiniteScrollFooter from "@/components/InfiniteScrollFooter";

type Range = { start: Dayjs | null; end: Dayjs | null };

const ROW_HEIGHT = 88; // InstanceListItem の概算高さ

interface Props {
  selectedId: number | null;
  onSelect: (instance: InstanceOut) => void;
}

export default function LocationList({ selectedId, onSelect }: Props) {
  const [draft, setDraft] = useState<Range>({ start: null, end: null });
  const [applied, setApplied] = useState<Range>({ start: null, end: null });
  const [openOnly, setOpenOnly] = useState(true);

  const query = useInstancesInfinite({
    start: applied.start?.toISOString(),
    end: applied.end?.toISOString(),
    isOpen: openOnly,
  });
  const { items, scrollRef, virtualItems, paddingTop, paddingBottom, measureElement } =
    useInfiniteTable(query, ROW_HEIGHT);

  return (
    <Box className="flex flex-col h-full bg-neutral-50 border-r border-neutral-200">
      <Stack spacing={1.5} className="p-3 border-b border-neutral-200">
        <Stack direction="row" spacing={1}>
          <DateTimePicker
            label="開始"
            value={draft.start}
            onChange={(v) => setDraft((d) => ({ ...d, start: v }))}
            slotProps={{ textField: { size: "small", fullWidth: true } }}
          />
          <DateTimePicker
            label="終了"
            value={draft.end}
            onChange={(v) => setDraft((d) => ({ ...d, end: v }))}
            slotProps={{ textField: { size: "small", fullWidth: true } }}
          />
        </Stack>
        <FormControlLabel
          control={
            <Checkbox
              size="small"
              checked={openOnly}
              onChange={(e) => setOpenOnly(e.target.checked)}
            />
          }
          label="オープン中のみ"
        />
        <Button variant="contained" color="inherit" size="small" onClick={() => setApplied(draft)}>
          絞り込み
        </Button>
      </Stack>

      <Box ref={scrollRef} className="flex-1 min-h-0 overflow-y-auto">
        {items.length === 0 ? (
          <Typography
            variant="body2"
            color={query.isError ? "error" : "text.secondary"}
            className="p-3"
          >
            {query.isError
              ? "読み込みに失敗しました"
              : query.isLoading
                ? "読み込み中..."
                : "該当なし"}
          </Typography>
        ) : (
          <>
            <div style={{ height: paddingTop }} />
            {virtualItems.map((vi) => {
              const inst = items[vi.index];
              return (
                <div key={inst.id} ref={measureElement} data-index={vi.index}>
                  <InstanceListItem
                    inst={inst}
                    selected={inst.id === selectedId}
                    onClick={() => onSelect(inst)}
                  />
                </div>
              );
            })}
            <div style={{ height: paddingBottom }} />
            <InfiniteScrollFooter visible={query.isFetchingNextPage} />
          </>
        )}
      </Box>
    </Box>
  );
}
