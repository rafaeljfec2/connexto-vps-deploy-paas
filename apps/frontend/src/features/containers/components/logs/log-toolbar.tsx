import { useCallback, useEffect, useRef } from "react";
import {
  Check,
  ChevronDown,
  ChevronUp,
  Copy,
  Expand,
  Minimize2,
  Search,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

const TAIL_OPTIONS = [
  { value: 100, label: "Last 100 lines" },
  { value: 500, label: "Last 500 lines" },
  { value: 1000, label: "Last 1000 lines" },
  { value: 5000, label: "Last 5000 lines" },
] as const;

interface LogToolbarProps {
  readonly searchTerm: string;
  readonly onSearchChange: (value: string) => void;
  readonly useRegex: boolean;
  readonly onUseRegexChange: (value: boolean) => void;
  readonly matchCount: number;
  readonly currentMatchIndex: number;
  readonly onNextMatch: () => void;
  readonly onPreviousMatch: () => void;
  readonly onCopy: () => void;
  readonly copied: boolean;
  readonly tail: number;
  readonly onTailChange: (value: number) => void;
  readonly isExpanded: boolean;
  readonly onExpandToggle: () => void;
  readonly isStreaming?: boolean;
  readonly isInDialog?: boolean;
  readonly onKeyDown?: (e: React.KeyboardEvent) => void;
  readonly className?: string;
}

export function LogToolbar({
  searchTerm,
  onSearchChange,
  useRegex,
  onUseRegexChange,
  matchCount,
  currentMatchIndex,
  onNextMatch,
  onPreviousMatch,
  onCopy,
  copied,
  tail,
  onTailChange,
  isExpanded,
  onExpandToggle,
  isStreaming = false,
  isInDialog = false,
  onKeyDown,
  className,
}: LogToolbarProps) {
  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "f") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
      if (e.key === "Escape") {
        searchInputRef.current?.blur();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const handleSearchKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        if (e.shiftKey) {
          onPreviousMatch();
        } else {
          onNextMatch();
        }
      }
      onKeyDown?.(e);
    },
    [onNextMatch, onPreviousMatch, onKeyDown],
  );

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 px-3 py-2 border-b border-slate-800 bg-slate-900/50",
        className,
      )}
    >
      <div className="flex items-center gap-1 flex-1 min-w-[200px] max-w-md">
        <Search className="h-4 w-4 text-slate-500 shrink-0" aria-hidden />
        <Input
          ref={searchInputRef}
          type="text"
          placeholder="Search logs..."
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
          onKeyDown={handleSearchKeyDown}
          className={cn(
            "h-8 text-sm font-mono bg-slate-800 border-slate-700",
            isInDialog && "bg-slate-800/80",
          )}
          aria-label="Search logs"
        />
        {matchCount > 0 && (
          <span className="flex items-center gap-0.5 text-xs text-slate-500 shrink-0">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={onPreviousMatch}
              aria-label="Previous match"
            >
              <ChevronUp className="h-3.5 w-3.5" />
            </Button>
            <span className="tabular-nums min-w-[3ch]">
              {currentMatchIndex + 1}/{matchCount}
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={onNextMatch}
              aria-label="Next match"
            >
              <ChevronDown className="h-3.5 w-3.5" />
            </Button>
          </span>
        )}
      </div>
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn(
            "h-8 text-xs",
            isInDialog
              ? "hover:bg-slate-800"
              : "bg-slate-800/50 hover:bg-slate-700/50",
            useRegex && "text-sky-400",
          )}
          onClick={() => onUseRegexChange(!useRegex)}
          title={useRegex ? "Regex on" : "Regex off"}
        >
          .*
        </Button>
        {!isStreaming && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className={cn(
                  "h-8 text-xs",
                  isInDialog
                    ? "hover:bg-slate-800"
                    : "bg-slate-800/50 hover:bg-slate-700/50",
                )}
              >
                {tail} lines
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {TAIL_OPTIONS.map((opt) => (
                <DropdownMenuItem
                  key={opt.value}
                  onClick={() => onTailChange(opt.value)}
                >
                  {opt.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "h-8 w-8",
            isInDialog
              ? "hover:bg-slate-800"
              : "bg-slate-800/50 hover:bg-slate-700/50",
          )}
          onClick={onCopy}
          title="Copy logs"
          aria-label="Copy logs"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-400" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "h-8 w-8",
            isInDialog
              ? "hover:bg-slate-800"
              : "bg-slate-800/50 hover:bg-slate-700/50",
          )}
          onClick={onExpandToggle}
          title={isExpanded ? "Minimize" : "Expand"}
          aria-label={isExpanded ? "Minimize" : "Expand"}
        >
          {isExpanded ? (
            <Minimize2 className="h-4 w-4" />
          ) : (
            <Expand className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}
