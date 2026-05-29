import { useCallback, useState } from 'react'

export type Order = 'asc' | 'desc'

// ソート可能なテーブル用の状態管理。
// 同じキーを再クリックで昇順/降順をトグル、別キー選択時は newKeyOrder を適用する。
export function useSortState<K extends string>(
  initialKey: K,
  initialOrder: Order = 'asc',
  newKeyOrder: Order = 'desc',
) {
  const [sort, setSort] = useState<{ by: K; order: Order }>({
    by: initialKey,
    order: initialOrder,
  })

  const toggleSort = useCallback(
    (key: K) => {
      setSort((s) =>
        s.by === key
          ? { by: key, order: s.order === 'asc' ? 'desc' : 'asc' }
          : { by: key, order: newKeyOrder },
      )
    },
    [newKeyOrder],
  )

  return { sortBy: sort.by, order: sort.order, toggleSort }
}
