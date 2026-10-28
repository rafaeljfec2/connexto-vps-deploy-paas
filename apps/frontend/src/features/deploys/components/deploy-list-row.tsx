import { GitBranch, MoreHorizontal, Terminal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatDuration, formatRelativeTime } from "@/lib/format";
import { cn, truncateCommitSha } from "@/lib/utils";
import type { Deployment } from "@/types";
import {
  type DeployStatusTone,
  getDeployDurationMs,
  getDeployStatusPresentation,
} from "../utils/deploy-status-label";

interface DeployListRowProps {
  readonly deployment: Deployment;
  readonly branch: string;
  readonly isCurrent?: boolean;
  readonly onOpenLogs: () => void;
}

const TONE_DOT_CLASS: Record<DeployStatusTone, string> = {
  success: "bg-emerald-500",
  error: "bg-red-500",
  running: "bg-amber-400 animate-pulse",
  muted: "bg-muted-foreground/60",
};

const TONE_TEXT_CLASS: Record<DeployStatusTone, string> = {
  success: "text-emerald-500",
  error: "text-red-500",
  running: "text-amber-500",
  muted: "text-muted-foreground",
};

export function DeployListRow({
  deployment,
  branch,
  isCurrent = false,
  onOpenLogs,
}: DeployListRowProps) {
  const { label, tone } = getDeployStatusPresentation(deployment.status);
  const durationMs = getDeployDurationMs(
    deployment.startedAt,
    deployment.finishedAt,
  );
  const trimmedMessage = deployment.commitMessage.trim();
  const shortSha = truncateCommitSha(deployment.commitSha);
  const message = trimmedMessage !== "" ? trimmedMessage : shortSha;
  const branchLabel = branch !== "" ? branch : "—";
  const appVersion = deployment.appVersion?.trim() ?? "";

  return (
    <div
      className={cn(
        "group relative grid grid-cols-1 gap-2 border-b border-border px-1 py-3 transition-colors",
        "hover:bg-muted/40",
        "sm:grid-cols-[minmax(0,1fr)_auto_auto_auto_auto_auto] sm:items-center sm:gap-4",
      )}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        aria-label={`View logs for deploy ${shortSha}`}
        onClick={onOpenLogs}
      />

      <div className="relative z-10 pointer-events-none min-w-0">
        <p className="truncate text-sm text-foreground/90">{message}</p>
      </div>

      <div className="relative z-10 pointer-events-none flex items-center gap-1.5 text-sm whitespace-nowrap">
        <span
          className={cn("h-2 w-2 shrink-0 rounded-full", TONE_DOT_CLASS[tone])}
          aria-hidden
        />
        <span className={cn("font-medium", TONE_TEXT_CLASS[tone])}>
          {label}
        </span>
        {durationMs !== null && (
          <span className="text-muted-foreground">
            {formatDuration(durationMs)}
          </span>
        )}
      </div>

      <div className="relative z-10 pointer-events-none flex items-center gap-1.5 whitespace-nowrap">
        <Badge
          variant={isCurrent ? "default" : "outline"}
          className={cn(
            "h-6 gap-1 px-2 text-xs font-medium",
            isCurrent &&
              "border-transparent bg-blue-600 text-white hover:bg-blue-600",
          )}
        >
          Production
        </Badge>
        {appVersion !== "" && (
          <Badge
            variant="secondary"
            className="h-6 px-2 font-mono text-xs font-medium"
          >
            v{appVersion}
          </Badge>
        )}
      </div>

      <div className="relative z-10 pointer-events-none flex items-center gap-1 font-mono text-xs text-muted-foreground whitespace-nowrap">
        <Terminal className="h-3.5 w-3.5 shrink-0" aria-hidden />
        <span>{shortSha}</span>
      </div>

      <div className="relative z-10 pointer-events-none flex items-center gap-1 text-xs text-muted-foreground whitespace-nowrap">
        <GitBranch className="h-3.5 w-3.5 shrink-0" aria-hidden />
        <span className="max-w-[8rem] truncate">{branchLabel}</span>
      </div>

      <div className="relative z-10 flex items-center justify-between gap-2 sm:justify-end">
        <span className="pointer-events-none text-xs text-muted-foreground whitespace-nowrap">
          {formatRelativeTime(deployment.createdAt)}
        </span>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="relative z-10 h-7 w-7 shrink-0 text-muted-foreground opacity-70 group-hover:opacity-100"
              aria-label="Deployment actions"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onSelect={onOpenLogs}>View logs</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
