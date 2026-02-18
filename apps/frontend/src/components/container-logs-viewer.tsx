import { useCallback, useEffect, useRef, useState } from "react";
import {
  Check,
  Copy,
  Expand,
  Minimize2,
  Pause,
  Play,
  RefreshCw,
  Terminal,
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
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EmptyState } from "@/components/empty-state";
import { useContainerLogs } from "@/features/apps/hooks/use-apps";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { type ContainerLogLine, parseContainerLogLine } from "@/lib/log-utils";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";

interface ContainerLogsViewerProps {
  readonly appId: string;
  readonly appName: string;
}

function LogLine({ line }: { readonly line: ContainerLogLine }) {
  if (!line.content) {
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

        <span
          className={cn(
            "whitespace-pre-wrap break-all text-sm leading-5 min-w-0",
            line.type === "error" && "text-red-400 font-medium",
            line.type === "warning" && "text-yellow-400",
            line.type === "info" && "text-sky-400",
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
  readonly lines: readonly ContainerLogLine[];
  readonly scrollRef: React.RefObject<HTMLDivElement>;
  readonly className?: string;
}

function LogContent({ lines, scrollRef, className }: LogContentProps) {
  return (
    <div
      ref={scrollRef}
      className={cn(
        "overflow-auto rounded-lg scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent",
        className,
      )}
    >
      <div className="p-3 font-mono space-y-0">
        {lines.map((line) => (
          <LogLine key={line.lineNumber} line={line} />
        ))}
      </div>
    </div>
  );
}

const TAIL_OPTIONS = [
  { value: 100, label: "Last 100 lines" },
  { value: 500, label: "Last 500 lines" },
  { value: 1000, label: "Last 1000 lines" },
  { value: 5000, label: "Last 5000 lines" },
] as const;

export function ContainerLogsViewer({
  appId,
  appName,
}: ContainerLogsViewerProps) {
  const [tail, setTail] = useState(100);
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamedLogs, setStreamedLogs] = useState<string[]>([]);
  const [isExpanded, setIsExpanded] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const { copy, copied } = useCopyToClipboard();

  const scrollRef = useRef<HTMLDivElement>(null);
  const expandedScrollRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const { data: logsData, refetch, isLoading } = useContainerLogs(appId, tail);

  const startStreaming = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const url = api.container.logsStreamUrl(appId);
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      setStreamedLogs((prev) => [...prev, event.data]);
    };

    eventSource.onerror = () => {
      setIsStreaming(false);
      eventSource.close();
    };

    eventSourceRef.current = eventSource;
    setIsStreaming(true);
    setStreamedLogs([]);
  }, [appId]);

  const stopStreaming = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsStreaming(false);
  }, []);

  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  useEffect(() => {
    const ref = isExpanded ? expandedScrollRef.current : scrollRef.current;
    if (autoScroll && ref) {
      ref.scrollTop = ref.scrollHeight;
    }
  }, [streamedLogs, logsData, autoScroll, isExpanded]);

  const handleCopy = useCallback(() => {
    const text = isStreaming ? streamedLogs.join("\n") : (logsData?.logs ?? "");
    copy(text);
  }, [isStreaming, streamedLogs, logsData, copy]);

  const logs = isStreaming ? streamedLogs.join("\n") : (logsData?.logs ?? "");
  const lines = logs
    .split("\n")
    .filter(Boolean)
    .map((line, index) => parseContainerLogLine(line, index));

  const hasLogs = lines.length > 0;

  const renderControls = (inDialog = false) => (
    <div className={cn("flex items-center gap-1", inDialog && "mr-2")}>
      {!isStreaming && (
        <>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className={cn(
                  "h-7 text-xs",
                  inDialog
                    ? "hover:bg-slate-800"
                    : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
                )}
              >
                {tail} lines
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {TAIL_OPTIONS.map((option) => (
                <DropdownMenuItem
                  key={option.value}
                  onClick={() => setTail(option.value)}
                >
                  {option.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "h-7 w-7",
              inDialog
                ? "hover:bg-slate-800"
                : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
            )}
            onClick={() => refetch()}
            title="Refresh logs"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        </>
      )}
      <Button
        variant="ghost"
        size="icon"
        className={cn(
          "h-7 w-7",
          inDialog
            ? "hover:bg-slate-800"
            : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
          isStreaming && "text-green-400",
        )}
        onClick={isStreaming ? stopStreaming : startStreaming}
        title={isStreaming ? "Stop streaming" : "Start streaming"}
      >
        {isStreaming ? (
          <Pause className="h-3.5 w-3.5" />
        ) : (
          <Play className="h-3.5 w-3.5" />
        )}
      </Button>
      {isStreaming && (
        <Button
          variant="ghost"
          size="icon"
          className={cn(
            "h-7 w-7",
            inDialog
              ? "hover:bg-slate-800"
              : "bg-black/50 hover:bg-black/70 backdrop-blur-sm",
            !autoScroll && "text-yellow-400",
          )}
          onClick={() => setAutoScroll(!autoScroll)}
          title={autoScroll ? "Pause auto-scroll" : "Resume auto-scroll"}
        >
          {autoScroll ? (
            <Pause className="h-3.5 w-3.5" />
          ) : (
            <Play className="h-3.5 w-3.5" />
          )}
        </Button>
      )}
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

  if (isLoading) {
    return (
      <div className="h-[300px] border border-slate-800 bg-slate-950 rounded-lg flex items-center justify-center">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!hasLogs && !isStreaming) {
    return (
      <div className="relative">
        <div className="absolute top-2 right-2 z-10">{renderControls()}</div>
        <EmptyState
          icon={Terminal}
          title="No container logs"
          description="Container logs will appear here when the container is running."
        />
      </div>
    );
  }

  return (
    <>
      <div className="relative">
        <div className="absolute top-2 right-2 z-10 flex gap-1">
          {renderControls()}
        </div>
        <LogContent
          lines={lines}
          scrollRef={scrollRef}
          className="h-[300px] border border-slate-800 bg-slate-950"
        />
        {isStreaming && (
          <div className="absolute bottom-2 left-2 flex items-center gap-2 text-xs text-green-400">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
            </span>{" "}
            Live streaming
          </div>
        )}
      </div>

      <Dialog open={isExpanded} onOpenChange={setIsExpanded}>
        <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] flex flex-col p-0 gap-0 bg-slate-950 border-slate-800">
          <DialogHeader className="px-4 py-3 border-b border-slate-800 flex-row items-center justify-between space-y-0">
            <DialogTitle className="flex items-center gap-2 text-base">
              <Terminal className="h-4 w-4" />
              Container Logs - {appName}
              <span className="text-xs text-muted-foreground font-normal">
                ({lines.length} lines)
              </span>
              {isStreaming && (
                <span className="flex items-center gap-1 text-xs text-green-400 font-normal">
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
                  </span>{" "}
                  Live
                </span>
              )}
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
