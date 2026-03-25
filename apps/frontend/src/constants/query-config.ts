export const STALE_TIMES = {
  REALTIME: 5_000,
  SHORT: 30_000,
  NORMAL: 60_000,
  LONG: 5 * 60 * 1000,
} as const;

export const REFETCH_INTERVALS = {
  FAST: 5_000,
  NORMAL: 10_000,
  STATS: 15_000,
  SLOW: 30_000,
} as const;

export const GC_TIMES = {
  DEFAULT: 5 * 60 * 1000,
} as const;
