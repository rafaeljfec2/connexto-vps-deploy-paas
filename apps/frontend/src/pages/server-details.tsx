import { useCallback, useState } from "react";
import { useParams } from "react-router-dom";
import {
  Activity,
  ArrowUpCircle,
  CheckCircle2,
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
import { api } from "@/services/api";
import type { Server, ServerStats } from "@/types";

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

function formatOsInfo(
  os: string | undefined,
  osVersion: string | undefined,
): string {
  if (os != null && osVersion != null) return `${os} ${osVersion}`;
  return os ?? osVersion ?? "—";
}

function getStatusBadgeVariant(
  status: string,
): "default" | "destructive" | "secondary" {
  if (status === "online" || status === "provisioning") return "default";
  if (status === "error" || status === "offline") return "destructive";
  return "secondary";
}

function SystemInfoBar({ stats }: { readonly stats: ServerStats }) {
  const { systemInfo } = stats;
  const items = [
    { label: "Host", value: systemInfo.hostname },
    { label: "OS", value: formatOsInfo(systemInfo.os, systemInfo.os_version) },
    { label: "Arch", value: systemInfo.architecture },
    { label: "Cores", value: String(systemInfo.cpu_cores ?? "—") },
    { label: "Kernel", value: systemInfo.kernel_version },
    {
      label: "RAM",
      value: formatBytes(systemInfo.memory_total_bytes ?? 0),
    },
    {
      label: "Disk",
      value: formatBytes(systemInfo.disk_total_bytes ?? 0),
    },
  ];

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {items.map((item) => (
        <span key={item.label}>
          <span className="font-medium text-foreground/70">{item.label}:</span>{" "}
          {item.value ?? "—"}
        </span>
      ))}
    </div>
  );
}

function getMemoryPercent(stats: ServerStats): number | undefined {
  const total = stats.systemInfo.memory_total_bytes ?? 0;
  if (total <= 0) return undefined;
  const used = stats.systemMetrics.memory_used_bytes ?? 0;
  return (used / total) * 100;
}

function getDiskPercent(stats: ServerStats): number | undefined {
  const used = stats.systemMetrics.disk_used_bytes ?? 0;
  const avail = stats.systemMetrics.disk_available_bytes ?? 0;
  const total = used + avail;
  if (total <= 0) return undefined;
  return (used / total) * 100;
}

function ResourceUsageSection({
  statsLoading,
  hasStats,
  statsUnavailable,
  stats,
  refetch,
  isFetching,
}: {
  readonly statsLoading: boolean;
  readonly hasStats: boolean;
  readonly statsUnavailable: boolean;
  readonly stats: ServerStats | null;
  readonly refetch: () => void;
  readonly isFetching: boolean;
}) {
  if (statsLoading && !hasStats) {
    return (
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
        {["cpu", "mem", "disk", "net", "load"].map((id) => (
          <Skeleton key={id} className="h-[72px]" />
        ))}
      </div>
    );
  }
  if (statsUnavailable) {
    return (
      <Card>
        <CardContent className="py-6 text-center">
          <p className="text-muted-foreground mb-3 text-sm">
            Agent unreachable. Ensure the server is provisioned and online.
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={refetch}
            disabled={isFetching}
          >
            Try again
          </Button>
        </CardContent>
      </Card>
    );
  }
  if (!hasStats || stats == null) return null;

  const m = stats.systemMetrics;
  const memTotal =
    stats.systemInfo.memory_total_bytes ??
    (m.memory_used_bytes ?? 0) + (m.memory_available_bytes ?? 0);
  const diskTotal = (m.disk_used_bytes ?? 0) + (m.disk_available_bytes ?? 0);
  const hasLoadAvg =
    m.load_average_1m != null ||
    m.load_average_5m != null ||
    m.load_average_15m != null;

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
      <MetricCard
        icon={Cpu}
        title="CPU"
        value={`${(m.cpu_usage_percent ?? 0).toFixed(1)}%`}
        percent={m.cpu_usage_percent}
      />
      <MetricCard
        icon={HardDrive}
        title="Memory"
        value={formatBytes(m.memory_used_bytes ?? 0)}
        subValue={`/ ${formatBytes(memTotal)}`}
        percent={getMemoryPercent(stats)}
      />
      <MetricCard
        icon={HardDrive}
        title="Disk"
        value={formatBytes(m.disk_used_bytes ?? 0)}
        subValue={`/ ${formatBytes(diskTotal)}`}
        percent={getDiskPercent(stats)}
      />
      <MetricCard
        icon={Network}
        title="Network"
        value={`↓ ${formatBytes(m.network_rx_bytes ?? 0)}`}
        subValue={`↑ ${formatBytes(m.network_tx_bytes ?? 0)}`}
      />
      {hasLoadAvg && (
        <MetricCard
          icon={Activity}
          title="Load avg"
          value={`1m: ${(m.load_average_1m ?? 0).toFixed(2)}`}
          subValue={`5m: ${(m.load_average_5m ?? 0).toFixed(2)} · 15m: ${(m.load_average_15m ?? 0).toFixed(2)}`}
        />
      )}
    </div>
  );
}

function AgentVersionCard({
  server,
  onUpdated,
}: {
  readonly server: Server;
  readonly onUpdated: () => void;
}) {
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateEnqueued, setUpdateEnqueued] = useState(false);
  const [updateError, setUpdateError] = useState<string | null>(null);

  const hasVersion = server.agentVersion != null;
  const isOutdated =
    hasVersion && server.agentVersion !== server.latestAgentVersion;
  const isUpToDate =
    hasVersion && server.agentVersion === server.latestAgentVersion;

  const handleUpdate = useCallback(async () => {
    setIsUpdating(true);
    setUpdateError(null);
    try {
      await api.servers.updateAgent(server.id);
      setUpdateEnqueued(true);
      setTimeout(() => {
        onUpdated();
      }, 35000);
    } catch {
      setUpdateError("Failed to enqueue agent update");
    } finally {
      setIsUpdating(false);
    }
  }, [server.id, onUpdated]);

  if (!hasVersion) return null;

  return (
    <Card>
      <CardContent className="py-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xs font-medium text-muted-foreground">
              Agent version:
            </span>
            <Badge variant={isOutdated ? "destructive" : "default"}>
              v{server.agentVersion}
            </Badge>
            {isOutdated && (
              <span className="text-xs text-muted-foreground">
                → v{server.latestAgentVersion} available
              </span>
            )}
            {isUpToDate && (
              <span className="flex items-center gap-1 text-xs text-emerald-500">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Up to date
              </span>
            )}
          </div>

          {isOutdated && !updateEnqueued && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleUpdate}
              disabled={isUpdating}
            >
              <ArrowUpCircle
                className={cn(
                  "h-3.5 w-3.5 mr-1.5",
                  isUpdating && "animate-spin",
                )}
              />
              {isUpdating ? "Sending..." : "Update Agent"}
            </Button>
          )}

          {updateEnqueued && (
            <span className="text-xs text-muted-foreground">
              Update enqueued. Agent will update on next heartbeat (~30s).
            </span>
          )}

          {updateError != null && (
            <span className="text-xs text-red-500">{updateError}</span>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export function ServerDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
    refetch: refetchServer,
  } = useServer(id);
  const {
    data: stats,
    isLoading: statsLoading,
    error: statsError,
    refetch: refetchStats,
    isFetching,
  } = useServerStats(id);

  const refetchAll = useCallback(() => {
    refetchServer();
    refetchStats();
  }, [refetchServer, refetchStats]);

  if (serverLoading || !id) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-14 w-full" />
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          {["cpu", "mem", "disk", "net", "load"].map((id) => (
            <Skeleton key={id} className="h-[72px]" />
          ))}
        </div>
      </div>
    );
  }

  if (serverError || !server) {
    return <ErrorMessage message="Server not found" />;
  }

  const hasStats = stats != null;
  const statsUnavailable = statsError != null;

  return (
    <div className="space-y-3">
      <PageHeader
        backTo="/servers"
        title={server.name}
        description={`${server.sshUser}@${server.host}:${server.sshPort}`}
        icon={ServerIcon}
        titleSuffix={
          <Badge variant={getStatusBadgeVariant(server.status)}>
            {server.status.charAt(0).toUpperCase() + server.status.slice(1)}
          </Badge>
        }
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={refetchAll}
            disabled={isFetching || statsUnavailable}
          >
            <RefreshCw
              className={cn("h-3.5 w-3.5 mr-1.5", isFetching && "animate-spin")}
            />
            Refresh
          </Button>
        }
      />

      {hasStats && stats != null && (
        <Card>
          <CardContent className="py-3">
            <SystemInfoBar stats={stats} />
          </CardContent>
        </Card>
      )}

      <AgentVersionCard server={server} onUpdated={refetchAll} />

      <ResourceUsageSection
        statsLoading={statsLoading}
        hasStats={hasStats}
        statsUnavailable={statsUnavailable}
        stats={stats ?? null}
        refetch={refetchStats}
        isFetching={isFetching}
      />

      <div>
        <h2 className="text-sm font-semibold mb-2">Apps</h2>
        <Card>
          <CardContent className="py-6 text-center text-muted-foreground text-sm">
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
      <CardContent className="p-3">
        <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
          <Icon className="h-3.5 w-3.5" />
          <span className="text-[11px] font-medium">{title}</span>
        </div>
        <div className="flex items-baseline gap-1.5">
          <span
            className={cn(
              "text-base font-semibold leading-tight",
              percent !== undefined && getUsageTextColor(percent),
            )}
          >
            {value}
          </span>
          {subValue != null && (
            <span className="text-[11px] text-muted-foreground">
              {subValue}
            </span>
          )}
        </div>
        {percent !== undefined && (
          <div className="mt-1.5 h-1 w-full bg-muted rounded-full overflow-hidden">
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
