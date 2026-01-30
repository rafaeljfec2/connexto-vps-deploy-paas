import { useCallback, useEffect, useRef, useState } from "react";
import { Check, Copy, Expand, Minimize2, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/empty-state";
import { cn } from "@/lib/utils";

interface LogViewerProps {
  readonly logs: string | null;
  readonly autoScroll?: boolean;
  readonly title?: string;
}

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

function LogLine({ line }: { readonly line: ParsedLogLine }) {
  if (line.isEmpty) {
    return null;
  }

  return (
    <div className="flex group hover:bg-white/5 transition-colors py-0.5">
      <span className="select-none text-muted-foreground/40 w-8 text-right mr-3 shrink-0 text-xs leading-5 tabular-nums">
        {line.lineNumber}
      </span>

      <div className="flex items-start gap-2 min-w-0 flex-1">
        {line.timestamp && (
          <span className="text-slate-500 shrink-0 font-medium text-xs leading-5 tabular-nums">
            {line.timestamp}
          </span>
        )}

        {line.prefix && (
          <span
            className={cn(
              "shrink-0 text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded leading-none",
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
          <span className="shrink-0 text-xs font-mono text-amber-400/80 leading-5">
            {line.step}
          </span>
        )}

        <span
          className={cn(
            "whitespace-pre-wrap break-all text-sm leading-5 min-w-0",
            line.type === "error" && "text-red-400 font-medium",
            line.type === "success" && "text-emerald-400",
            line.type === "warning" && "text-yellow-400",
            line.type === "info" && "text-sky-400",
            line.type === "build" && "text-slate-400",
            line.type === "default" && "text-slate-300",
          )}
        >
          {line.content}
        </span>
      </div>
    </div>
  );
}

interface LogContentProps {
  readonly lines: readonly ParsedLogLine[];
  readonly scrollRef: React.RefObject<HTMLDivElement>;
  readonly className?: string;
}

function LogContent({ lines, scrollRef, className }: LogContentProps) {
  const visibleLines = lines.filter((line) => !line.isEmpty);

  return (
    <div
      ref={scrollRef}
      className={cn(
        "overflow-auto rounded-lg scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent",
        className,
      )}
    >
      <div className="p-3 font-mono space-y-0">
        {visibleLines.map((line) => (
          <LogLine key={line.lineNumber} line={line} />
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

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  useEffect(() => {
    if (autoScroll && isExpanded && expandedScrollRef.current) {
      expandedScrollRef.current.scrollTop =
        expandedScrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll, isExpanded]);

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

  if (!logs) {
    return (
      <EmptyState
        icon={Terminal}
        title="No logs available"
        description="Logs will appear here when a deployment starts."
      />
    );
  }

  const lines = logs
    .split("\n")
    .filter(Boolean)
    .map((line, index) => parseLogLine(line, index));

  const visibleCount = lines.filter((l) => !l.isEmpty).length;

  return (
    <>
      <div className="relative">
        <div className="absolute top-2 right-2 z-10 flex gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 bg-black/50 hover:bg-black/70 backdrop-blur-sm"
            onClick={handleCopy}
            title="Copy logs"
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-green-400" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 bg-black/50 hover:bg-black/70 backdrop-blur-sm"
            onClick={() => setIsExpanded(true)}
            title="Expand logs"
          >
            <Expand className="h-3.5 w-3.5" />
          </Button>
        </div>
        <LogContent
          lines={lines}
          scrollRef={scrollRef}
          className="h-[400px] border border-slate-800 bg-slate-950"
        />
      </div>

      <Dialog open={isExpanded} onOpenChange={setIsExpanded}>
        <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] flex flex-col p-0 gap-0 bg-slate-950 border-slate-800">
          <DialogHeader className="px-4 py-3 border-b border-slate-800 flex-row items-center justify-between space-y-0">
            <DialogTitle className="flex items-center gap-2 text-base">
              <Terminal className="h-4 w-4" />
              {title}
              <span className="text-xs text-muted-foreground font-normal">
                ({visibleCount} lines)
              </span>
            </DialogTitle>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 hover:bg-slate-800"
                onClick={handleCopy}
                title="Copy logs"
              >
                {copied ? (
                  <Check className="h-3.5 w-3.5 text-green-400" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </Button>
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
          </DialogHeader>
          <LogContent
            lines={lines}
            scrollRef={expandedScrollRef}
            className="flex-1 min-h-0"
          />
        </DialogContent>
      </Dialog>
    </>
  );
}
