import { Activity, Cpu, HardDrive, Network } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes } from "@/lib/format";
import type { ServerStats } from "@/types";
import { MetricCard } from "./metric-card";

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

interface ResourceUsageSectionProps {
  readonly statsLoading: boolean;
  readonly hasStats: boolean;
  readonly statsUnavailable: boolean;
  readonly stats: ServerStats | null;
  readonly refetch: () => void;
  readonly isFetching: boolean;
}

export function ResourceUsageSection({
  statsLoading,
  hasStats,
  statsUnavailable,
  stats,
  refetch,
  isFetching,
}: ResourceUsageSectionProps) {
  if (statsLoading && !hasStats) {
    return (
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-5">
        {["cpu", "mem", "disk", "net", "load"].map((id) => (
          <Skeleton key={id} className="h-[80px]" />
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
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-5">
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
