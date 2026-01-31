import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  Activity,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  ChevronsDownUp,
  ChevronsUpDown,
  Cpu,
  ExternalLink,
  FileText,
  Folder,
  GitBranch,
  GitCommit,
  Globe,
  HardDrive,
  HeartPulse,
  History,
  Key,
  Link2,
  Link2Off,
  Network,
  Play,
  RefreshCw,
  Rocket,
  RotateCcw,
  Settings,
  Square,
  Terminal,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ContainerLogsViewer } from "@/components/container-logs-viewer";
import { ContainerMetrics } from "@/components/container-metrics";
import { HealthDetail, HealthIndicator } from "@/components/health-indicator";
import { IconText } from "@/components/icon-text";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { AppSettingsDialog } from "@/features/apps/components/app-settings-dialog";
import { CommitSelectorInline } from "@/features/apps/components/commit-selector";
import { DomainManager } from "@/features/apps/components/domain-manager";
import { EnvVarsManager } from "@/features/apps/components/env-vars-manager";
import {
  useApp,
  useAppConfig,
  useAppURL,
  useContainerStats,
  useRemoveWebhook,
  useRestartContainer,
  useSetupWebhook,
  useStartContainer,
  useStopContainer,
  useWebhookStatus,
} from "@/features/apps/hooks/use-apps";
import { useEnvVars } from "@/features/apps/hooks/use-env-vars";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { useAppHealth } from "@/hooks/use-sse";
import { cn, formatRepositoryUrl } from "@/lib/utils";

interface CollapsibleSectionProps {
  readonly title: string;
  readonly icon: React.ComponentType<{ className?: string }>;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly summary?: React.ReactNode;
  readonly actions?: React.ReactNode;
  readonly children: React.ReactNode;
}

function CollapsibleSection({
  title,
  icon: Icon,
  expanded,
  onToggle,
  summary,
  actions,
  children,
}: CollapsibleSectionProps) {
  return (
    <Card>
      <CardHeader
        className={cn(
          "flex flex-row items-center justify-between cursor-pointer select-none transition-colors hover:bg-muted/50",
          !expanded && "pb-4",
        )}
        onClick={onToggle}
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {expanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
            <Icon className="h-4 w-4 text-muted-foreground" />
          </div>
          <CardTitle className="text-base">{title}</CardTitle>
          {!expanded && summary && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground ml-2 truncate">
              <span className="text-muted-foreground/50">—</span>
              {summary}
            </div>
          )}
        </div>
        {actions && (
          <div
            className="flex items-center gap-2"
            onClick={(e) => e.stopPropagation()}
          >
            {actions}
          </div>
        )}
      </CardHeader>
      {expanded && <CardContent>{children}</CardContent>}
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function getHealthColor(health: string): string {
  if (health === "healthy") return "text-green-500";
  if (health === "unhealthy") return "text-red-500";
  return "text-yellow-500";
}

export function AppDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const { data: app, isLoading: appLoading } = useApp(id ?? "");
  const { data: deployments } = useDeploys(id ?? "");
  const { data: webhookStatus } = useWebhookStatus(id ?? "");
  const { data: health } = useAppHealth(id);
  const { data: appUrl } = useAppURL(id);
  const { data: appConfig } = useAppConfig(id);
  const { data: envVars } = useEnvVars(id ?? "");
  const { data: containerStats } = useContainerStats(id);
  const redeploy = useRedeploy();
  const rollback = useRollback();
  const setupWebhook = useSetupWebhook();
  const removeWebhook = useRemoveWebhook();
  const restartContainer = useRestartContainer();
  const stopContainer = useStopContainer();
  const startContainer = useStartContainer();

  const [selectedDeployId, setSelectedDeployId] = useState<string | null>(null);

  const [expandedSections, setExpandedSections] = useState<
    Record<string, boolean>
  >({
    deployments: false,
    containerLogs: false,
    metrics: false,
    envVars: false,
    health: false,
    config: false,
    webhook: false,
    domains: false,
  });

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => ({ ...prev, [section]: !prev[section] }));
  };

  const allExpanded = Object.values(expandedSections).every(Boolean);

  const toggleAllSections = () => {
    const newState = !allExpanded;
    setExpandedSections({
      deployments: newState,
      containerLogs: newState,
      metrics: newState,
      envVars: newState,
      health: newState,
      config: newState,
      webhook: newState,
      domains: newState,
    });
  };

  const latestDeploy = deployments?.[0];
  const selectedDeploy = selectedDeployId
    ? deployments?.find((d) => d.id === selectedDeployId)
    : latestDeploy;

  const handleRedeploy = (sha?: string) => {
    if (id) {
      redeploy.mutate({ appId: id, commitSha: sha });
    }
  };

  const handleRollback = () => {
    if (id) {
      rollback.mutate(id);
    }
  };

  const handleSetupWebhook = () => {
    if (id) {
      setupWebhook.mutate(id);
    }
  };

  const handleRemoveWebhook = () => {
    if (id) {
      removeWebhook.mutate(id);
    }
  };

  if (appLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4">
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
        </div>
      </div>
    );
  }

  if (!app) {
    return (
      <div className="text-center py-12">
        <p className="text-destructive">Application not found</p>
        <Button asChild variant="link">
          <Link to="/">Go back to dashboard</Link>
        </Button>
      </div>
    );
  }

  const successfulDeploys = deployments?.filter(
    (d) => d.status === "success",
  ).length;
  const totalDeploys = deployments?.length ?? 0;

  return (
    <div className="space-y-4">
      <PageHeader
        backTo="/"
        title={app.name}
        titleSuffix={
          <div className="flex items-center gap-2">
            <HealthIndicator health={health} />
            {latestDeploy && <StatusBadge status={latestDeploy.status} />}
          </div>
        }
        description={
          <div className="flex items-center gap-4 text-sm text-muted-foreground flex-wrap">
            <IconText icon={GitBranch} as="span">
              {app.branch}
            </IconText>
            <a
              href={app.repositoryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 hover:text-foreground"
            >
              <ExternalLink className="h-4 w-4" />
              {formatRepositoryUrl(app.repositoryUrl)}
            </a>
            {app.workdir && app.workdir !== "." && (
              <IconText icon={Folder} as="span">
                <span className="font-mono text-xs">{app.workdir}</span>
              </IconText>
            )}
          </div>
        }
        actions={
          <>
            <Button variant="outline" onClick={toggleAllSections}>
              {allExpanded ? (
                <>
                  <ChevronsDownUp className="h-4 w-4 mr-2" />
                  Collapse All
                </>
              ) : (
                <>
                  <ChevronsUpDown className="h-4 w-4 mr-2" />
                  Expand All
                </>
              )}
            </Button>
            {appUrl?.url && (
              <Button variant="outline" asChild>
                <a href={appUrl.url} target="_blank" rel="noopener noreferrer">
                  <Globe className="h-4 w-4 mr-2" />
                  Open App
                </a>
              </Button>
            )}
            <AppSettingsDialog app={app} />
            <Button
              variant="outline"
              onClick={handleRollback}
              disabled={
                rollback.isPending ||
                !deployments?.some((d) => d.status === "success")
              }
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Rollback
            </Button>
            <Button
              onClick={() => handleRedeploy()}
              disabled={redeploy.isPending}
            >
              <RefreshCw
                className={`h-4 w-4 mr-2 ${redeploy.isPending ? "animate-spin" : ""}`}
              />
              Redeploy
            </Button>
          </>
        }
      />

      <CollapsibleSection
        title="Deployments"
        icon={Rocket}
        expanded={expandedSections.deployments ?? false}
        onToggle={() => toggleSection("deployments")}
        summary={
          <span>
            {totalDeploys} deploys ({successfulDeploys} successful)
            {latestDeploy && (
              <>
                {" "}
                • Latest: <StatusBadge status={latestDeploy.status} size="sm" />
              </>
            )}
          </span>
        }
      >
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 overflow-hidden">
          <Tabs
            defaultValue="history"
            className="w-full min-w-0 overflow-hidden"
          >
            <TabsList className="grid w-full grid-cols-2 mb-4">
              <TabsTrigger value="history" className="gap-2">
                <History className="h-4 w-4" />
                Deploy History
              </TabsTrigger>
              <TabsTrigger value="commits" className="gap-2">
                <GitCommit className="h-4 w-4" />
                Commits
              </TabsTrigger>
            </TabsList>
            <TabsContent value="history" className="mt-0">
              <DeployTimeline
                appId={app.id}
                onSelectDeploy={setSelectedDeployId}
              />
            </TabsContent>
            <TabsContent value="commits" className="mt-0">
              <CommitSelectorInline
                appId={app.id}
                onSelect={handleRedeploy}
                disabled={redeploy.isPending}
              />
            </TabsContent>
          </Tabs>
          <div className="space-y-4 min-w-0 overflow-hidden">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <FileText className="h-4 w-4" />
              Deploy Logs
              {selectedDeploy && (
                <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                  {selectedDeploy.commitSha.slice(0, 7)}
                </span>
              )}
            </div>
            <LogViewer
              logs={selectedDeploy?.logs ?? null}
              title={
                selectedDeploy
                  ? `Logs (${selectedDeploy.commitSha.slice(0, 7)})`
                  : "Logs"
              }
            />
          </div>
        </div>
      </CollapsibleSection>

      <CollapsibleSection
        title="Container Logs"
        icon={Terminal}
        expanded={expandedSections.containerLogs ?? false}
        onToggle={() => toggleSection("containerLogs")}
        summary={
          health?.status === "running" ? (
            <span className="text-green-500">Container running</span>
          ) : (
            <span className="text-muted-foreground">
              Container {health?.status ?? "unknown"}
            </span>
          )
        }
      >
        <ContainerLogsViewer appId={app.id} appName={app.name} />
      </CollapsibleSection>

      <CollapsibleSection
        title="Resource Usage"
        icon={Activity}
        expanded={expandedSections.metrics ?? false}
        onToggle={() => toggleSection("metrics")}
        summary={
          containerStats && containerStats.cpuPercent > 0 ? (
            <span>
              CPU: {containerStats.cpuPercent.toFixed(1)}% • Memory:{" "}
              {formatBytes(containerStats.memoryUsage)} /{" "}
              {formatBytes(containerStats.memoryLimit)}
            </span>
          ) : (
            <span className="text-muted-foreground">No metrics available</span>
          )
        }
      >
        {id && <ContainerMetrics appId={id} embedded />}
      </CollapsibleSection>

      <CollapsibleSection
        title="Environment Variables"
        icon={Key}
        expanded={expandedSections.envVars ?? false}
        onToggle={() => toggleSection("envVars")}
        summary={
          <span>
            {envVars?.length ?? 0} variable
            {envVars?.length === 1 ? "" : "s"} configured
          </span>
        }
      >
        <EnvVarsManager appId={app.id} embedded />
      </CollapsibleSection>

      <CollapsibleSection
        title="Container Health"
        icon={HeartPulse}
        expanded={expandedSections.health ?? false}
        onToggle={() => toggleSection("health")}
        summary={
          <div className="flex items-center gap-2">
            {health?.status === "running" ? (
              <>
                <span className={getHealthColor(health.health)}>
                  {health.health}
                </span>
                {health.uptime && (
                  <span className="text-muted-foreground">
                    • Uptime: {health.uptime}
                  </span>
                )}
              </>
            ) : (
              <span className="text-muted-foreground">
                {health?.status ?? "unknown"}
              </span>
            )}
          </div>
        }
        actions={
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={() => id && restartContainer.mutate(id)}
              disabled={
                restartContainer.isPending || health?.status === "not_found"
              }
            >
              <RefreshCw
                className={`h-4 w-4 mr-1 ${restartContainer.isPending ? "animate-spin" : ""}`}
              />
              Restart
            </Button>
            {health?.status === "running" ? (
              <Button
                variant="outline"
                size="sm"
                onClick={() => id && stopContainer.mutate(id)}
                disabled={stopContainer.isPending}
              >
                <Square className="h-4 w-4 mr-1" />
                Stop
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                onClick={() => id && startContainer.mutate(id)}
                disabled={
                  startContainer.isPending || health?.status === "not_found"
                }
              >
                <Play className="h-4 w-4 mr-1" />
                Start
              </Button>
            )}
          </>
        }
      >
        <HealthDetail health={health} />
      </CollapsibleSection>

      {appConfig && (
        <CollapsibleSection
          title="Deployment Config"
          icon={Settings}
          expanded={expandedSections.config ?? false}
          onToggle={() => toggleSection("config")}
          summary={
            <span>
              Port{" "}
              {appConfig.hostPort === appConfig.port
                ? appConfig.port
                : `${appConfig.hostPort}:${appConfig.port}`}{" "}
              • {appConfig.resources.memory} RAM • {appConfig.resources.cpu} CPU
            </span>
          }
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm">
                <Network className="h-4 w-4 text-muted-foreground" />
                <span className="font-medium">Port:</span>
                <span className="font-mono">
                  {appConfig.hostPort === appConfig.port
                    ? appConfig.port
                    : `${appConfig.hostPort}:${appConfig.port}`}
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <HeartPulse className="h-4 w-4 text-muted-foreground" />
                <span className="font-medium">Health Check:</span>
                <span className="font-mono">{appConfig.healthcheck.path}</span>
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="ml-6">
                  Interval: {appConfig.healthcheck.interval} | Timeout:{" "}
                  {appConfig.healthcheck.timeout} | Retries:{" "}
                  {appConfig.healthcheck.retries}
                </span>
              </div>
            </div>
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm">
                <HardDrive className="h-4 w-4 text-muted-foreground" />
                <span className="font-medium">Memory:</span>
                <span className="font-mono">{appConfig.resources.memory}</span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <Cpu className="h-4 w-4 text-muted-foreground" />
                <span className="font-medium">CPU:</span>
                <span className="font-mono">{appConfig.resources.cpu}</span>
              </div>
              {appConfig.domains && appConfig.domains.length > 0 && (
                <div className="flex items-start gap-2 text-sm">
                  <Globe className="h-4 w-4 text-muted-foreground mt-0.5" />
                  <div>
                    <span className="font-medium">Domains:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {appConfig.domains.map((domain) => (
                        <span
                          key={domain}
                          className="px-2 py-0.5 bg-muted rounded text-xs font-mono"
                        >
                          {domain}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </CollapsibleSection>
      )}

      <CollapsibleSection
        title="GitHub Webhook"
        icon={Link2}
        expanded={expandedSections.webhook ?? false}
        onToggle={() => toggleSection("webhook")}
        summary={
          app.webhookId ? (
            <span className="text-green-500 flex items-center gap-1">
              <CheckCircle className="h-3 w-3" />
              Configured
              {webhookStatus?.active === false && (
                <span className="text-yellow-500 ml-1">(inactive)</span>
              )}
            </span>
          ) : (
            <span className="text-muted-foreground flex items-center gap-1">
              <XCircle className="h-3 w-3" />
              Not configured
            </span>
          )
        }
        actions={
          app.webhookId ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleRemoveWebhook}
              disabled={removeWebhook.isPending}
            >
              <Link2Off className="h-4 w-4 mr-1" />
              Remove
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={handleSetupWebhook}
              disabled={setupWebhook.isPending}
            >
              <Link2 className="h-4 w-4 mr-1" />
              Setup
            </Button>
          )
        }
      >
        <div className="flex items-center gap-3">
          {app.webhookId ? (
            <>
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <p className="font-medium">Webhook configured</p>
                <p className="text-sm text-muted-foreground">
                  Auto-deploy enabled for push events
                  {webhookStatus?.active === false && (
                    <span className="text-yellow-500 ml-2">(inactive)</span>
                  )}
                </p>
              </div>
            </>
          ) : (
            <>
              <XCircle className="h-5 w-5 text-muted-foreground" />
              <div>
                <p className="font-medium">Webhook not configured</p>
                <p className="text-sm text-muted-foreground">
                  Configure to enable auto-deploy on push
                </p>
              </div>
            </>
          )}
        </div>
      </CollapsibleSection>

      <CollapsibleSection
        title="Custom Domains"
        icon={Globe}
        expanded={expandedSections.domains ?? false}
        onToggle={() => toggleSection("domains")}
        summary={<span className="text-muted-foreground">Cloudflare DNS</span>}
      >
        <DomainManager appId={app.id} />
      </CollapsibleSection>
    </div>
  );
}
