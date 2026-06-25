import { CircularProgress, TableCell, TableRow, Typography } from "@mui/material";

interface Props {
  colSpan: number;
  loading: boolean;
  emptyText: string;
}

export default function TablePlaceholderRow({ colSpan, loading, emptyText }: Props) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} align="center" sx={{ py: 3 }}>
        {loading ? (
          <CircularProgress size={20} />
        ) : (
          <Typography variant="body2" color="text.secondary">
            {emptyText}
          </Typography>
        )}
      </TableCell>
    </TableRow>
  );
}
