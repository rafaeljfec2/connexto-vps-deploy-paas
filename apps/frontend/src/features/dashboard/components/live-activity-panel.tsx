import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import {
  CheckCircle2,
  Circle,
  Clock,
  Loader2,
  Rocket,
  XCircle,
} from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { useRelativeTick } from "@/hooks/use-relative-tick";
import { formatRelativeTime } from "@/lib/format";
import { cn, truncateCommitSha } from "@/lib/utils";
import type { DeployStatus, DeploymentSummary } from "@/types";
import { LivePulse } from "./live-pulse";

interface DeployActivityLike {
  readonly appId: string;
  readonly appName: string;
  readonly deployment: DeploymentSummary;
}

interface LiveActivityPanelProps {
  readonly recentDeploys: readonly DeployActivityLike[];
  readonly isLoading: boolean;
  readonly isSSEConnected?: boolean;
}

const MAX_ITEMS = 3;

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

export function LiveActivityPanel({
  recentDeploys,
  isLoading,
  isSSEConnected,
}: LiveActivityPanelProps) {
  useRelativeTick();

  const items = recentDeploys.slice(0, MAX_ITEMS);

  return (
    <div className="flex h-full flex-col gap-3 rounded-xl border border-border/60 bg-background/50 p-4 backdrop-blur">
      <div className="flex items-center justify-between">
        <h2 className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
          Live activity
        </h2>
        <LivePulse isLoading={isLoading} isSSEConnected={isSSEConnected} />
      </div>

      {isLoading ? (
        <LiveActivitySkeleton />
      ) : items.length === 0 ? (
        <EmptyState />
      ) : (
        <ul className="flex flex-col gap-1">
          {items.map((entry) => {
            const timestamp =
              entry.deployment.startedAt ?? entry.deployment.finishedAt ?? null;
            const { icon: Icon, className } =
              statusIcon[entry.deployment.status];
            return (
              <li key={entry.deployment.id}>
                <Link
                  to={ROUTES.APP_DETAIL(entry.appId)}
                  className="group flex items-start gap-3 rounded-md px-2 py-2 transition-colors hover:bg-accent/40"
                >
                  <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", className)} />
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <p className="truncate text-sm font-medium group-hover:text-primary">
                      {entry.appName}
                    </p>
                    <div className="flex items-center gap-2 text-[10px] text-muted-foreground/80">
                      <span className="font-mono">
                        {truncateCommitSha(entry.deployment.commitSha)}
                      </span>
                      {timestamp && (
                        <span>{formatRelativeTime(timestamp)}</span>
                      )}
                    </div>
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function LiveActivitySkeleton() {
  return (
    <div className="space-y-2" aria-busy="true">
      {["sk-live-1", "sk-live-2", "sk-live-3"].map((key) => (
        <div key={key} className="flex items-start gap-3 px-2 py-1.5">
          <Skeleton className="mt-0.5 h-4 w-4 rounded-full" />
          <div className="flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-24" />
            <Skeleton className="h-2.5 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-border/50 px-4 py-6 text-center">
      <Rocket className="h-6 w-6 text-muted-foreground/60" aria-hidden="true" />
      <p className="text-xs text-muted-foreground">
        No deploys yet. Ship your first app to see activity here.
      </p>
      <Link
        to={ROUTES.NEW_APP}
        className="text-xs font-medium text-primary hover:underline"
      >
        Create an app
      </Link>
    </div>
  );
}
