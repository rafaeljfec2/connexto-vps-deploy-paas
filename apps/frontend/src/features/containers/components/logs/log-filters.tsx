import { Filter, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import type { TimePreset } from "./hooks/use-log-filters";
import type { LogLevel } from "./types";
import { LOG_LEVELS } from "./types";

interface LogFiltersProps {
  readonly levelFilter: readonly LogLevel[];
  readonly onLevelsChange: (levels: LogLevel[]) => void;
  readonly contextFilter: readonly string[];
  readonly availableContexts: readonly string[];
  readonly onContextsChange: (contexts: string[]) => void;
  readonly timePreset: TimePreset | null;
  readonly onTimePresetChange: (preset: TimePreset) => void;
  readonly hasActiveFilters: boolean;
  readonly onClear: () => void;
  readonly className?: string;
}

const LEVEL_LABELS: Record<LogLevel, string> = {
  trace: "Trace",
  debug: "Debug",
  info: "Info",
  warn: "Warn",
  error: "Error",
  fatal: "Fatal",
};

const TIME_PRESETS: { value: TimePreset; label: string }[] = [
  { value: "all", label: "All" },
  { value: "5m", label: "Last 5m" },
  { value: "15m", label: "Last 15m" },
  { value: "1h", label: "Last 1h" },
  { value: "24h", label: "Last 24h" },
];

export function LogFilters({
  levelFilter,
  onLevelsChange,
  contextFilter,
  availableContexts,
  onContextsChange,
  timePreset,
  onTimePresetChange,
  onClear,
  hasActiveFilters,
  className,
}: LogFiltersProps) {
  const toggleLevel = (level: LogLevel) => {
    if (levelFilter.includes(level)) {
      onLevelsChange(levelFilter.filter((l) => l !== level));
    } else {
      onLevelsChange([...levelFilter, level]);
    }
  };

  const toggleContext = (ctx: string) => {
    if (contextFilter.includes(ctx)) {
      onContextsChange(contextFilter.filter((c) => c !== ctx));
    } else {
      onContextsChange([...contextFilter, ctx]);
    }
  };

  const levelLabel =
    levelFilter.length === 0 || levelFilter.length === LOG_LEVELS.length
      ? "All levels"
      : `${levelFilter.length} level(s)`;
  const contextLabel =
    contextFilter.length === 0
      ? "All contexts"
      : `${contextFilter.length} context(s)`;
  const timeLabel =
    timePreset == null
      ? "All time"
      : (TIME_PRESETS.find((p) => p.value === timePreset)?.label ?? "All time");

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 border-b border-slate-800 bg-slate-900/50 px-3 py-2",
        className,
      )}
    >
      <Filter className="h-4 w-4 text-slate-500 shrink-0" aria-hidden />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs border-slate-700 bg-slate-800/50 hover:bg-slate-700/50"
          >
            {levelLabel}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-48">
          <DropdownMenuLabel>Log level</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {LOG_LEVELS.map((level) => (
            <DropdownMenuCheckboxItem
              key={level}
              checked={levelFilter.length === 0 || levelFilter.includes(level)}
              onCheckedChange={() => toggleLevel(level)}
            >
              {LEVEL_LABELS[level]}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs border-slate-700 bg-slate-800/50 hover:bg-slate-700/50"
          >
            {contextLabel}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="max-h-64 overflow-auto">
          <DropdownMenuLabel>Context</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {availableContexts.length === 0 ? (
            <div className="px-2 py-4 text-xs text-slate-500">
              No contexts in logs
            </div>
          ) : (
            availableContexts.map((ctx) => (
              <DropdownMenuCheckboxItem
                key={ctx}
                checked={
                  contextFilter.length === 0 || contextFilter.includes(ctx)
                }
                onCheckedChange={() => toggleContext(ctx)}
              >
                <span className="truncate max-w-[200px]" title={ctx}>
                  {ctx}
                </span>
              </DropdownMenuCheckboxItem>
            ))
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs border-slate-700 bg-slate-800/50 hover:bg-slate-700/50"
          >
            {timeLabel}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuLabel>Time range</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {TIME_PRESETS.map((preset) => (
            <DropdownMenuCheckboxItem
              key={preset.value}
              checked={timePreset === preset.value}
              onCheckedChange={() => onTimePresetChange(preset.value)}
            >
              {preset.label}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      {hasActiveFilters && (
        <Button
          variant="ghost"
          size="sm"
          className="h-8 text-xs text-slate-400 hover:text-slate-200"
          onClick={onClear}
        >
          <X className="h-3.5 w-3.5 mr-1" />
          Clear
        </Button>
      )}
    </div>
  );
}
