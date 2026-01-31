import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Check,
  ChevronDown,
  ChevronUp,
  Copy,
  Expand,
  Filter,
  Minimize2,
  Search,
  Terminal,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { EmptyState } from "@/components/empty-state";
import { cn } from "@/lib/utils";

interface LogViewerProps {
  readonly logs: string | null;
  readonly autoScroll?: boolean;
  readonly title?: string;
}

interface LogFilters {
  readonly search: string;
  readonly types: readonly LogType[];
}

const ALL_FILTER_TYPES: readonly LogType[] = [
  "error",
  "warning",
  "success",
  "info",
  "build",
  "default",
] as const;

const FILTER_LABELS: Record<LogType, string> = {
  error: "Errors",
  warning: "Warnings",
  success: "Success",
  info: "Info",
  build: "Build",
  default: "Other",
};

type LogType = "info" | "success" | "error" | "warning" | "build" | "default";
type LogPrefix = "build" | "deploy" | null;

interface ParsedLogLine {
  readonly lineNumber: number;
  readonly timestamp: string | null;
  readonly prefix: LogPrefix;
  readonly step: string | null;
  readonly content: string;
  readonly type: LogType;
  readonly isEmpty: boolean;
}

const TIMESTAMP_REGEX = /^\[(\d{2}:\d{2}:\d{2})\]\s*/;
const PREFIX_REGEX = /^\[(build|deploy)\]\s*/i;
const STEP_REGEX = /^(#\d+)\s+/;

function determineLogType(content: string, prefix: LogPrefix): LogType {
  const lower = content.toLowerCase();

  if (
    lower.includes("error") ||
    lower.includes("failed") ||
    lower.includes("fatal") ||
    lower.includes("exception") ||
    lower.includes("could not be found")
  ) {
    return "error";
  }

  if (
    lower.includes("success") ||
    lower.includes("completed") ||
    lower.includes("deployed") ||
    lower.includes("healthy") ||
    lower.includes("running") ||
    lower.includes("done") ||
    lower.includes("cached")
  ) {
    return "success";
  }

  if (
    lower.includes("warning") ||
    lower.includes("warn") ||
    lower.includes("deprecated") ||
    lower.includes("obsolete")
  ) {
    return "warning";
  }

  if (
    lower.includes("starting") ||
    lower.includes("syncing") ||
    lower.includes("fetching") ||
    lower.includes("building") ||
    lower.includes("checking") ||
    lower.includes("pulling") ||
    lower.includes("pushing") ||
    lower.includes("deploying") ||
    lower.includes("exporting") ||
    lower.includes("transferring") ||
    lower.includes("unpacking") ||
    lower.includes("naming")
  ) {
    return "info";
  }

  if (prefix === "build") {
    return "build";
  }

  return "default";
}

function parseLogLine(line: string, index: number): ParsedLogLine {
  let remaining = line;
  let timestamp: string | null = null;
  let prefix: LogPrefix = null;
  let step: string | null = null;

  const timestampMatch = TIMESTAMP_REGEX.exec(remaining);
  if (timestampMatch?.[1]) {
    timestamp = timestampMatch[1];
    remaining = remaining.slice(timestampMatch[0].length);
  }

  const prefixMatch = PREFIX_REGEX.exec(remaining);
  if (prefixMatch?.[1]) {
    prefix = prefixMatch[1].toLowerCase() as LogPrefix;
    remaining = remaining.slice(prefixMatch[0].length);
  }

  const stepMatch = STEP_REGEX.exec(remaining);
  if (stepMatch?.[1]) {
    step = stepMatch[1];
    remaining = remaining.slice(stepMatch[0].length);
  }

  const content = remaining.trim();
  const isEmpty = content === "" || content === "...";
  const type = determineLogType(content, prefix);

  return {
    lineNumber: index + 1,
    timestamp,
    prefix,
    step,
    content,
    type,
    isEmpty,
  };
}

interface LogLineProps {
  readonly line: ParsedLogLine;
  readonly searchTerm?: string;
  readonly isCurrentMatch?: boolean;
  readonly compact?: boolean;
}

function highlightText(text: string, searchTerm: string): React.ReactNode {
  if (!searchTerm) return text;

  const escapedTerm = searchTerm.replaceAll(
    /[.*+?^${}()|[\]\\]/g,
    String.raw`\$&`,
  );
  const regex = new RegExp(`(${escapedTerm})`, "gi");
  const parts = text.split(regex);

  return parts.map((part, index) => {
    const isMatch = regex.test(part);
    regex.lastIndex = 0;
    return isMatch ? (
      <mark
        key={`${part}-${index}`}
        className="bg-yellow-500/40 text-inherit rounded px-0.5"
      >
        {part}
      </mark>
    ) : (
      <span key={`${part}-${index}`}>{part}</span>
    );
  });
}

function LogLine({ line, searchTerm, isCurrentMatch, compact }: LogLineProps) {
  if (line.isEmpty) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex group hover:bg-white/5 transition-colors",
        compact ? "py-px" : "py-0.5",
        isCurrentMatch && "bg-yellow-500/20 ring-1 ring-yellow-500/50",
      )}
      data-line-number={line.lineNumber}
    >
      <span
        className={cn(
          "select-none text-muted-foreground/40 text-right mr-2 shrink-0 tabular-nums",
          compact ? "w-6 text-[10px] leading-4" : "w-8 text-xs leading-5 mr-3",
        )}
      >
        {line.lineNumber}
      </span>

      <div
        className={cn(
          "flex items-start min-w-0 flex-1",
          compact ? "gap-1.5" : "gap-2",
        )}
      >
        {line.timestamp && (
          <span
            className={cn(
              "text-slate-500 shrink-0 font-medium tabular-nums",
              compact ? "text-[10px] leading-4" : "text-xs leading-5",
            )}
          >
            {line.timestamp}
          </span>
        )}

        {line.prefix && (
          <span
            className={cn(
              "shrink-0 font-semibold uppercase tracking-wider rounded leading-none",
              compact ? "text-[8px] px-1 py-px" : "text-[10px] px-1.5 py-0.5",
              line.prefix === "build" &&
                "bg-violet-500/20 text-violet-400 border border-violet-500/30",
              line.prefix === "deploy" &&
                "bg-blue-500/20 text-blue-400 border border-blue-500/30",
            )}
          >
            {line.prefix}
          </span>
        )}

        {line.step && (
          <span
            className={cn(
              "shrink-0 font-mono text-amber-400/80",
              compact ? "text-[10px] leading-4" : "text-xs leading-5",
            )}
          >
            {line.step}
          </span>
        )}

        <span
          className={cn(
            "whitespace-pre-wrap break-all min-w-0",
            compact ? "text-xs leading-4" : "text-sm leading-5",
            line.type === "error" && "text-red-400 font-medium",
            line.type === "success" && "text-emerald-400",
            line.type === "warning" && "text-yellow-400",
            line.type === "info" && "text-sky-400",
            line.type === "build" && "text-slate-400",
            line.type === "default" && "text-slate-300",
          )}
        >
          {highlightText(line.content, searchTerm ?? "")}
        </span>
      </div>
    </div>
  );
}

interface LogContentProps {
  readonly lines: readonly ParsedLogLine[];
  readonly scrollRef: React.RefObject<HTMLDivElement>;
  readonly className?: string;
  readonly searchTerm?: string;
  readonly currentMatchIndex?: number;
  readonly matchingLineNumbers?: readonly number[];
  readonly compact?: boolean;
}

function LogContent({
  lines,
  scrollRef,
  className,
  searchTerm,
  currentMatchIndex,
  matchingLineNumbers = [],
  compact = false,
}: LogContentProps) {
  const visibleLines = lines.filter((line) => !line.isEmpty);
  const currentMatchLine =
    currentMatchIndex !== undefined && matchingLineNumbers.length > 0
      ? matchingLineNumbers[currentMatchIndex]
      : undefined;

  return (
    <div
      ref={scrollRef}
      className={cn(
        "overflow-auto rounded-lg scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent",
        className,
      )}
    >
      <div className={cn("font-mono space-y-0", compact ? "p-2" : "p-3")}>
        {visibleLines.map((line) => (
          <LogLine
            key={line.lineNumber}
            line={line}
            searchTerm={searchTerm}
            isCurrentMatch={line.lineNumber === currentMatchLine}
            compact={compact}
          />
        ))}
      </div>
    </div>
  );
}

export function LogViewer({
  logs,
  autoScroll = true,
  title = "Logs",
}: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const expandedScrollRef = useRef<HTMLDivElement>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const [filters, setFilters] = useState<LogFilters>({
    search: "",
    types: [...ALL_FILTER_TYPES],
  });
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0);
  const [showSearch, setShowSearch] = useState(false);

  const allLines = useMemo(
    () =>
      logs
        ?.split("\n")
        .filter(Boolean)
        .map((line, index) => parseLogLine(line, index)) ?? [],
    [logs],
  );

  const filteredLines = useMemo(() => {
    return allLines.filter((line) => {
      if (line.isEmpty) return false;
      if (!filters.types.includes(line.type)) return false;
      return true;
    });
  }, [allLines, filters.types]);

  const matchingLineNumbers = useMemo(() => {
    if (!filters.search) return [];
    const searchLower = filters.search.toLowerCase();
    return filteredLines
      .filter((line) => line.content.toLowerCase().includes(searchLower))
      .map((line) => line.lineNumber);
  }, [filteredLines, filters.search]);

  const scrollToMatch = useCallback(
    (index: number, ref: React.RefObject<HTMLDivElement>) => {
      if (!ref.current || matchingLineNumbers.length === 0) return;

      const lineNumber = matchingLineNumbers[index];
      const element = ref.current.querySelector(
        `[data-line-number="${lineNumber}"]`,
      );
      if (element) {
        element.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    },
    [matchingLineNumbers],
  );

  const goToNextMatch = useCallback(() => {
    if (matchingLineNumbers.length === 0) return;
    const nextIndex = (currentMatchIndex + 1) % matchingLineNumbers.length;
    setCurrentMatchIndex(nextIndex);
    scrollToMatch(nextIndex, isExpanded ? expandedScrollRef : scrollRef);
  }, [
    currentMatchIndex,
    matchingLineNumbers.length,
    isExpanded,
    scrollToMatch,
  ]);

  const goToPreviousMatch = useCallback(() => {
    if (matchingLineNumbers.length === 0) return;
    const prevIndex =
      (currentMatchIndex - 1 + matchingLineNumbers.length) %
      matchingLineNumbers.length;
    setCurrentMatchIndex(prevIndex);
    scrollToMatch(prevIndex, isExpanded ? expandedScrollRef : scrollRef);
  }, [
    currentMatchIndex,
    matchingLineNumbers.length,
    isExpanded,
    scrollToMatch,
  ]);

  useEffect(() => {
    setCurrentMatchIndex(0);
  }, [filters.search]);

  useEffect(() => {
    if (autoScroll && !filters.search && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll, filters.search]);

  useEffect(() => {
    if (
      autoScroll &&
      !filters.search &&
      isExpanded &&
      expandedScrollRef.current
    ) {
      expandedScrollRef.current.scrollTop =
        expandedScrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll, isExpanded, filters.search]);

  const handleCopy = useCallback(async () => {
    if (!logs) return;
    try {
      await navigator.clipboard.writeText(logs);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      console.error("Failed to copy logs");
    }
  }, [logs]);

  const toggleTypeFilter = useCallback((type: LogType) => {
    setFilters((prev) => ({
      ...prev,
      types: prev.types.includes(type)
        ? prev.types.filter((t) => t !== type)
        : [...prev.types, type],
    }));
  }, []);

  const clearFilters = useCallback(() => {
    setFilters({ search: "", types: [...ALL_FILTER_TYPES] });
    setShowSearch(false);
  }, []);

  const hasActiveFilters =
    filters.search || filters.types.length !== ALL_FILTER_TYPES.length;

  if (!logs) {
    return (
      <EmptyState
        icon={Terminal}
        title="No logs available"
        description="Logs will appear here when a deployment starts."
      />
    );
  }

  const visibleCount = filteredLines.length;

  const renderSearchBar = (inDialog = false) => (
    <div
      className={cn(
        "flex items-center gap-2 flex-wrap",
        inDialog ? "flex-1" : "mb-2",
      )}
    >
      {showSearch && (
        <div className="relative flex items-center">
          <Search className="absolute left-2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            placeholder="Search logs..."
            value={filters.search}
            onChange={(e) =>
              setFilters((prev) => ({ ...prev, search: e.target.value }))
            }
            className="h-8 w-48 pl-8 pr-8 text-sm bg-slate-900 border-slate-700"
          />
          {filters.search && (
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 h-6 w-6"
              onClick={() => setFilters((prev) => ({ ...prev, search: "" }))}
            >
              <X className="h-3 w-3" />
            </Button>
          )}
        </div>
      )}

      {filters.search && matchingLineNumbers.length > 0 && (
        <div className="flex items-center gap-1">
          <span className="text-xs text-muted-foreground">
            {currentMatchIndex + 1}/{matchingLineNumbers.length}
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={goToPreviousMatch}
            title="Previous match"
          >
            <ChevronUp className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={goToNextMatch}
            title="Next match"
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </Button>
        </div>
      )}

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 gap-1",
              filters.types.length !== ALL_FILTER_TYPES.length &&
                "text-yellow-400",
            )}
          >
            <Filter className="h-3.5 w-3.5" />
            <span className="text-xs">Filter</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {ALL_FILTER_TYPES.map((type) => (
            <DropdownMenuCheckboxItem
              key={type}
              checked={filters.types.includes(type)}
              onCheckedChange={() => toggleTypeFilter(type)}
            >
              {FILTER_LABELS[type]}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {hasActiveFilters && (
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          onClick={clearFilters}
        >
          Clear
        </Button>
      )}
    </div>
  );

  const renderControls = (inDialog = false) => (
    <div className="flex items-center gap-1">
      <Button
        variant="ghost"
        size="icon"
        className={cn(
          "h-7 w-7",
          inDialog
            ? "hover:bg-slate-800"
            : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
          showSearch && "text-yellow-400",
        )}
        onClick={() => setShowSearch(!showSearch)}
        title="Search logs"
      >
        <Search className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className={cn(
          "h-7 w-7",
          inDialog
            ? "hover:bg-slate-800"
            : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
        )}
        onClick={handleCopy}
        title="Copy logs"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-green-400" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
      </Button>
      {!inDialog && (
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 bg-black/50 hover:bg-black/70 backdrop-blur-sm"
          onClick={() => setIsExpanded(true)}
          title="Expand logs"
        >
          <Expand className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  );

  return (
    <>
      <div className="space-y-2">
        {showSearch && renderSearchBar()}
        <div className="relative">
          <div className="absolute top-2 right-2 z-10 flex gap-1">
            {renderControls()}
          </div>
          <LogContent
            lines={filteredLines}
            scrollRef={scrollRef}
            className="h-[450px] border border-slate-800 bg-slate-950"
            searchTerm={filters.search}
            currentMatchIndex={currentMatchIndex}
            matchingLineNumbers={matchingLineNumbers}
            compact
          />
        </div>
      </div>

      <Dialog open={isExpanded} onOpenChange={setIsExpanded}>
        <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] flex flex-col p-0 gap-0 bg-slate-950 border-slate-800">
          <DialogHeader className="px-4 py-3 border-b border-slate-800 space-y-2">
            <div className="flex items-center justify-between">
              <DialogTitle className="flex items-center gap-2 text-base">
                <Terminal className="h-4 w-4" />
                {title}
                <span className="text-xs text-muted-foreground font-normal">
                  ({visibleCount} lines)
                </span>
              </DialogTitle>
              <div className="flex items-center gap-1">
                {renderControls(true)}
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 hover:bg-slate-800"
                  onClick={() => setIsExpanded(false)}
                  title="Minimize"
                >
                  <Minimize2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            {showSearch && renderSearchBar(true)}
          </DialogHeader>
          <LogContent
            lines={filteredLines}
            scrollRef={expandedScrollRef}
            className="flex-1 min-h-0"
            searchTerm={filters.search}
            currentMatchIndex={currentMatchIndex}
            matchingLineNumbers={matchingLineNumbers}
          />
        </DialogContent>
      </Dialog>
    </>
  );
}
