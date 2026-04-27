import {
  CheckCircle2,
  Circle,
  Clock,
  History,
  Loader2,
  RefreshCw,
  XCircle,
} from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { useRelativeTick } from "@/hooks/use-relative-tick";
import { formatRelativeTime } from "@/lib/format";
import { cn, truncateCommitSha } from "@/lib/utils";
import type { DeployStatus, Deployment } from "@/types";

interface AppRecentDeploysPanelProps {
  readonly deployments: readonly Deployment[] | undefined;
  readonly isLoading?: boolean;
  readonly onRedeploy?: () => void;
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

export function AppRecentDeploysPanel({
  deployments,
  isLoading = false,
  onRedeploy,
}: AppRecentDeploysPanelProps) {
  useRelativeTick();

  const items = (deployments ?? []).slice(0, MAX_ITEMS);

  return (
    <div className="flex h-full flex-col gap-3 rounded-xl border border-border/60 bg-background/50 p-4 backdrop-blur">
      <div className="flex items-center justify-between">
        <h2 className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
          Recent deploys
        </h2>
        <span className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <History className="h-3 w-3" aria-hidden="true" />
          Last {MAX_ITEMS}
        </span>
      </div>

      {isLoading ? (
        <RecentDeploysSkeleton />
      ) : items.length === 0 ? (
        <EmptyState onRedeploy={onRedeploy} />
      ) : (
        <ul className="flex flex-col gap-1">
          {items.map((deploy) => {
            const timestamp = deploy.finishedAt ?? deploy.startedAt ?? null;
            const { icon: Icon, className } = statusIcon[deploy.status];
            return (
              <li
                key={deploy.id}
                className="group flex items-start gap-3 rounded-md px-2 py-2 transition-colors hover:bg-accent/40"
              >
                <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", className)} />
                <div className="min-w-0 flex-1 space-y-0.5">
                  <p className="truncate text-sm font-medium">
                    {deploy.commitMessage?.trim() ||
                      truncateCommitSha(deploy.commitSha)}
                  </p>
                  <div className="flex items-center gap-2 text-[10px] text-muted-foreground/80">
                    <span className="font-mono">
                      {truncateCommitSha(deploy.commitSha)}
                    </span>
                    {timestamp && <span>{formatRelativeTime(timestamp)}</span>}
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function RecentDeploysSkeleton() {
  return (
    <div className="space-y-2" aria-busy="true">
      {["sk-rd-1", "sk-rd-2", "sk-rd-3"].map((key) => (
        <div key={key} className="flex items-start gap-3 px-2 py-1.5">
          <Skeleton className="mt-0.5 h-4 w-4 rounded-full" />
          <div className="flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-2.5 w-20" />
          </div>
        </div>
      ))}
    </div>
  );
}

interface EmptyStateProps {
  readonly onRedeploy?: () => void;
}

function EmptyState({ onRedeploy }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-border/50 px-4 py-6 text-center">
      <RefreshCw
        className="h-6 w-6 text-muted-foreground/60"
        aria-hidden="true"
      />
      <p className="text-xs text-muted-foreground">
        No deploys recorded yet for this app.
      </p>
      {onRedeploy && (
        <button
          type="button"
          onClick={onRedeploy}
          className="text-xs font-medium text-primary hover:underline"
        >
          Trigger first deploy
        </button>
      )}
    </div>
  );
}
