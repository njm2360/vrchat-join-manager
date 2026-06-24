import { useState, type ReactNode } from "react";
import { Box, Stack } from "@mui/material";
import { DateTimePicker } from "@mui/x-date-pickers/DateTimePicker";
import type { Dayjs } from "dayjs";

export interface DateRange {
  start?: string;
  end?: string;
}

interface Props {
  onApply: (range: DateRange) => void;
  children?: ReactNode;
}

const iso = (d: Dayjs | null) => (d && d.isValid() ? d.toISOString() : undefined);

export default function DateRangeFilter({ onApply, children }: Props) {
  const [start, setStart] = useState<Dayjs | null>(null);
  const [end, setEnd] = useState<Dayjs | null>(null);

  return (
    <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: "center", flexWrap: "wrap" }}>
      <DateTimePicker
        label="開始"
        value={start}
        onChange={setStart}
        onAccept={(v) => onApply({ start: iso(v), end: iso(end) })}
        slotProps={{ textField: { size: "small" } }}
      />
      <Box className="text-neutral-500 text-sm">〜</Box>
      <DateTimePicker
        label="終了"
        value={end}
        onChange={setEnd}
        onAccept={(v) => onApply({ start: iso(start), end: iso(v) })}
        slotProps={{ textField: { size: "small" } }}
      />
      {children}
    </Stack>
  );
}
