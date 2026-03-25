import { useState } from "react";
import { cn } from "@/lib/utils";
import { highlightText } from "./hooks/use-log-search";
import { JsonExpandable } from "./json-expandable";
import type { LogLevel, ParsedLogEntry } from "./types";

interface LogRowProps {
  readonly line: ParsedLogEntry;
  readonly searchTerm: string;
  readonly searchRegex: RegExp | null;
  readonly isCurrentMatch?: boolean;
  readonly compact?: boolean;
}

const LEVEL_CLASSES: Record<LogLevel, string> = {
  trace: "bg-slate-600/30 text-slate-400",
  debug: "bg-slate-600/40 text-slate-300",
  info: "bg-sky-500/20 text-sky-400",
  warn: "bg-yellow-500/20 text-yellow-400",
  error: "bg-red-500/20 text-red-400",
  fatal: "bg-red-600/30 text-red-300 font-semibold",
};

function formatTime(date: Date): string {
  return date.toLocaleTimeString("pt-BR", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

export function LogRow({
  line,
  searchTerm,
  searchRegex,
  isCurrentMatch = false,
  compact = false,
}: LogRowProps) {
  const [expanded, setExpanded] = useState(false);
  const hasExtra = line.isJson && Object.keys(line.extra).length > 0;

  return (
    <div
      className={cn(
        "flex group hover:bg-white/5 transition-colors border-b border-slate-800/50",
        compact ? "py-px" : "py-1",
        isCurrentMatch && "bg-yellow-500/20 ring-1 ring-yellow-500/50",
      )}
      data-line-number={line.lineNumber}
      role="row"
    >
      <span
        className={cn(
          "select-none text-muted-foreground/50 text-right shrink-0 tabular-nums border-r border-slate-800/50 pr-2",
          compact ? "w-10 text-[10px] leading-4" : "w-12 text-xs leading-5",
        )}
      >
        {line.lineNumber}
      </span>
      <span
        className={cn(
          "shrink-0 font-mono text-slate-500 tabular-nums pr-2 border-r border-slate-800/50",
          compact ? "w-16 text-[10px] leading-4" : "w-20 text-xs leading-5",
        )}
      >
        {line.timestamp != null ? formatTime(line.timestamp) : "—"}
      </span>
      <span
        className={cn(
          "shrink-0 px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wider mr-2",
          LEVEL_CLASSES[line.level],
        )}
      >
        {line.level}
      </span>
      <span
        className={cn(
          "shrink-0 text-slate-500 font-mono truncate max-w-[120px]",
          compact ? "text-[10px] leading-4" : "text-xs leading-5",
        )}
        title={line.context ?? undefined}
      >
        {line.context ?? "—"}
      </span>
      <div
        className={cn(
          "min-w-0 flex-1 font-mono cursor-default",
          compact ? "text-xs leading-4" : "text-sm leading-5",
        )}
      >
        <button
          type="button"
          className={cn(
            "text-left w-full break-words",
            hasExtra && "cursor-pointer",
          )}
          onClick={() => hasExtra && setExpanded((e) => !e)}
        >
          {highlightText(line.message, searchTerm, searchRegex)}
        </button>
        {expanded && hasExtra && (
          <JsonExpandable extra={line.extra} className="mt-1" />
        )}
      </div>
    </div>
  );
}
