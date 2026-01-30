import { useEffect, useRef } from "react";
import { Terminal } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { EmptyState } from "@/components/empty-state";

interface LogViewerProps {
  readonly logs: string | null;
  readonly autoScroll?: boolean;
}

export function LogViewer({ logs, autoScroll = true }: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  if (!logs) {
    return (
      <EmptyState
        icon={Terminal}
        title="No logs available"
        description="Logs will appear here when a deployment starts."
      />
    );
  }

  const lines = logs.split("\n").filter(Boolean);

  return (
    <ScrollArea
      className="h-[400px] rounded-md border bg-black/50"
      ref={scrollRef}
    >
      <div className="p-4 font-mono text-sm">
        {lines.map((line, index) => (
          <div key={index} className="flex">
            <span className="select-none text-muted-foreground w-10 text-right mr-4">
              {index + 1}
            </span>
            <span className="text-foreground whitespace-pre-wrap break-all">
              {line}
            </span>
          </div>
        ))}
      </div>
    </ScrollArea>
  );
}
