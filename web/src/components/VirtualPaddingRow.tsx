import { TableCell, TableRow } from "@mui/material";

interface Props {
  height: number;
  colSpan: number;
}

export default function VirtualPaddingRow({ height, colSpan }: Props) {
  if (height <= 0) return null;
  return (
    <TableRow style={{ height }}>
      <TableCell colSpan={colSpan} sx={{ p: 0, border: 0 }} />
    </TableRow>
  );
}
