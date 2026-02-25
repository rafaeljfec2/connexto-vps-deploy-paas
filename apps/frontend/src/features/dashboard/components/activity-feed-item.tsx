import { Link } from "react-router-dom";
import { CheckCircle2, Circle, Clock, Loader2, XCircle } from "lucide-react";
import { cn, formatRelativeTime, truncateCommitSha } from "@/lib/utils";
import type { DeployStatus } from "@/types";

interface ActivityFeedItemProps {
  readonly appId: string;
  readonly appName: string;
  readonly status: DeployStatus;
  readonly commitSha: string;
  readonly commitMessage?: string;
  readonly timestamp: string | null;
}

const statusIcon: Record<
  DeployStatus,
  { icon: typeof Circle; className: string }
> = {
  success: { icon: CheckCircle2, className: "text-emerald-500" },
  failed: { icon: XCircle, className: "text-red-500" },
  running: { icon: Loader2, className: "text-blue-500 animate-spin" },
  pending: { icon: Clock, className: "text-yellow-500" },
  cancelled: { icon: Circle, className: "text-muted-foreground" },
};

export function ActivityFeedItem({
  appId,
  appName,
  status,
  commitSha,
  commitMessage,
  timestamp,
}: ActivityFeedItemProps) {
  const { icon: Icon, className: iconClass } = statusIcon[status];

  return (
    <Link
      to={`/apps/${appId}`}
      className="group flex items-start gap-3 rounded-md px-2 py-2 transition-colors hover:bg-accent/50"
    >
      <div className="mt-0.5 shrink-0">
        <Icon className={cn("h-4 w-4", iconClass)} />
      </div>
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="truncate text-sm font-medium group-hover:text-primary">
          {appName}
        </p>
        {commitMessage && (
          <p className="truncate text-xs text-muted-foreground">
            {commitMessage}
          </p>
        )}
        <div className="flex items-center gap-2 text-[10px] text-muted-foreground/70">
          <span className="font-mono">{truncateCommitSha(commitSha)}</span>
          {timestamp && <span>{formatRelativeTime(timestamp)}</span>}
        </div>
      </div>
    </Link>
  );
}
