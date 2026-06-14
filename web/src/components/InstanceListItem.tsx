import type { ElementType } from "react";
import { Chip, ListItemButton, Stack, Typography } from "@mui/material";
import type { ListItemButtonProps } from "@mui/material";
import type { InstanceOut } from "@/api/schemas";
import { extractInstanceNumber, instanceRangeLabel } from "@/utils/format";

type Props = ListItemButtonProps & {
  inst: InstanceOut;
  component?: ElementType;
  to?: string;
  target?: string;
};

export default function InstanceListItem({ inst, className, ...rest }: Props) {
  const ongoing = !inst.closed_at;

  return (
    <ListItemButton className={`block! py-2! ${className ?? ""}`} {...rest}>
      <Typography variant="body2" color="text.secondary" className="block font-mono">
        {extractInstanceNumber(inst.location_id) || "—"}
      </Typography>
      <Typography variant="caption" color="text.secondary" className="block truncate">
        {inst.location_id}
      </Typography>
      <Stack direction="row" spacing={0.5} className="mt-1" sx={{ alignItems: "center" }}>
        <Chip
          size="small"
          label={instanceRangeLabel(inst)}
          color={ongoing ? "success" : "default"}
          variant="filled"
        />
        {ongoing && inst.user_count > 0 && (
          <Chip size="small" label={`${inst.user_count}人`} color="warning" />
        )}
      </Stack>
    </ListItemButton>
  );
}
