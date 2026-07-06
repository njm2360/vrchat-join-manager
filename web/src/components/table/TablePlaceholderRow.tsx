import { CircularProgress, TableCell, TableRow, Typography } from "@mui/material";

interface Props {
  colSpan: number;
  loading: boolean;
  error?: boolean;
  emptyText: string;
  errorText?: string;
}

export default function TablePlaceholderRow({
  colSpan,
  loading,
  error,
  emptyText,
  errorText = "読み込みに失敗しました",
}: Props) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} align="center" sx={{ py: 3 }}>
        {error ? (
          <Typography variant="body2" color="error">
            {errorText}
          </Typography>
        ) : loading ? (
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
