import { useEffect, useMemo, useRef, type RefObject } from "react";
import { useVirtualizer, type VirtualItem } from "@tanstack/react-virtual";

const ROW_HEIGHT = 33;

interface InfiniteQueryLike<T> {
  data?: { pages: T[][] };
  fetchNextPage: () => void;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
}

export interface InfiniteTableState<T> {
  items: T[];
  scrollRef: RefObject<HTMLDivElement | null>;
  virtualItems: VirtualItem[];
  paddingTop: number;
  paddingBottom: number;
  measureElement: (node: Element | null) => void;
}

export function useInfiniteTable<T>(
  query: InfiniteQueryLike<T>,
  rowHeight = ROW_HEIGHT,
): InfiniteTableState<T> {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } = query;
  const items = useMemo(() => data?.pages.flat() ?? [], [data]);

  const scrollRef = useRef<HTMLDivElement | null>(null);
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => rowHeight,
    overscan: 12,
  });
  const virtualItems = virtualizer.getVirtualItems();

  const lastIndex = virtualItems.at(-1)?.index ?? 0;
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage && lastIndex >= items.length - 1) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, lastIndex, items.length, fetchNextPage]);

  const paddingTop = virtualItems[0]?.start ?? 0;
  const paddingBottom = virtualizer.getTotalSize() - (virtualItems.at(-1)?.end ?? 0);

  return {
    items,
    scrollRef,
    virtualItems,
    paddingTop,
    paddingBottom,
    measureElement: virtualizer.measureElement,
  };
}
