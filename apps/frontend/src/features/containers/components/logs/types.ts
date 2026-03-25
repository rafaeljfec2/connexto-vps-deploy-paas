export type LogLevel = "trace" | "debug" | "info" | "warn" | "error" | "fatal";

export interface ParsedLogEntry {
  readonly id: string;
  readonly lineNumber: number;
  readonly raw: string;
  readonly timestamp: Date | null;
  readonly level: LogLevel;
  readonly levelNumber: number;
  readonly message: string;
  readonly context: string | null;
  readonly pid: number | null;
  readonly hostname: string | null;
  readonly extra: Record<string, unknown>;
  readonly isJson: boolean;
}

export interface LogFilters {
  readonly search: string;
  readonly levels: readonly LogLevel[];
  readonly contexts: readonly string[];
  readonly timeRange: { start: Date | null; end: Date | null };
}

export const LOG_LEVELS: readonly LogLevel[] = [
  "trace",
  "debug",
  "info",
  "warn",
  "error",
  "fatal",
] as const;

export const PINO_LEVEL_MAP: Record<number, LogLevel> = {
  10: "trace",
  20: "debug",
  30: "info",
  40: "warn",
  50: "error",
  60: "fatal",
} as const;

export const LOG_LEVEL_ORDER: Record<LogLevel, number> = {
  trace: 0,
  debug: 1,
  info: 2,
  warn: 3,
  error: 4,
  fatal: 5,
};
