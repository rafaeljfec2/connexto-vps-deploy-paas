import type { ContainerLogLine } from "@/lib/log-utils";
import { cn } from "@/lib/utils";

interface LogLineProps {
  readonly line: ContainerLogLine;
}

const typeStyles: Record<ContainerLogLine["type"], string> = {
  error: "text-red-400",
  warning: "text-yellow-400",
  info: "text-blue-400",
  default: "text-slate-300",
};

export function LogLine({ line }: LogLineProps) {
  return (
    <div className="flex gap-2 text-xs leading-5 hover:bg-slate-800/50">
      <span className="select-none text-slate-600 w-8 text-right shrink-0">
        {line.lineNumber}
      </span>
      {line.timestamp && (
        <span className="select-none text-slate-500 shrink-0">
          {line.timestamp}
        </span>
      )}
      <span
        className={cn("break-all whitespace-pre-wrap", typeStyles[line.type])}
      >
        {line.content}
      </span>
    </div>
  );
}
