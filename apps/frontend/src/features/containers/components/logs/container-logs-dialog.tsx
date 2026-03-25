import { useCallback, useState } from "react";
import { Terminal } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/empty-state";
import { useContainerLogs } from "../../hooks/use-containers";
import { useLogFilters } from "./hooks/use-log-filters";
import { useLogParser } from "./hooks/use-log-parser";
import { useLogSearch } from "./hooks/use-log-search";
import { LogFilters } from "./log-filters";
import { LogTable } from "./log-table";
import { LogToolbar } from "./log-toolbar";

interface ContainerLogsDialogProps {
  readonly containerId: string | null;
  readonly containerName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function ContainerLogsDialog({
  containerId,
  containerName,
  open,
  onOpenChange,
}: ContainerLogsDialogProps) {
  const [tail, setTail] = useState(100);
  const [isExpanded, setIsExpanded] = useState(false);
  const [autoScroll] = useState(true);
  const [copied, setCopied] = useState(false);

  const { data: logsData, isLoading } = useContainerLogs(
    open ? (containerId ?? undefined) : undefined,
    tail,
  );
  const rawLogs = logsData?.logs ?? null;
  const { entries, uniqueContexts } = useLogParser(rawLogs);
  const {
    filters,
    timePreset,
    setLevels,
    setContexts,
    setTimeRange,
    clearFilters,
    filterEntries,
    hasActiveFilters,
  } = useLogFilters(uniqueContexts);

  const filteredByLevelContextTime = filterEntries(entries);
  const {
    searchTerm,
    setSearchTerm,
    useRegex,
    setUseRegex,
    regex,
    currentMatchLine,
    currentMatchIndex,
    matchCount,
    goToNext,
    goToPrevious,
    hasSearch,
  } = useLogSearch(filteredByLevelContextTime);

  const handleCopy = useCallback(async () => {
    if (rawLogs == null) return;
    try {
      await navigator.clipboard.writeText(rawLogs);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // ignore
    }
  }, [rawLogs]);

  const hasLogs = filteredByLevelContextTime.length > 0;

  const dialogContent = (
    <>
      <LogToolbar
        searchTerm={searchTerm}
        onSearchChange={setSearchTerm}
        useRegex={useRegex}
        onUseRegexChange={setUseRegex}
        matchCount={matchCount}
        currentMatchIndex={currentMatchIndex}
        onNextMatch={goToNext}
        onPreviousMatch={goToPrevious}
        onCopy={handleCopy}
        copied={copied}
        tail={tail}
        onTailChange={setTail}
        isExpanded={isExpanded}
        onExpandToggle={() => setIsExpanded((e) => !e)}
        isInDialog
        className="shrink-0"
      />
      <LogFilters
        levelFilter={filters.levels}
        onLevelsChange={setLevels}
        contextFilter={filters.contexts}
        availableContexts={uniqueContexts}
        onContextsChange={setContexts}
        timePreset={timePreset}
        onTimePresetChange={setTimeRange}
        onClear={clearFilters}
        hasActiveFilters={hasActiveFilters}
        className="shrink-0"
      />
      {hasLogs ? (
        <LogTable
          entries={filteredByLevelContextTime}
          searchTerm={searchTerm}
          searchRegex={regex}
          currentMatchLine={currentMatchLine}
          autoScroll={autoScroll && !hasSearch}
          compact={false}
          className="flex-1 min-h-0"
        />
      ) : (
        <div className="flex-1 flex items-center justify-center min-h-[200px]">
          <EmptyState
            icon={Terminal}
            title="No logs"
            description={
              isLoading
                ? "Loading logs..."
                : "No log entries match the current filters or the container has no logs yet."
            }
          />
        </div>
      )}
    </>
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className="max-w-[95vw] w-[95vw] max-h-[85vh] flex flex-col p-0 gap-0 bg-slate-950 border-slate-800"
          aria-describedby={undefined}
        >
          <DialogHeader className="px-4 py-3 border-b border-slate-800 flex-row items-center justify-between space-y-0 shrink-0">
            <DialogTitle className="flex items-center gap-2 text-base">
              <Terminal className="h-4 w-4 shrink-0" />
              Container Logs - {containerName}
              <span className="text-xs text-muted-foreground font-normal">
                ({entries.length} lines)
              </span>
            </DialogTitle>
          </DialogHeader>
          <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
            {isLoading && !rawLogs ? (
              <div className="flex-1 flex items-center justify-center min-h-[240px]">
                <EmptyState
                  icon={Terminal}
                  title="Loading logs"
                  description="Fetching container logs..."
                />
              </div>
            ) : (
              dialogContent
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
