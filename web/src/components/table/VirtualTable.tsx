import type { Key, ReactNode } from "react";
import { Table, TableBody, TableContainer, TableRow, Paper } from "@mui/material";
import type { Order } from "@/hooks/useSortState";
import type { InfiniteTableState } from "@/hooks/useInfiniteTable";
import SortableTableHead, { type TableColumn } from "@/components/table/SortableTableHead";
import TablePlaceholderRow from "@/components/table/TablePlaceholderRow";
import VirtualPaddingRow from "@/components/table/VirtualPaddingRow";
import InfiniteScrollFooter from "@/components/InfiniteScrollFooter";

interface Props<T, K extends string> {
  columns: TableColumn<K>[];
  sortBy: K;
  order: Order;
  onSort: (key: K) => void;
  table: InfiniteTableState<T>;
  isLoading: boolean;
  isFetchingNextPage: boolean;
  emptyText: string;
  rowKey: (item: T) => Key;
  renderCells: (item: T) => ReactNode;
  maxHeight?: number;
}

export default function VirtualTable<T, K extends string>({
  columns,
  sortBy,
  order,
  onSort,
  table,
  isLoading,
  isFetchingNextPage,
  emptyText,
  rowKey,
  renderCells,
  maxHeight = 520,
}: Props<T, K>) {
  const { items, scrollRef, virtualItems, paddingTop, paddingBottom, measureElement } = table;

  return (
    <>
      <TableContainer ref={scrollRef} component={Paper} variant="outlined" sx={{ maxHeight }}>
        <Table size="small" stickyHeader>
          <SortableTableHead columns={columns} sortBy={sortBy} order={order} onSort={onSort} />
          <TableBody>
            {items.length === 0 ? (
              <TablePlaceholderRow
                colSpan={columns.length}
                loading={isLoading}
                emptyText={emptyText}
              />
            ) : (
              <>
                <VirtualPaddingRow height={paddingTop} colSpan={columns.length} />
                {virtualItems.map((vi) => {
                  const item = items[vi.index];
                  return (
                    <TableRow key={rowKey(item)} hover ref={measureElement} data-index={vi.index}>
                      {renderCells(item)}
                    </TableRow>
                  );
                })}
                <VirtualPaddingRow height={paddingBottom} colSpan={columns.length} />
              </>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <InfiniteScrollFooter visible={isFetchingNextPage} />
    </>
  );
}
