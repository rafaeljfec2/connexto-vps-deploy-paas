import { useMemo } from "react";
import { type LogLevel, PINO_LEVEL_MAP, type ParsedLogEntry } from "../types";

const KNOWN_KEYS = new Set([
  "time",
  "level",
  "msg",
  "message",
  "context",
  "pid",
  "hostname",
]);

function parseLevel(level: unknown): { level: LogLevel; levelNumber: number } {
  if (typeof level === "number" && level in PINO_LEVEL_MAP) {
    const mapped = PINO_LEVEL_MAP[level as keyof typeof PINO_LEVEL_MAP];
    return { level: mapped ?? "info", levelNumber: level };
  }
  if (typeof level === "string") {
    const lower = level.toLowerCase();
    const mapped: LogLevel[] = [
      "trace",
      "debug",
      "info",
      "warn",
      "error",
      "fatal",
    ];
    const idx = mapped.indexOf(lower as LogLevel);
    if (idx >= 0) {
      return { level: mapped[idx] as LogLevel, levelNumber: (idx + 1) * 10 };
    }
  }
  return { level: "info", levelNumber: 30 };
}

function parseTimestamp(time: unknown): Date | null {
  if (time == null) return null;
  if (typeof time === "number") {
    const d = new Date(time > 1e12 ? time : time * 1000);
    return Number.isNaN(d.getTime()) ? null : d;
  }
  if (typeof time === "string") {
    const d = new Date(time);
    return Number.isNaN(d.getTime()) ? null : d;
  }
  return null;
}

function tryParseJsonLine(line: string): ParsedLogEntry | null {
  const trimmed = line.trim();
  if ((trimmed.startsWith("{") && trimmed.endsWith("}")) === false) {
    return null;
  }
  let obj: Record<string, unknown>;
  try {
    obj = JSON.parse(trimmed) as Record<string, unknown>;
  } catch {
    return null;
  }

  const { level: levelKey, levelNumber } = parseLevel(obj.level);
  const timestamp = parseTimestamp(obj.time);
  const message =
    (typeof obj.msg === "string" ? obj.msg : null) ??
    (typeof obj.message === "string" ? obj.message : null) ??
    "";
  const context = typeof obj.context === "string" ? obj.context : null;
  const pid = typeof obj.pid === "number" ? obj.pid : null;
  const hostname = typeof obj.hostname === "string" ? obj.hostname : null;

  const extra: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (KNOWN_KEYS.has(key)) continue;
    extra[key] = value;
  }

  return {
    id: `line-${timestamp?.getTime() ?? 0}-${Math.random().toString(36).slice(2, 9)}`,
    lineNumber: 0,
    raw: line,
    timestamp,
    level: levelKey,
    levelNumber,
    message,
    context,
    pid,
    hostname,
    extra,
    isJson: true,
  };
}

function parsePlainLine(line: string, index: number): ParsedLogEntry {
  const trimmed = line.trim();
  return {
    id: `plain-${index}`,
    lineNumber: index + 1,
    raw: line,
    timestamp: null,
    level: "info",
    levelNumber: 30,
    message: trimmed,
    context: null,
    pid: null,
    hostname: null,
    extra: {},
    isJson: false,
  };
}

export function useLogParser(rawLogs: string | null | undefined): {
  entries: ParsedLogEntry[];
  uniqueContexts: string[];
} {
  return useMemo(() => {
    if (rawLogs == null || rawLogs === "") {
      return { entries: [], uniqueContexts: [] };
    }
    const lines = rawLogs.split("\n").filter((l) => l.length > 0);
    const entries: ParsedLogEntry[] = [];
    const contextSet = new Set<string>();

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line == null) continue;
      const parsed = tryParseJsonLine(line);
      if (parsed) {
        const entry: ParsedLogEntry = {
          ...parsed,
          lineNumber: i + 1,
          id: `line-${i}-${parsed.level}-${parsed.lineNumber}`,
        };
        entries.push(entry);
        if (entry.context) contextSet.add(entry.context);
      } else {
        entries.push(parsePlainLine(line, i));
      }
    }

    const uniqueContexts = Array.from(contextSet).sort();
    return { entries, uniqueContexts };
  }, [rawLogs]);
}
