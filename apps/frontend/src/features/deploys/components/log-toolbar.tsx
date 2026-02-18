import {
  Check,
  ChevronDown,
  ChevronUp,
  Copy,
  Expand,
  Filter,
  Minimize2,
  Search,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import type { DeployLogType } from "@/lib/log-utils";
import { cn } from "@/lib/utils";

export const ALL_FILTER_TYPES: readonly DeployLogType[] = [
  "error",
  "warning",
  "success",
  "info",
  "build",
  "default",
] as const;

const FILTER_LABELS: Record<DeployLogType, string> = {
  error: "Errors",
  warning: "Warnings",
  success: "Success",
  info: "Info",
  build: "Build",
  default: "Other",
};

export interface LogToolbarProps {
  readonly showSearch: boolean;
  readonly onShowSearchToggle: () => void;
  readonly onCopy: () => void;
  readonly copied: boolean;
  readonly onExpand?: () => void;
  readonly onMinimize?: () => void;
  readonly inDialog?: boolean;
}

export function LogToolbar({
  showSearch,
  onShowSearchToggle,
  onCopy,
  copied,
  onExpand,
  onMinimize,
  inDialog = false,
}: LogToolbarProps) {
  if (inDialog) {
    return (
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          className={cn(
            "h-7 w-7",
            "hover:bg-slate-800",
            showSearch && "text-yellow-400",
          )}
          onClick={onShowSearchToggle}
          title="Search logs"
        >
          <Search className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 hover:bg-slate-800"
          onClick={onCopy}
          title="Copy logs"
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-green-400" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </Button>
        {onMinimize && (
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 hover:bg-slate-800"
            onClick={onMinimize}
            title="Minimize"
          >
            <Minimize2 className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="ghost"
        size="icon"
        className={cn(
          "h-7 w-7",
          "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
          showSearch && "text-yellow-400",
        )}
        onClick={onShowSearchToggle}
        title="Search logs"
      >
        <Search className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 bg-black/50 hover:bg-black/70 backdrop-blur-sm"
        onClick={onCopy}
        title="Copy logs"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-green-400" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
      </Button>
      {onExpand && (
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 bg-black/50 hover:bg-black/70 backdrop-blur-sm"
          onClick={onExpand}
          title="Expand logs"
        >
          <Expand className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  );
}

export interface LogSearchBarProps {
  readonly search: string;
  readonly onSearchChange: (value: string) => void;
  readonly filterTypes: readonly DeployLogType[];
  readonly onFilterTypeToggle: (type: DeployLogType) => void;
  readonly onClearFilters: () => void;
  readonly matchingLineNumbers: readonly number[];
  readonly currentMatchIndex: number;
  readonly onPreviousMatch: () => void;
  readonly onNextMatch: () => void;
  readonly inDialog?: boolean;
}

export function LogSearchBar({
  search,
  onSearchChange,
  filterTypes,
  onFilterTypeToggle,
  onClearFilters,
  matchingLineNumbers,
  currentMatchIndex,
  onPreviousMatch,
  onNextMatch,
  inDialog = false,
}: LogSearchBarProps) {
  const hasActiveFilters =
    search || filterTypes.length !== ALL_FILTER_TYPES.length;

  return (
    <div
      className={cn(
        "flex items-center gap-2 flex-wrap",
        inDialog ? "flex-1" : "mb-2",
      )}
    >
      <div className="relative flex items-center">
        <Search className="absolute left-2 h-3.5 w-3.5 text-muted-foreground" />
        <Input
          placeholder="Search logs..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="h-8 w-48 pl-8 pr-8 text-sm bg-slate-900 border-slate-700"
        />
        {search && (
          <Button
            variant="ghost"
            size="icon"
            className="absolute right-1 h-6 w-6"
            onClick={() => onSearchChange("")}
          >
            <X className="h-3 w-3" />
          </Button>
        )}
      </div>

      {search && matchingLineNumbers.length > 0 && (
        <div className="flex items-center gap-1">
          <span className="text-xs text-muted-foreground">
            {currentMatchIndex + 1}/{matchingLineNumbers.length}
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={onPreviousMatch}
            title="Previous match"
          >
            <ChevronUp className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={onNextMatch}
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
              filterTypes.length !== ALL_FILTER_TYPES.length &&
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
              checked={filterTypes.includes(type)}
              onCheckedChange={() => onFilterTypeToggle(type)}
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
          onClick={onClearFilters}
        >
          Clear
        </Button>
      )}
    </div>
  );
}
