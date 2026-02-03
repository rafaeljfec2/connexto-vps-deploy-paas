import { useParams } from "react-router-dom";
import {
  Activity,
  Cpu,
  HardDrive,
  Network,
  RefreshCw,
  Server as ServerIcon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorMessage } from "@/components/error-message";
import { PageHeader } from "@/components/page-header";
import { useServerStats } from "@/features/servers/hooks/use-server-stats";
import { useServer } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
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

export function ServerDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
  } = useServer(id);
  const {
    data: stats,
    isLoading: statsLoading,
    error: statsError,
    refetch,
    isFetching,
  } = useServerStats(id);

  if (serverLoading || !id) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-16 w-full" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      </div>
    );
  }

  if (serverError || !server) {
    return <ErrorMessage message="Server not found" />;
  }

  const status = server.status;
  const hasStats = stats != null;
  const statsUnavailable = statsError != null;

  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/servers"
        title={server.name}
        description={`${server.sshUser}@${server.host}:${server.sshPort}`}
        icon={ServerIcon}
      />

      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <Badge
                variant={
                  status === "online"
                    ? "default"
                    : status === "error" || status === "offline"
                      ? "destructive"
                      : status === "provisioning"
                        ? "default"
                        : "secondary"
                }
              >
                {status.charAt(0).toUpperCase() + status.slice(1)}
              </Badge>
              {server.agentVersion != null && (
                <span className="text-sm text-muted-foreground">
                  Agent {server.agentVersion}
                </span>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              disabled={isFetching || statsUnavailable}
            >
              <RefreshCw
                className={cn("h-4 w-4 mr-2", isFetching && "animate-spin")}
              />
              Refresh
            </Button>
          </div>
        </CardContent>
      </Card>

      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Activity className="h-5 w-5" />
          Resource Usage
        </h2>
        {statsLoading && !hasStats ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
          </div>
        ) : statsUnavailable ? (
          <Card>
            <CardContent className="py-8 text-center">
              <p className="text-muted-foreground mb-4">
                Agent unreachable. Ensure the server is provisioned and online.
              </p>
              <Button
                variant="outline"
                onClick={() => refetch()}
                disabled={isFetching}
              >
                Try again
              </Button>
            </CardContent>
          </Card>
        ) : hasStats ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <MetricCard
              icon={Cpu}
              title="CPU"
              value={`${(stats.systemMetrics.cpu_usage_percent ?? 0).toFixed(1)}%`}
              percent={stats.systemMetrics.cpu_usage_percent}
            />
            <MetricCard
              icon={HardDrive}
              title="Memory"
              value={formatBytes(stats.systemMetrics.memory_used_bytes ?? 0)}
              subValue={`/ ${formatBytes(
                (stats.systemInfo.memory_total_bytes ?? 0) ||
                  (stats.systemMetrics.memory_used_bytes ?? 0) +
                    (stats.systemMetrics.memory_available_bytes ?? 0),
              )}`}
              percent={
                (stats.systemInfo.memory_total_bytes ?? 0) > 0
                  ? ((stats.systemMetrics.memory_used_bytes ?? 0) /
                      (stats.systemInfo.memory_total_bytes ?? 1)) *
                    100
                  : undefined
              }
            />
            <MetricCard
              icon={HardDrive}
              title="Disk"
              value={formatBytes(stats.systemMetrics.disk_used_bytes ?? 0)}
              subValue={`/ ${formatBytes(
                (stats.systemMetrics.disk_used_bytes ?? 0) +
                  (stats.systemMetrics.disk_available_bytes ?? 0),
              )}`}
              percent={
                (stats.systemMetrics.disk_used_bytes ?? 0) +
                  (stats.systemMetrics.disk_available_bytes ?? 0) >
                0
                  ? ((stats.systemMetrics.disk_used_bytes ?? 0) /
                      ((stats.systemMetrics.disk_used_bytes ?? 0) +
                        (stats.systemMetrics.disk_available_bytes ?? 1))) *
                    100
                  : undefined
              }
            />
            <MetricCard
              icon={Network}
              title="Network"
              value={`↓ ${formatBytes(stats.systemMetrics.network_rx_bytes ?? 0)}`}
              subValue={`↑ ${formatBytes(stats.systemMetrics.network_tx_bytes ?? 0)}`}
            />
          </div>
        ) : null}
      </div>

      <div>
        <h2 className="text-lg font-semibold mb-4">Apps</h2>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Em breve
          </CardContent>
        </Card>
      </div>
    </div>
  );
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
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center gap-2 text-muted-foreground mb-2">
          <Icon className="h-4 w-4" />
          <span className="text-xs font-medium">{title}</span>
        </div>
        <div className="flex flex-col gap-1">
          <span
            className={cn(
              "text-lg font-semibold",
              percent !== undefined && getUsageTextColor(percent),
            )}
          >
            {value}
          </span>
          {subValue != null && (
            <span className="text-xs text-muted-foreground">{subValue}</span>
          )}
        </div>
        {percent !== undefined && (
          <div className="mt-3 h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all",
                getUsageColor(percent),
              )}
              style={{ width: `${Math.min(percent, 100)}%` }}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
