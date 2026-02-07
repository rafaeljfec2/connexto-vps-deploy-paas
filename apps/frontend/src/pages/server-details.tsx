import { useParams } from "react-router-dom";
import {
  Activity,
  Cpu,
  HardDrive,
  Info,
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
import type { ServerStats } from "@/types";

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

function InfoRow({
  label,
  value,
}: {
  readonly label: string;
  readonly value: string | undefined | null;
}) {
  return (
    <div>
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium">{value ?? "—"}</dd>
    </div>
  );
}

function getStatusBadgeVariant(
  status: string,
): "default" | "destructive" | "secondary" {
  if (status === "online" || status === "provisioning") return "default";
  if (status === "error" || status === "offline") return "destructive";
  return "secondary";
}

function ServerStatusCard({
  status,
  agentVersion,
  onRefresh,
  isFetching,
  statsUnavailable,
}: {
  readonly status: string;
  readonly agentVersion: string | undefined;
  readonly onRefresh: () => void;
  readonly isFetching: boolean;
  readonly statsUnavailable: boolean;
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={getStatusBadgeVariant(status)}>
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </Badge>
            {agentVersion != null && (
              <span className="text-sm text-muted-foreground">
                Agent {agentVersion}
              </span>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
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
  );
}

function SystemInfoSection({ stats }: { readonly stats: ServerStats }) {
  const { systemInfo } = stats;
  return (
    <div>
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <Info className="h-5 w-5" />
        System Information
      </h2>
      <Card>
        <CardContent className="pt-6">
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <InfoRow label="Hostname" value={systemInfo.hostname} />
            <InfoRow
              label="OS"
              value={formatOsInfo(systemInfo.os, systemInfo.os_version)}
            />
            <InfoRow label="Architecture" value={systemInfo.architecture} />
            <InfoRow
              label="CPU cores"
              value={String(systemInfo.cpu_cores ?? "—")}
            />
            <InfoRow label="Kernel" value={systemInfo.kernel_version} />
            <InfoRow
              label="Memory total"
              value={formatBytes(systemInfo.memory_total_bytes ?? 0)}
            />
            <InfoRow
              label="Disk total"
              value={formatBytes(systemInfo.disk_total_bytes ?? 0)}
            />
          </dl>
        </CardContent>
      </Card>
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
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
    );
  }
  if (statsUnavailable) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <p className="text-muted-foreground mb-4">
            Agent unreachable. Ensure the server is provisioned and online.
          </p>
          <Button variant="outline" onClick={refetch} disabled={isFetching}>
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
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
          title="Load average"
          value={`1m: ${(m.load_average_1m ?? 0).toFixed(2)}`}
          subValue={`5m: ${(m.load_average_5m ?? 0).toFixed(2)} · 15m: ${(m.load_average_15m ?? 0).toFixed(2)}`}
        />
      )}
    </div>
  );
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
      <ServerStatusCard
        status={server.status}
        agentVersion={server.agentVersion}
        onRefresh={refetch}
        isFetching={isFetching}
        statsUnavailable={statsUnavailable}
      />
      {hasStats && stats != null && <SystemInfoSection stats={stats} />}
      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Activity className="h-5 w-5" />
          Resource Usage
        </h2>
        <ResourceUsageSection
          statsLoading={statsLoading}
          hasStats={hasStats}
          statsUnavailable={statsUnavailable}
          stats={stats ?? null}
          refetch={refetch}
          isFetching={isFetching}
        />
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
