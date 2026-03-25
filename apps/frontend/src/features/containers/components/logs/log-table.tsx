import { useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { LogRow } from "./log-row";
import type { ParsedLogEntry } from "./types";

interface LogTableProps {
  readonly entries: readonly ParsedLogEntry[];
  readonly searchTerm: string;
  readonly searchRegex: RegExp | null;
  readonly currentMatchLine?: number;
  readonly autoScroll?: boolean;
  readonly scrollRef?: React.RefObject<HTMLDivElement | null>;
  readonly compact?: boolean;
  readonly className?: string;
}

const TABLE_HEADER = (
  <div
    className="flex shrink-0 border-b border-slate-700 bg-slate-900/80 sticky top-0 z-10 text-xs font-semibold text-slate-400 uppercase tracking-wider"
    role="row"
  >
    <span className="w-12 pr-2 text-right border-r border-slate-800/50 py-2">
      #
    </span>
    <span className="w-20 pr-2 border-r border-slate-800/50 py-2">Time</span>
    <span className="w-16 pr-2 py-2">Level</span>
    <span className="w-[120px] shrink-0 truncate py-2">Context</span>
    <span className="min-w-0 flex-1 py-2 pl-2">Message</span>
  </div>
);

export function LogTable({
  entries,
  searchTerm,
  searchRegex,
  currentMatchLine,
  autoScroll = true,
  scrollRef: externalScrollRef,
  compact = false,
  className,
}: LogTableProps) {
  const internalScrollRef = useRef<HTMLDivElement>(null);
  const scrollRef = externalScrollRef ?? internalScrollRef;

  useEffect(() => {
    if (!autoScroll || !scrollRef.current) return;
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [entries.length, autoScroll, scrollRef]);

  useEffect(() => {
    if (currentMatchLine == null || !scrollRef.current) return;
    const row = scrollRef.current.querySelector(
      `[data-line-number="${currentMatchLine}"]`,
    );
    row?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [currentMatchLine, scrollRef]);

  return (
    <div
      className={cn(
        "flex flex-col rounded-lg border border-slate-800 bg-slate-950 overflow-hidden",
        className,
      )}
    >
      {TABLE_HEADER}
      <div
        ref={scrollRef as React.RefObject<HTMLDivElement>}
        className="overflow-auto flex-1 min-h-0 scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent"
        role="log"
        aria-live="polite"
      >
        <div className="min-w-0">
          {entries.map((entry) => (
            <LogRow
              key={entry.id}
              line={entry}
              searchTerm={searchTerm}
              searchRegex={searchRegex}
              isCurrentMatch={entry.lineNumber === currentMatchLine}
              compact={compact}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
