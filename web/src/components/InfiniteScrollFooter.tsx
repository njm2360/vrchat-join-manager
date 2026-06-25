import { Box, CircularProgress } from "@mui/material";

export default function InfiniteScrollFooter({ visible }: { visible: boolean }) {
  if (!visible) return null;
  return (
    <Box sx={{ display: "flex", justifyContent: "center", py: 1 }}>
      <CircularProgress size={20} />
    </Box>
  );
}
