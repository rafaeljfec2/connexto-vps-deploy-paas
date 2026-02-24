import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Activity,
  ArrowUpCircle,
  Box,
  CheckCircle2,
  Cpu,
  GitBranch,
  HardDrive,
  Loader2,
  Monitor,
  Network,
  Plus,
  RefreshCw,
  Server as ServerIcon,
  Settings,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ErrorMessage } from "@/components/error-message";
import { PageHeader } from "@/components/page-header";
import { ContainerList } from "@/features/containers";
import { ImageList } from "@/features/images";
import { clearAgentUpdateState } from "@/features/servers/agent-update-store";
import { useAgentUpdate } from "@/features/servers/hooks/use-agent-update";
import { useServerStats } from "@/features/servers/hooks/use-server-stats";
import { useServer, useServerApps } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import type { AgentUpdateMode, App, Server, ServerStats } from "@/types";

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

function compareSemver(a: string, b: string): number {
  const pa = a.split(".").map(Number);
  const pb = b.split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    const diff = (pa[i] ?? 0) - (pb[i] ?? 0);
    if (diff !== 0) return diff;
  }
  return 0;
}

function getAgentVersionStatus(server: Server) {
  const currentVersion = server.agentVersion ?? null;
  const latestVersion = server.latestAgentVersion;
  const isUnknown = currentVersion == null;
  const isUpToDate =
    !isUnknown && compareSemver(currentVersion, latestVersion) >= 0;
  const isOutdated = !isUnknown && !isUpToDate;
  const needsUpdate = isUnknown || isOutdated;

  return {
    currentVersion,
    latestVersion,
    isUnknown,
    isUpToDate,
    isOutdated,
    needsUpdate,
  };
}

const AGENT_UPDATE_STEP_LABELS: Record<string, string> = {
  enqueued: "Waiting for agent heartbeat...",
  delivered: "Agent downloading update...",
  updated: "Update completed!",
};

function AgentVersionCard({
  server,
  onUpdated,
}: {
  readonly server: Server;
  readonly onUpdated: () => void;
}) {
  const [isSending, setIsSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const agentUpdate = useAgentUpdate(server.id);

  const {
    currentVersion,
    latestVersion,
    isUnknown,
    isUpToDate,
    isOutdated,
    needsUpdate,
  } = getAgentVersionStatus(server);

  const isUpdateInProgress = agentUpdate?.status === "running";
  const isUpdateCompleted = agentUpdate?.status === "completed";
  const isUpdateError = agentUpdate?.status === "error";

  useEffect(() => {
    if (!isUpdateCompleted) return;

    onUpdated();

    const timeout = globalThis.setTimeout(() => {
      clearAgentUpdateState(server.id);
    }, 5000);

    return () => globalThis.clearTimeout(timeout);
  }, [isUpdateCompleted, onUpdated, server.id]);

  useEffect(() => {
    if (!isUpdateError) return;

    const timeout = globalThis.setTimeout(() => {
      clearAgentUpdateState(server.id);
    }, 10000);

    return () => globalThis.clearTimeout(timeout);
  }, [isUpdateError, server.id]);

  const handleUpdate = useCallback(async () => {
    setIsSending(true);
    setSendError(null);
    try {
      await api.servers.updateAgent(server.id);
    } catch {
      setSendError("Failed to enqueue agent update");
    } finally {
      setIsSending(false);
    }
  }, [server.id]);

  if (server.status !== "online") return null;

  return (
    <Card>
      <CardContent className="py-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <span className="text-xs font-medium text-muted-foreground">
              Agent:
            </span>
            {isUnknown && <Badge variant="secondary">Unknown version</Badge>}
            {!isUnknown && (
              <Badge variant={isOutdated ? "destructive" : "default"}>
                v{currentVersion}
              </Badge>
            )}
            {(isOutdated || isUnknown) &&
              latestVersion &&
              !isUpdateInProgress &&
              !isUpdateCompleted && (
                <span className="text-xs text-muted-foreground">
                  v{latestVersion} available
                </span>
              )}
            {isUpToDate && !isUpdateCompleted && (
              <span className="flex items-center gap-1 text-xs text-emerald-500">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Up to date
              </span>
            )}
          </div>

          {isUpdateInProgress && agentUpdate != null && (
            <span className="flex items-center gap-1.5 text-xs text-blue-500">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {AGENT_UPDATE_STEP_LABELS[agentUpdate.step] ?? agentUpdate.step}
            </span>
          )}

          {isUpdateCompleted && agentUpdate != null && (
            <span className="flex items-center gap-1.5 text-xs text-emerald-500">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Updated to v{agentUpdate.version}
            </span>
          )}

          {isUpdateError && agentUpdate != null && (
            <span className="text-xs text-red-500">
              Update failed: {agentUpdate.errorMessage ?? "unknown error"}
            </span>
          )}

          {!isUpdateInProgress && !isUpdateCompleted && needsUpdate && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleUpdate}
              disabled={isSending}
            >
              <ArrowUpCircle
                className={cn(
                  "h-3.5 w-3.5 mr-1.5",
                  isSending && "animate-spin",
                )}
              />
              {isSending ? "Sending..." : "Update Agent"}
            </Button>
          )}

          {sendError != null && (
            <span className="text-xs text-red-500">{sendError}</span>
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

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">
            <Monitor className="h-3.5 w-3.5 mr-1.5" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="containers">
            <Box className="h-3.5 w-3.5 mr-1.5" />
            Containers
          </TabsTrigger>
          <TabsTrigger value="images">
            <HardDrive className="h-3.5 w-3.5 mr-1.5" />
            Images
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="h-3.5 w-3.5 mr-1.5" />
            Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-3">
          <ResourceUsageSection
            statsLoading={statsLoading}
            hasStats={hasStats}
            statsUnavailable={statsUnavailable}
            stats={stats ?? null}
            refetch={refetchStats}
            isFetching={isFetching}
          />
          <ServerAppsSection serverId={server.id} />
        </TabsContent>

        <TabsContent value="containers">
          <ContainerList serverId={server.id} />
        </TabsContent>

        <TabsContent value="images">
          <ImageList serverId={server.id} />
        </TabsContent>

        <TabsContent value="settings">
          <ServerSettingsSection server={server} onSaved={refetchAll} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

interface ServerAppsSectionProps {
  readonly serverId: string;
}

function ServerAppsSection({ serverId }: ServerAppsSectionProps) {
  const { data: apps, isLoading } = useServerApps(serverId);

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-sm font-semibold">Apps</h2>
        <Button variant="outline" size="sm" className="h-7 text-xs" asChild>
          <Link to={`/apps/new?serverId=${serverId}`}>
            <Plus className="h-3.5 w-3.5 mr-1" />
            New App
          </Link>
        </Button>
      </div>

      {isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      )}

      {!isLoading && !apps?.length && (
        <Card>
          <CardContent className="py-8 text-center">
            <ServerIcon className="h-8 w-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              No apps deployed on this server
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3 text-xs"
              asChild
            >
              <Link to={`/apps/new?serverId=${serverId}`}>
                <Plus className="h-3.5 w-3.5 mr-1" />
                Deploy your first app
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}

      {!isLoading && apps && apps.length > 0 && (
        <div className="space-y-2">
          {apps.map((app) => (
            <ServerAppRow key={app.id} app={app} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ServerSettingsSectionProps {
  readonly server: Server;
  readonly onSaved: () => void;
}

function ServerSettingsSection({
  server,
  onSaved,
}: ServerSettingsSectionProps) {
  const [updateMode, setUpdateMode] = useState<AgentUpdateMode>(
    server.agentUpdateMode ?? "grpc",
  );
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const isDirty = updateMode !== (server.agentUpdateMode ?? "grpc");

  const handleSave = useCallback(async () => {
    setIsSaving(true);
    setSaveError(null);
    setSaved(false);
    try {
      await api.servers.update(server.id, { agentUpdateMode: updateMode });
      setSaved(true);
      onSaved();
      globalThis.setTimeout(() => setSaved(false), 3000);
    } catch {
      setSaveError("Failed to save settings");
    } finally {
      setIsSaving(false);
    }
  }, [server.id, updateMode, onSaved]);

  return (
    <Card>
      <CardContent className="py-4 space-y-4">
        <h3 className="text-sm font-semibold">Agent Settings</h3>

        <div className="space-y-2 max-w-sm">
          <Label htmlFor="agent-update-mode">Agent Update Mode</Label>
          <Select
            value={updateMode}
            onValueChange={(v) => setUpdateMode(v as AgentUpdateMode)}
          >
            <SelectTrigger id="agent-update-mode">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="grpc">gRPC (Push direto)</SelectItem>
              <SelectItem value="https">HTTPS (Pull via heartbeat)</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            gRPC pushes the binary directly through the existing connection.
            HTTPS makes the agent download the binary via HTTP.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button
            size="sm"
            onClick={handleSave}
            disabled={!isDirty || isSaving}
          >
            {isSaving ? "Saving..." : "Save"}
          </Button>
          {saved && (
            <span className="flex items-center gap-1 text-xs text-emerald-500">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Saved
            </span>
          )}
          {saveError != null && (
            <span className="text-xs text-red-500">{saveError}</span>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function getDeployBadgeVariant(
  status: string,
): "default" | "destructive" | "secondary" {
  if (status === "success") return "default";
  if (status === "failed") return "destructive";
  return "secondary";
}

interface ServerAppRowProps {
  readonly app: App;
}

function ServerAppRow({ app }: ServerAppRowProps) {
  const deployment = app.lastDeployment;
  const isDeploying =
    deployment?.status === "running" || deployment?.status === "pending";

  return (
    <Link
      to={`/apps/${app.id}`}
      className={cn(
        "flex items-center gap-4 rounded-lg border bg-card p-3 transition-colors",
        isDeploying ? "border-primary/50" : "hover:bg-accent/50",
      )}
    >
      {isDeploying && (
        <div className="absolute inset-0 z-5 bg-background/60 backdrop-blur-[1px] rounded-lg flex items-center justify-center pointer-events-none">
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
        </div>
      )}

      <div className="flex-1 min-w-0">
        <span className="font-medium text-sm truncate">{app.name}</span>
        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
          <span className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            {app.branch}
          </span>
          {deployment?.commitSha && (
            <span className="font-mono text-[11px]">
              {deployment.commitSha.slice(0, 7)}
            </span>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        {deployment && (
          <Badge
            variant={getDeployBadgeVariant(deployment.status)}
            className="text-[10px] px-1.5 h-5"
          >
            {deployment.status}
          </Badge>
        )}
      </div>
    </Link>
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
