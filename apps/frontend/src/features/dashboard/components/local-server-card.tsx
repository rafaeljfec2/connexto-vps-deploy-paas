import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { Cpu, HardDrive, Home, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { formatBytes } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { ServerStats } from "@/types";
import { useLocalServerStats } from "../hooks/use-local-server-stats";

interface LocalServerCardProps {
  readonly appCount: number;
  readonly containerCount: number;
}

function getUsageColor(percent: number): string {
  if (percent < 60) return "bg-emerald-500";
  if (percent < 80) return "bg-yellow-500";
  return "bg-red-500";
}

function UsageBar({ percent }: { readonly percent: number }) {
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div
        className={cn(
          "h-full rounded-full transition-all",
          getUsageColor(percent),
        )}
        style={{ width: `${Math.min(percent, 100)}%` }}
      />
    </div>
  );
}

function StatsContent({ stats }: { readonly stats: ServerStats }) {
  const cpuPercent = stats.systemMetrics.cpu_usage_percent ?? 0;
  const memUsed = stats.systemMetrics.memory_used_bytes ?? 0;
  const memTotal =
    stats.systemInfo.memory_total_bytes ??
    memUsed + (stats.systemMetrics.memory_available_bytes ?? 0);
  const memPercent = memTotal > 0 ? (memUsed / memTotal) * 100 : 0;

  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <Cpu className="h-3 w-3" />
            CPU
          </span>
          <span className="font-mono">{cpuPercent.toFixed(1)}%</span>
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
  );
}

function StatsSection({
  stats,
  isLoading,
}: {
  readonly stats: ServerStats | undefined;
  readonly isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-2">
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!stats) return null;

  return <StatsContent stats={stats} />;
}

export function LocalServerCard({
  appCount,
  containerCount,
}: LocalServerCardProps) {
  const { data: stats, isLoading } = useLocalServerStats();

  return (
    <Link to={ROUTES.CONTAINERS}>
      <Card className="transition-colors hover:bg-accent/30">
        <CardContent className="space-y-3 p-4">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <Home className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <p className="truncate text-sm font-medium">Local</p>
            </div>
            <Badge
              variant="outline"
              className="shrink-0 border-emerald-500/20 bg-emerald-500/10 text-[10px] text-emerald-500"
            >
              self-hosted
            </Badge>
          </div>

          <StatsSection stats={stats} isLoading={isLoading} />

          <p className="text-xs text-muted-foreground">
            {appCount} {appCount === 1 ? "app" : "apps"}, {containerCount}{" "}
            {containerCount === 1 ? "container" : "containers"}
          </p>
        </CardContent>
      </Card>
    </Link>
  );
}
