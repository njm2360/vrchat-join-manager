import { TableCell, TableHead, TableRow, TableSortLabel } from "@mui/material";
import type { Order } from "@/hooks/useSortState";

export interface TableColumn<K extends string> {
  key: string;
  label: string;
  width?: number;
  align?: "right";
  sortKey?: K;
}

interface Props<K extends string> {
  columns: TableColumn<K>[];
  sortBy: K;
  order: Order;
  onSort: (key: K) => void;
}

export default function SortableTableHead<K extends string>({
  columns,
  sortBy,
  order,
  onSort,
}: Props<K>) {
  return (
    <TableHead>
      <TableRow>
        {columns.map((c) => {
          const active = c.sortKey != null && sortBy === c.sortKey;
          return (
            <TableCell
              key={c.key}
              width={c.width}
              align={c.align}
              sortDirection={active ? order : false}
            >
              {c.sortKey != null ? (
                <TableSortLabel
                  active={active}
                  direction={active ? order : "asc"}
                  onClick={() => onSort(c.sortKey!)}
                >
                  {c.label}
                </TableSortLabel>
              ) : (
                c.label
              )}
            </TableCell>
          );
        })}
      </TableRow>
    </TableHead>
  );
}
