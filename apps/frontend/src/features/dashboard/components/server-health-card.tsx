import { Link } from "react-router-dom";
import { Cpu, HardDrive, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useServerStats } from "@/features/servers/hooks/use-server-stats";
import { formatBytes } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Server } from "@/types";

interface ServerHealthCardProps {
  readonly server: Server;
  readonly appCount: number;
}

function UsageBar({ percent }: { readonly percent: number }) {
  const color =
    percent < 60
      ? "bg-emerald-500"
      : percent < 80
        ? "bg-yellow-500"
        : "bg-red-500";

  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div
        className={cn("h-full rounded-full transition-all", color)}
        style={{ width: `${Math.min(percent, 100)}%` }}
      />
    </div>
  );
}

const statusStyles: Record<string, string> = {
  online: "bg-emerald-500/10 text-emerald-500 border-emerald-500/20",
  offline: "bg-red-500/10 text-red-500 border-red-500/20",
  provisioning: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
  pending: "bg-muted text-muted-foreground border-border",
  error: "bg-red-500/10 text-red-500 border-red-500/20",
};

export function ServerHealthCard({ server, appCount }: ServerHealthCardProps) {
  const { data: stats, isLoading: statsLoading } = useServerStats(server.id);

  const cpuPercent = stats?.systemMetrics.cpu_usage_percent ?? 0;
  const memUsed = stats?.systemMetrics.memory_used_bytes ?? 0;
  const memTotal =
    stats?.systemInfo.memory_total_bytes ??
    memUsed + (stats?.systemMetrics.memory_available_bytes ?? 0);
  const memPercent = memTotal > 0 ? (memUsed / memTotal) * 100 : 0;

  return (
    <Link to={`/servers/${server.id}`}>
      <Card className="transition-colors hover:bg-accent/30">
        <CardContent className="space-y-3 p-4">
          <div className="flex items-center justify-between gap-2">
            <p className="truncate text-sm font-medium">{server.name}</p>
            <Badge
              variant="outline"
              className={cn(
                "shrink-0 text-[10px]",
                statusStyles[server.status] ?? statusStyles.pending,
              )}
            >
              {server.status}
            </Badge>
          </div>

          {server.status === "online" && (
            <>
              {statsLoading ? (
                <div className="flex items-center justify-center py-2">
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                </div>
              ) : stats ? (
                <div className="space-y-2">
                  <div className="space-y-1">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Cpu className="h-3 w-3" />
                        CPU
                      </span>
                      <span className="font-mono">
                        {cpuPercent.toFixed(1)}%
                      </span>
                    </div>
                    <UsageBar percent={cpuPercent} />
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <HardDrive className="h-3 w-3" />
                        Memory
                      </span>
                      <span className="font-mono">
                        {formatBytes(memUsed)} / {formatBytes(memTotal)}
                      </span>
                    </div>
                    <UsageBar percent={memPercent} />
                  </div>
                </div>
              ) : null}
            </>
          )}

          <p className="text-xs text-muted-foreground">
            {appCount} {appCount === 1 ? "app" : "apps"} deployed
          </p>
        </CardContent>
      </Card>
    </Link>
  );
}

export function ServerHealthCardSkeleton() {
  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-5 w-14" />
        </div>
        <div className="space-y-2">
          <Skeleton className="h-1.5 w-full" />
          <Skeleton className="h-1.5 w-full" />
        </div>
        <Skeleton className="h-3 w-20" />
      </CardContent>
    </Card>
  );
}
