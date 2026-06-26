import { useState } from "react";
import { Box, Card, CardContent, Stack, Typography } from "@mui/material";
import { useInstancesInfinite } from "@/api/queries";
import { useInfiniteTable } from "@/hooks/useInfiniteTable";
import type { InstanceOut } from "@/api/schemas";
import InstanceListItem from "@/components/InstanceListItem";
import InfiniteScrollFooter from "@/components/InfiniteScrollFooter";
import CompareInstanceDialog from "@/components/dialogs/CompareInstanceDialog";

const ROW_HEIGHT = 88;

export default function CompareLanding() {
  const [base, setBase] = useState<InstanceOut | null>(null);

  const query = useInstancesInfinite({ isOpen: false });
  const { items, scrollRef, virtualItems, paddingTop, paddingBottom, measureElement } =
    useInfiniteTable(query, ROW_HEIGHT);

  return (
    <Box className="h-full overflow-hidden p-3 bg-neutral-50">
      <Card className="h-full max-w-[720px] mx-auto flex flex-col">
        <CardContent className="flex flex-col gap-2 min-h-0 flex-1">
          <Stack spacing={0.5}>
            <Typography variant="subtitle1" className="font-medium">
              違反検知するインスタンスを選択
            </Typography>
          </Stack>

          <Box
            ref={scrollRef}
            className="flex-1 min-h-0 overflow-y-auto border-t border-neutral-200"
          >
            {items.length === 0 ? (
              <Typography variant="body2" color="text.secondary" className="p-3">
                {query.isLoading ? "読み込み中..." : "該当なし"}
              </Typography>
            ) : (
              <>
                <div style={{ height: paddingTop }} />
                {virtualItems.map((vi) => {
                  const inst = items[vi.index];
                  return (
                    <div key={inst.id} ref={measureElement} data-index={vi.index}>
                      <InstanceListItem inst={inst} onClick={() => setBase(inst)} />
                    </div>
                  );
                })}
                <div style={{ height: paddingBottom }} />
                <InfiniteScrollFooter visible={query.isFetchingNextPage} />
              </>
            )}
          </Box>
        </CardContent>
      </Card>

      {base && (
        <CompareInstanceDialog
          open
          current={base}
          onClose={() => setBase(null)}
          linkTarget="_self"
        />
      )}
    </Box>
  );
}
