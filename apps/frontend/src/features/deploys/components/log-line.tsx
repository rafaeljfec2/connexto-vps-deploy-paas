import { type DeployLogLine, type DeployLogType } from "@/lib/log-utils";
import { cn } from "@/lib/utils";

export interface LogLineProps {
  readonly line: DeployLogLine;
  readonly searchTerm?: string;
  readonly isCurrentMatch?: boolean;
  readonly compact?: boolean;
}

const typeColorMap: Record<DeployLogType, string> = {
  error: "text-red-400 font-medium",
  success: "text-emerald-400",
  warning: "text-yellow-400",
  info: "text-sky-400",
  build: "text-slate-400",
  default: "text-slate-300",
};

const prefixStyleMap: Record<string, string> = {
  build: "bg-violet-500/20 text-violet-400 border border-violet-500/30",
  deploy: "bg-blue-500/20 text-blue-400 border border-blue-500/30",
};

function highlightText(text: string, searchTerm: string): React.ReactNode {
  if (!searchTerm) return text;

  const escapedTerm = searchTerm.replaceAll(
    /[.*+?^${}()|[\]\\]/g,
    String.raw`\$&`,
  );
  const regex = new RegExp(`(${escapedTerm})`, "gi");
  const parts = text.split(regex);

  return parts.map((part, index) => {
    const isMatch = regex.test(part);
    regex.lastIndex = 0;
    return isMatch ? (
      <mark
        key={`${part}-${index}`}
        className="bg-yellow-500/40 text-inherit rounded px-0.5"
      >
        {part}
      </mark>
    ) : (
      <span key={`${part}-${index}`}>{part}</span>
    );
  });
}

export function LogLine({
  line,
  searchTerm,
  isCurrentMatch,
  compact,
}: LogLineProps) {
  if (line.isEmpty) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex group hover:bg-white/5 transition-colors",
        compact ? "py-px" : "py-0.5",
        isCurrentMatch && "bg-yellow-500/20 ring-1 ring-yellow-500/50",
      )}
      data-line-number={line.lineNumber}
    >
      <span
        className={cn(
          "select-none text-muted-foreground/40 text-right mr-2 shrink-0 tabular-nums",
          compact ? "w-6 text-[10px] leading-4" : "w-8 text-xs leading-5 mr-3",
        )}
      >
        {line.lineNumber}
      </span>

      <div
        className={cn(
          "flex items-start min-w-0 flex-1",
          compact ? "gap-1.5" : "gap-2",
        )}
      >
        {line.timestamp && (
          <span
            className={cn(
              "text-slate-500 shrink-0 font-medium tabular-nums",
              compact ? "text-[10px] leading-4" : "text-xs leading-5",
            )}
          >
            {line.timestamp}
          </span>
        )}

        {line.prefix && (
          <span
            className={cn(
              "shrink-0 font-semibold uppercase tracking-wider rounded leading-none",
              compact ? "text-[8px] px-1 py-px" : "text-[10px] px-1.5 py-0.5",
              prefixStyleMap[line.prefix],
            )}
          >
            {line.prefix}
          </span>
        )}

        {line.step && (
          <span
            className={cn(
              "shrink-0 font-mono text-amber-400/80",
              compact ? "text-[10px] leading-4" : "text-xs leading-5",
            )}
          >
            {line.step}
          </span>
        )}

        <span
          className={cn(
            "whitespace-pre-wrap break-all min-w-0",
            compact ? "text-xs leading-4" : "text-sm leading-5",
            typeColorMap[line.type],
          )}
        >
          {highlightText(line.content, searchTerm ?? "")}
        </span>
      </div>
    </div>
  );
}
