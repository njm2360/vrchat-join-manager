import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Box,
  Card,
  CardContent,
  CircularProgress,
  InputAdornment,
  Stack,
  TableCell,
  TextField,
  Typography,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import { usePlayerDetail, usePlayersInfinite } from "@/api/queries";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import { useDebouncedValue } from "@/hooks/useDebouncedValue";
import PlayerLink from "@/components/PlayerLink";
import VirtualTable from "@/components/table/VirtualTable";
import type { TableColumn } from "@/components/table/SortableTableHead";

const COLUMNS: TableColumn<never>[] = [
  { key: "display_name", label: "名前" },
  { key: "user_id", label: "ユーザーID", width: 360 },
];

function Hint({ text }: { text: string }) {
  return (
    <Typography color="text.secondary" variant="body2" className="p-4 text-center">
      {text}
    </Typography>
  );
}

export default function PlayerSearchPage() {
  const [params, setParams] = useSearchParams();
  const [term, setTerm] = useState(params.get("q") ?? "");

  const onChange = (v: string) => {
    setTerm(v);
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (v) p.set("q", v);
        else p.delete("q");
        return p;
      },
      { replace: true },
    );
  };

  const debounced = useDebouncedValue(term.trim(), 300);
  const idMode = debounced.startsWith("usr_");
  const nameMode = !idMode && debounced.length >= 1;

  const query = usePlayersInfinite({ name: debounced }, { enabled: nameMode });
  const table = useInfiniteTable(query);
  const detail = usePlayerDetail(debounced, { enabled: idMode });

  return (
    <Box className="h-full overflow-auto p-3">
      <title>プレイヤー検索</title>
      <Card className="max-w-[960px] mx-auto">
        <CardContent>
          <Stack spacing={2}>
            <TextField
              fullWidth
              size="small"
              autoFocus
              value={term}
              onChange={(e) => onChange(e.target.value)}
              placeholder="ユーザー名またはユーザーID"
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon fontSize="small" />
                    </InputAdornment>
                  ),
                },
              }}
            />

            {idMode ? (
              detail.isLoading ? (
                <Box className="flex justify-center p-4">
                  <CircularProgress size={24} />
                </Box>
              ) : detail.isError || !detail.data ? (
                <Hint text="プレイヤーが見つかりません" />
              ) : (
                <Card variant="outlined">
                  <CardContent>
                    <Typography variant="subtitle1" className="font-medium">
                      <PlayerLink
                        userId={detail.data.user_id}
                        displayName={detail.data.display_name}
                      />
                    </Typography>
                    <Typography variant="caption" color="text.secondary" className="font-mono">
                      {detail.data.user_id}
                    </Typography>
                  </CardContent>
                </Card>
              )
            ) : nameMode ? (
              <VirtualTable
                columns={COLUMNS}
                sortBy={"" as never}
                order="asc"
                onSort={() => {}}
                table={table}
                isLoading={query.isLoading}
                isError={query.isError}
                isFetchingNextPage={query.isFetchingNextPage}
                emptyText="該当なし"
                rowKey={(p) => p.user_id}
                renderCells={(p) => (
                  <>
                    <TableCell className="truncate max-w-[240px]">
                      <PlayerLink userId={p.user_id} displayName={p.display_name} stopPropagation />
                    </TableCell>
                    <TableCell className="font-mono text-neutral-500 truncate max-w-[360px]">
                      {p.user_id}
                    </TableCell>
                  </>
                )}
              />
            ) : (
              <Hint text="ユーザー名またはユーザーID（usr_...）で検索してください" />
            )}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}
