import { useCallback, useEffect, useState } from "react";

export const PAGE_SIZE = 5;

interface UseVisibleWindowOptions {
  readonly totalCount: number;
  /** Change this when the list identity changes (e.g. first id + length, or appId). */
  readonly resetKey: string | number;
  readonly pageSize?: number;
}

interface UseVisibleWindowResult {
  readonly visibleCount: number;
  readonly hasMore: boolean;
  readonly loadMore: () => void;
}

/**
 * Progressive client-side window over an already-loaded list.
 * Starts at `pageSize` and grows by `pageSize` on each `loadMore`, never past `totalCount`.
 */
export function useVisibleWindow({
  totalCount,
  resetKey,
  pageSize = PAGE_SIZE,
}: UseVisibleWindowOptions): UseVisibleWindowResult {
  const [visibleCount, setVisibleCount] = useState(pageSize);

  useEffect(() => {
    setVisibleCount(pageSize);
  }, [resetKey, pageSize]);

  const clampedVisible = Math.min(visibleCount, Math.max(totalCount, 0));
  const hasMore = clampedVisible < totalCount;

  const loadMore = useCallback(() => {
    setVisibleCount((prev) => Math.min(prev + pageSize, totalCount));
  }, [pageSize, totalCount]);

  return {
    visibleCount: clampedVisible,
    hasMore,
    loadMore,
  };
}
