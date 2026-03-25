import { useCallback, useState } from "react";
import type { LogFilters, LogLevel, ParsedLogEntry } from "../types";
import { LOG_LEVELS } from "../types";

const DEFAULT_FILTERS: LogFilters = {
  search: "",
  levels: [...LOG_LEVELS],
  contexts: [],
  timeRange: { start: null, end: null },
};

export type TimePreset = "5m" | "15m" | "1h" | "24h" | "all";

function timePresetToRange(preset: TimePreset): {
  start: Date | null;
  end: Date | null;
} {
  const end = new Date();
  if (preset === "all") return { start: null, end: null };
  const start = new Date(end.getTime());
  if (preset === "5m") start.setMinutes(start.getMinutes() - 5);
  else if (preset === "15m") start.setMinutes(start.getMinutes() - 15);
  else if (preset === "1h") start.setHours(start.getHours() - 1);
  else if (preset === "24h") start.setDate(start.getDate() - 1);
  return { start, end };
}

export function useLogFilters(_uniqueContexts: readonly string[]) {
  const [filters, setFilters] = useState<LogFilters>(DEFAULT_FILTERS);
  const [timePreset, setTimePreset] = useState<TimePreset | null>("all");

  const setSearch = useCallback((search: string) => {
    setFilters((prev) => ({ ...prev, search }));
  }, []);

  const setLevels = useCallback((levels: readonly LogLevel[]) => {
    setFilters((prev) => ({ ...prev, levels: [...levels] }));
  }, []);

  const setContexts = useCallback((contexts: readonly string[]) => {
    setFilters((prev) => ({ ...prev, contexts: [...contexts] }));
  }, []);

  const setTimeRange = useCallback(
    (preset: TimePreset | { start: Date | null; end: Date | null }) => {
      if (typeof preset === "string") {
        setTimePreset(preset);
        setFilters((prev) => ({
          ...prev,
          timeRange: timePresetToRange(preset),
        }));
      } else {
        setTimePreset(null);
        setFilters((prev) => ({ ...prev, timeRange: preset }));
      }
    },
    [],
  );

  const clearFilters = useCallback(() => {
    setTimePreset("all");
    setFilters(DEFAULT_FILTERS);
  }, []);

  const filterEntries = useCallback(
    (entries: readonly ParsedLogEntry[]): ParsedLogEntry[] => {
      return entries.filter((entry) => {
        if (
          filters.levels.length > 0 &&
          !filters.levels.includes(entry.level)
        ) {
          return false;
        }
        if (
          filters.contexts.length > 0 &&
          (entry.context == null || !filters.contexts.includes(entry.context))
        ) {
          return false;
        }
        if (filters.timeRange.start != null && entry.timestamp != null) {
          if (entry.timestamp < filters.timeRange.start) return false;
        }
        if (filters.timeRange.end != null && entry.timestamp != null) {
          if (entry.timestamp > filters.timeRange.end) return false;
        }
        return true;
      });
    },
    [filters],
  );

  const hasActiveFilters =
    (filters.levels.length > 0 && filters.levels.length < LOG_LEVELS.length) ||
    filters.contexts.length > 0 ||
    filters.timeRange.start != null;

  return {
    filters,
    timePreset,
    setSearch,
    setLevels,
    setContexts,
    setTimeRange,
    clearFilters,
    filterEntries,
    hasActiveFilters,
  };
}
