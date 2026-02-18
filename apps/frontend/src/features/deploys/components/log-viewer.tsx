import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/empty-state";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  type DeployLogLine,
  type DeployLogType,
  parseDeployLogLine,
} from "@/lib/log-utils";
import { cn } from "@/lib/utils";
import { LogLine } from "./log-line";
import { ALL_FILTER_TYPES, LogSearchBar, LogToolbar } from "./log-toolbar";

interface LogViewerProps {
  readonly logs: string | null;
  readonly autoScroll?: boolean;
  readonly title?: string;
}

interface LogFilters {
  readonly search: string;
  readonly types: readonly DeployLogType[];
}

interface LogContentProps {
  readonly lines: readonly DeployLogLine[];
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
  const { copy, copied } = useCopyToClipboard();
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
        .map((line, index) => parseDeployLogLine(line, index)) ?? [],
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

  const handleCopy = useCallback(() => {
    if (logs) copy(logs);
  }, [logs, copy]);

  const toggleTypeFilter = useCallback((type: DeployLogType) => {
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

  return (
    <>
      <div className="space-y-2">
        {showSearch && (
          <LogSearchBar
            search={filters.search}
            onSearchChange={(v) =>
              setFilters((prev) => ({ ...prev, search: v }))
            }
            filterTypes={filters.types}
            onFilterTypeToggle={toggleTypeFilter}
            onClearFilters={clearFilters}
            matchingLineNumbers={matchingLineNumbers}
            currentMatchIndex={currentMatchIndex}
            onPreviousMatch={goToPreviousMatch}
            onNextMatch={goToNextMatch}
          />
        )}
        <div className="relative">
          <div className="absolute top-2 right-2 z-10 flex gap-1">
            <LogToolbar
              search={filters.search}
              onSearchChange={(v) =>
                setFilters((prev) => ({ ...prev, search: v }))
              }
              showSearch={showSearch}
              onShowSearchToggle={() => setShowSearch(!showSearch)}
              filterTypes={filters.types}
              onFilterTypeToggle={toggleTypeFilter}
              onClearFilters={clearFilters}
              matchingLineNumbers={matchingLineNumbers}
              currentMatchIndex={currentMatchIndex}
              onPreviousMatch={goToPreviousMatch}
              onNextMatch={goToNextMatch}
              onCopy={handleCopy}
              copied={copied}
              onExpand={() => setIsExpanded(true)}
            />
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
                <LogToolbar
                  search={filters.search}
                  onSearchChange={(v) =>
                    setFilters((prev) => ({ ...prev, search: v }))
                  }
                  showSearch={showSearch}
                  onShowSearchToggle={() => setShowSearch(!showSearch)}
                  filterTypes={filters.types}
                  onFilterTypeToggle={toggleTypeFilter}
                  onClearFilters={clearFilters}
                  matchingLineNumbers={matchingLineNumbers}
                  currentMatchIndex={currentMatchIndex}
                  onPreviousMatch={goToPreviousMatch}
                  onNextMatch={goToNextMatch}
                  onCopy={handleCopy}
                  copied={copied}
                  onMinimize={() => setIsExpanded(false)}
                  inDialog
                />
              </div>
            </div>
            {showSearch && (
              <LogSearchBar
                search={filters.search}
                onSearchChange={(v) =>
                  setFilters((prev) => ({ ...prev, search: v }))
                }
                filterTypes={filters.types}
                onFilterTypeToggle={toggleTypeFilter}
                onClearFilters={clearFilters}
                matchingLineNumbers={matchingLineNumbers}
                currentMatchIndex={currentMatchIndex}
                onPreviousMatch={goToPreviousMatch}
                onNextMatch={goToNextMatch}
                inDialog
              />
            )}
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
