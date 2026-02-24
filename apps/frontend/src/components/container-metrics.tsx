import { Cpu, HardDrive, Network, RefreshCw, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useContainerStats } from "@/features/apps/hooks/use-apps";
import { formatBytes } from "@/lib/format";
import { cn } from "@/lib/utils";

interface ContainerMetricsProps {
  readonly appId: string;
  readonly embedded?: boolean;
}

function getUsageColor(percent: number): string {
  if (percent < 60) return "bg-emerald-500";
  if (percent < 80) return "bg-yellow-500";
  return "bg-red-500";
}

function getUsageTextColor(percent: number): string {
  if (percent < 60) return "text-emerald-400";
  if (percent < 80) return "text-yellow-400";
  return "text-red-400";
}

interface MetricCardProps {
  readonly icon: React.ComponentType<{ className?: string }>;
  readonly title: string;
  readonly value: string;
  readonly subValue?: string;
  readonly percent?: number;
}

function MetricCard({
  icon: Icon,
  title,
  value,
  subValue,
  percent,
}: MetricCardProps) {
  return (
    <div className="flex flex-col gap-2 p-3 rounded-lg bg-slate-900/50 border border-slate-800">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="h-4 w-4" />
        <span className="text-xs font-medium">{title}</span>
      </div>
      <div className="flex items-baseline gap-2">
        <span
          className={cn(
            "text-lg font-semibold",
            percent !== undefined && getUsageTextColor(percent),
          )}
        >
          {value}
        </span>
        {subValue && (
          <span className="text-xs text-muted-foreground">{subValue}</span>
        )}
      </div>
      {percent !== undefined && (
        <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden">
          <div
            className={cn(
              "h-full rounded-full transition-all",
              getUsageColor(percent),
            )}
            style={{ width: `${Math.min(percent, 100)}%` }}
          />
        </div>
      )}
    </div>
  );
}

export function ContainerMetrics({
  appId,
  embedded = false,
}: ContainerMetricsProps) {
  const {
    data: stats,
    isLoading,
    refetch,
    isFetching,
  } = useContainerStats(appId);

  const hasData = stats && (stats.cpuPercent > 0 || stats.memoryUsage > 0);

  const renderMetricsContent = () => {
    if (isLoading) {
      return (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      );
    }

    if (!hasData) {
      return (
        <div className="text-center py-6 text-muted-foreground text-sm">
          No metrics available. Container may not be running.
        </div>
      );
    }

    return (
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          icon={Cpu}
          title="CPU"
          value={`${stats.cpuPercent.toFixed(1)}%`}
          percent={stats.cpuPercent}
        />
        <MetricCard
          icon={HardDrive}
          title="Memory"
          value={formatBytes(stats.memoryUsage)}
          subValue={`/ ${formatBytes(stats.memoryLimit)}`}
          percent={stats.memoryPercent}
        />
        <MetricCard
          icon={Network}
          title="Network I/O"
          value={`↓ ${formatBytes(stats.networkRx)}`}
          subValue={`↑ ${formatBytes(stats.networkTx)}`}
        />
        <MetricCard
          icon={Users}
          title="Processes"
          value={stats.pids.toString()}
          subValue="PIDs"
        />
      </div>
    );
  };

  if (embedded) {
    return (
      <div className="space-y-3">
        <div className="flex justify-end">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="h-7"
          >
            <RefreshCw
              className={cn("h-3.5 w-3.5 mr-1", isFetching && "animate-spin")}
            />
            Refresh
          </Button>
        </div>
        {renderMetricsContent()}
      </div>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-3">
        <CardTitle className="text-base">Resource Usage</CardTitle>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          className="h-7"
        >
          <RefreshCw
            className={cn("h-3.5 w-3.5 mr-1", isFetching && "animate-spin")}
          />
          Refresh
        </Button>
      </CardHeader>
      <CardContent>{renderMetricsContent()}</CardContent>
    </Card>
  );
}
