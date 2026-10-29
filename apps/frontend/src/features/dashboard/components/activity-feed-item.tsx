import { Link } from "react-router-dom";
import { CheckCircle2, Circle, Clock, Loader2, XCircle } from "lucide-react";
import { formatRelativeTime } from "@/lib/format";
import { cn, truncateCommitSha } from "@/lib/utils";
import type { DeployStatus } from "@/types";

interface ActivityFeedItemProps {
  readonly appId: string;
  readonly appName: string;
  readonly status: DeployStatus;
  readonly commitSha: string;
  readonly commitMessage?: string;
  readonly timestamp: string | null;
  readonly isLast?: boolean;
}

const statusIcon: Record<
  DeployStatus,
  { icon: typeof Circle; className: string }
> = {
  success: { icon: CheckCircle2, className: "text-status-success-text" },
  failed: { icon: XCircle, className: "text-status-failed-text" },
  running: { icon: Loader2, className: "text-primary animate-spin" },
  pending: { icon: Clock, className: "text-status-running-text" },
  cancelled: { icon: Circle, className: "text-muted-foreground" },
};

export function ActivityFeedItem({
  appId,
  appName,
  status,
  commitSha,
  commitMessage,
  timestamp,
  isLast = false,
}: ActivityFeedItemProps) {
  const { icon: Icon, className: iconClass } = statusIcon[status];
  const trimmed = commitMessage?.trim() ?? "";
  const displayMessage = trimmed !== "" ? trimmed : "Manual deploy triggered";
  const displaySha = truncateCommitSha(commitSha);

  return (
    <Link
      to={`/apps/${appId}`}
      className="group relative flex items-start gap-3 rounded-lg px-2.5 py-2.5 transition-colors hover:bg-accent/40"
    >
      {!isLast && (
        <span
          className="absolute left-[21px] top-8 bottom-0 w-px bg-border/40 pointer-events-none"
          aria-hidden="true"
        />
      )}
      <div className="relative z-10 mt-0.5 shrink-0 bg-card/80 rounded-full">
        <Icon className={cn("h-4 w-4", iconClass)} aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1 space-y-1">
        <p className="truncate text-sm font-semibold text-foreground group-hover:text-primary transition-colors">
          {appName}
        </p>
        <p
          className="truncate text-xs text-muted-foreground"
          title={displayMessage}
        >
          {displayMessage}
        </p>
        <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground/70">
          {displaySha !== "" && (
            <span className="font-mono">
              {commitSha === "HEAD" ? "HEAD" : displaySha}
            </span>
          )}
          {displaySha !== "" && timestamp && <span>•</span>}
          {timestamp && <span>{formatRelativeTime(timestamp)}</span>}
        </div>
      </div>
    </Link>
  );
}
