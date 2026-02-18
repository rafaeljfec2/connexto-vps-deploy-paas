import { useCallback, useEffect, useRef, useState } from "react";
import { Check, Copy, Pause, Play, RefreshCw, Terminal } from "lucide-react";
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
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { type ContainerLogLine, parseContainerLogLine } from "@/lib/log-utils";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { useContainerLogs } from "../hooks/use-containers";

interface ContainerLogsDialogProps {
  readonly containerId: string | null;
  readonly containerName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
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

const TAIL_OPTIONS = [
  { value: 100, label: "Last 100 lines" },
  { value: 500, label: "Last 500 lines" },
  { value: 1000, label: "Last 1000 lines" },
  { value: 5000, label: "Last 5000 lines" },
] as const;

interface LogsContentProps {
  readonly isLoading: boolean;
  readonly hasLogs: boolean;
  readonly isStreaming: boolean;
  readonly lines: readonly ContainerLogLine[];
  readonly scrollRef: React.RefObject<HTMLDivElement>;
}

function LogsContent({
  isLoading,
  hasLogs,
  isStreaming,
  lines,
  scrollRef,
}: LogsContentProps) {
  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!hasLogs && !isStreaming) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <EmptyState
          icon={Terminal}
          title="No container logs"
          description="Container logs will appear here when the container is running."
        />
      </div>
    );
  }

  return (
    <div
      ref={scrollRef}
      className="flex-1 min-h-0 overflow-auto scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent"
    >
      <div className="p-3 font-mono space-y-0">
        {lines.map((line) => (
          <LogLine key={line.lineNumber} line={line} />
        ))}
      </div>
    </div>
  );
}

export function ContainerLogsDialog({
  containerId,
  containerName,
  open,
  onOpenChange,
}: ContainerLogsDialogProps) {
  const [tail, setTail] = useState(100);
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamedLogs, setStreamedLogs] = useState<string[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const { copy, copied } = useCopyToClipboard();

  const scrollRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const {
    data: logsData,
    refetch,
    isLoading,
  } = useContainerLogs(open ? (containerId ?? undefined) : undefined, tail);

  const startStreaming = useCallback(() => {
    if (!containerId) return;

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const url = api.containers.logsStreamUrl(containerId);
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
  }, [containerId]);

  const stopStreaming = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsStreaming(false);
  }, []);

  useEffect(() => {
    if (!open) {
      stopStreaming();
    }
  }, [open, stopStreaming]);

  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [streamedLogs, logsData, autoScroll]);

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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl h-[80vh] flex flex-col p-0 gap-0 bg-slate-950 border-slate-800">
        <DialogHeader className="px-4 py-3 border-b border-slate-800 flex-row items-center justify-between space-y-0">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Terminal className="h-4 w-4" />
            Container Logs - {containerName}
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
            {!isStreaming && (
              <>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs hover:bg-slate-800"
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
                  className="h-7 w-7 hover:bg-slate-800"
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
                "h-7 w-7 hover:bg-slate-800",
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
                  "h-7 w-7 hover:bg-slate-800",
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
          </div>
        </DialogHeader>

        <LogsContent
          isLoading={isLoading}
          hasLogs={hasLogs}
          isStreaming={isStreaming}
          lines={lines}
          scrollRef={scrollRef}
        />
      </DialogContent>
    </Dialog>
  );
}
