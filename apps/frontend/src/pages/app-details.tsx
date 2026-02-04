import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
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
import { useContainers } from "@/features/containers/hooks/use-containers";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { NetworksManager, VolumesManager } from "@/features/resources";
import { useAppHealth } from "@/hooks/use-sse";
import { cn, formatRepositoryUrl } from "@/lib/utils";
import { api } from "@/services/api";
import type { App, CustomDomain, Deployment } from "@/types";

type SectionKey =
  | "deployments"
  | "containerLogs"
  | "metrics"
  | "envVars"
  | "health"
  | "config"
  | "webhook"
  | "domains"
  | "networks"
  | "volumes";

function useExpandedSections() {
  const [expandedSections, setExpandedSections] = useState<
    Record<SectionKey, boolean>
  >({
    deployments: false,
    containerLogs: false,
    metrics: false,
    envVars: false,
    health: false,
    config: false,
    webhook: false,
    domains: false,
    networks: false,
    volumes: false,
  });

  const toggleSection = (section: SectionKey) => {
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
      networks: newState,
      volumes: newState,
    });
  };

  return { expandedSections, toggleSection, allExpanded, toggleAllSections };
}

function useAppActions(id: string | undefined) {
  const redeploy = useRedeploy();
  const rollback = useRollback();
  const setupWebhook = useSetupWebhook();
  const removeWebhook = useRemoveWebhook();
  const restartContainer = useRestartContainer();
  const stopContainer = useStopContainer();
  const startContainer = useStartContainer();

  const handleRedeploy = (sha?: string) => {
    if (id) redeploy.mutate({ appId: id, commitSha: sha });
  };

  const handleRollback = () => {
    if (id) rollback.mutate(id);
  };

  const handleSetupWebhook = () => {
    if (id) setupWebhook.mutate(id);
  };

  const handleRemoveWebhook = () => {
    if (id) removeWebhook.mutate(id);
  };

  return {
    redeploy,
    rollback,
    setupWebhook,
    removeWebhook,
    restartContainer,
    stopContainer,
    startContainer,
    handleRedeploy,
    handleRollback,
    handleSetupWebhook,
    handleRemoveWebhook,
  };
}

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
    <div>
      <CardHeader
        className={cn(
          "flex flex-col sm:flex-row sm:items-center justify-between cursor-pointer select-none transition-colors hover:bg-muted/50 gap-2 sm:gap-3 p-4",
          !expanded && "pb-4",
        )}
        onClick={onToggle}
      >
        <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0">
          <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
            {expanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
            <Icon className="h-4 w-4 text-muted-foreground" />
          </div>
          <CardTitle className="text-sm sm:text-base">{title}</CardTitle>
          {!expanded && summary && (
            <div className="hidden sm:flex items-center gap-2 text-sm text-muted-foreground ml-2 truncate">
              <span className="text-muted-foreground/50">—</span>
              {summary}
            </div>
          )}
        </div>
        {actions && (
          <div
            className="flex items-center gap-2 ml-7 sm:ml-0"
            onPointerDown={(e) => e.stopPropagation()}
          >
            {actions}
          </div>
        )}
      </CardHeader>
      <div
        className={cn(
          "overflow-hidden transition-all duration-300",
          expanded ? "max-h-[1000px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <CardContent className="pt-0 sm:pt-0 px-4 pb-4">{children}</CardContent>
      </div>
    </div>
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

function buildDomainUrl(domain: CustomDomain): string {
  const rawPath = domain.pathPrefix?.trim() ?? "";
  let path = "";
  if (rawPath !== "") {
    path = rawPath.startsWith("/") ? rawPath : `/${rawPath}`;
  }
  return `https://${domain.domain}${path}`;
}

function getOpenAppUrl(
  customDomains: readonly CustomDomain[],
  fallbackUrl: string | null,
): string | null {
  const rootDomain = customDomains.find(
    (domain) => domain.pathPrefix?.trim() === "",
  );
  if (rootDomain) {
    return buildDomainUrl(rootDomain);
  }
  const firstDomain = customDomains[0];
  if (firstDomain) {
    return buildDomainUrl(firstDomain);
  }
  return fallbackUrl;
}

interface DeploymentsSectionProps {
  readonly app: App;
  readonly deployments: readonly Deployment[] | undefined;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly actions: ReturnType<typeof useAppActions>;
  readonly selectedDeploy: Deployment | undefined;
  readonly onSelectDeploy: (id: string | null) => void;
}

function DeploymentsSection({
  app,
  deployments,
  expanded,
  onToggle,
  actions,
  selectedDeploy,
  onSelectDeploy,
}: DeploymentsSectionProps) {
  const successfulDeploys =
    deployments?.filter((d) => d.status === "success").length ?? 0;
  const totalDeploys = deployments?.length ?? 0;
  const latestDeploy = deployments?.[0];

  return (
    <CollapsibleSection
      title="Deployments"
      icon={Rocket}
      expanded={expanded}
      onToggle={onToggle}
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
        <Tabs defaultValue="history" className="w-full min-w-0 overflow-hidden">
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger
              value="history"
              className="gap-1 sm:gap-2 text-xs sm:text-sm"
            >
              <History className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
              <span className="hidden xs:inline">Deploy</span> History
            </TabsTrigger>
            <TabsTrigger
              value="commits"
              className="gap-1 sm:gap-2 text-xs sm:text-sm"
            >
              <GitCommit className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
              Commits
            </TabsTrigger>
          </TabsList>
          <TabsContent value="history" className="mt-0">
            <DeployTimeline appId={app.id} onSelectDeploy={onSelectDeploy} />
          </TabsContent>
          <TabsContent value="commits" className="mt-0">
            <CommitSelectorInline
              appId={app.id}
              onSelect={actions.handleRedeploy}
              disabled={actions.redeploy.isPending}
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
  );
}

interface ContainerLogsSectionProps {
  readonly appId: string;
  readonly appName: string;
  readonly health: ReturnType<typeof useAppHealth>["data"];
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function ContainerLogsSection({
  appId,
  appName,
  health,
  expanded,
  onToggle,
}: ContainerLogsSectionProps) {
  return (
    <CollapsibleSection
      title="Container Logs"
      icon={Terminal}
      expanded={expanded}
      onToggle={onToggle}
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
      <ContainerLogsViewer appId={appId} appName={appName} />
    </CollapsibleSection>
  );
}

interface ResourceUsageSectionProps {
  readonly appId: string;
  readonly containerStats: ReturnType<typeof useContainerStats>["data"];
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function ResourceUsageSection({
  appId,
  containerStats,
  expanded,
  onToggle,
}: ResourceUsageSectionProps) {
  return (
    <CollapsibleSection
      title="Resource Usage"
      icon={Activity}
      expanded={expanded}
      onToggle={onToggle}
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
      <ContainerMetrics appId={appId} embedded />
    </CollapsibleSection>
  );
}

interface EnvVarsSectionProps {
  readonly appId: string;
  readonly envVarsCount: number;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function EnvVarsSection({
  appId,
  envVarsCount,
  expanded,
  onToggle,
}: EnvVarsSectionProps) {
  return (
    <CollapsibleSection
      title="Environment Variables"
      icon={Key}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span>
          {envVarsCount} variable{envVarsCount === 1 ? "" : "s"} configured
        </span>
      }
    >
      <EnvVarsManager appId={appId} embedded />
    </CollapsibleSection>
  );
}

interface ContainerHealthSectionProps {
  readonly appId: string;
  readonly health: ReturnType<typeof useAppHealth>["data"];
  readonly actions: ReturnType<typeof useAppActions>;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function ContainerHealthSection({
  appId,
  health,
  actions,
  expanded,
  onToggle,
}: ContainerHealthSectionProps) {
  return (
    <CollapsibleSection
      title="Container Health"
      icon={HeartPulse}
      expanded={expanded}
      onToggle={onToggle}
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
            onClick={() => actions.restartContainer.mutate(appId)}
            disabled={
              actions.restartContainer.isPending ||
              health?.status === "not_found"
            }
          >
            <RefreshCw
              className={`h-4 w-4 sm:mr-1 ${actions.restartContainer.isPending ? "animate-spin" : ""}`}
            />
            <span className="hidden sm:inline">Restart</span>
          </Button>
          {health?.status === "running" ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => actions.stopContainer.mutate(appId)}
              disabled={actions.stopContainer.isPending}
            >
              <Square className="h-4 w-4 sm:mr-1" />
              <span className="hidden sm:inline">Stop</span>
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => actions.startContainer.mutate(appId)}
              disabled={
                actions.startContainer.isPending ||
                health?.status === "not_found"
              }
            >
              <Play className="h-4 w-4 sm:mr-1" />
              <span className="hidden sm:inline">Start</span>
            </Button>
          )}
        </>
      }
    >
      <HealthDetail health={health} />
    </CollapsibleSection>
  );
}

interface DeploymentConfigSectionProps {
  readonly appConfig: ReturnType<typeof useAppConfig>["data"];
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function DeploymentConfigSection({
  appConfig,
  expanded,
  onToggle,
}: DeploymentConfigSectionProps) {
  if (!appConfig) return null;

  const portDisplay =
    appConfig.hostPort === appConfig.port
      ? appConfig.port
      : `${appConfig.hostPort}:${appConfig.port}`;

  return (
    <CollapsibleSection
      title="Deployment Config"
      icon={Settings}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span>
          Port {portDisplay} • {appConfig.resources.memory} RAM •{" "}
          {appConfig.resources.cpu} CPU
        </span>
      }
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            <Network className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">Port:</span>
            <span className="font-mono">{portDisplay}</span>
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
  );
}

interface WebhookSectionProps {
  readonly webhookId: number | null;
  readonly webhookStatus: ReturnType<typeof useWebhookStatus>["data"];
  readonly actions: ReturnType<typeof useAppActions>;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function WebhookSection({
  webhookId,
  webhookStatus,
  actions,
  expanded,
  onToggle,
}: WebhookSectionProps) {
  const isConfigured = Boolean(webhookId);

  return (
    <CollapsibleSection
      title="GitHub Webhook"
      icon={Link2}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        isConfigured ? (
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
        isConfigured ? (
          <Button
            variant="outline"
            size="sm"
            onClick={actions.handleRemoveWebhook}
            disabled={actions.removeWebhook.isPending}
          >
            <Link2Off className="h-4 w-4 sm:mr-1" />
            <span className="hidden sm:inline">Remove</span>
          </Button>
        ) : (
          <Button
            size="sm"
            onClick={actions.handleSetupWebhook}
            disabled={actions.setupWebhook.isPending}
          >
            <Link2 className="h-4 w-4 sm:mr-1" />
            <span className="hidden sm:inline">Setup</span>
          </Button>
        )
      }
    >
      <div className="flex items-center gap-3">
        {isConfigured ? (
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
              {webhookStatus?.configuredUrl && (
                <p className="text-xs text-muted-foreground font-mono mt-1 break-all">
                  URL: {webhookStatus.configuredUrl}
                </p>
              )}
            </div>
          </>
        )}
      </div>
    </CollapsibleSection>
  );
}

interface DomainsSectionProps {
  readonly appId: string;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function DomainsSection({ appId, expanded, onToggle }: DomainsSectionProps) {
  return (
    <CollapsibleSection
      title="Custom Domains"
      icon={Globe}
      expanded={expanded}
      onToggle={onToggle}
      summary={<span className="text-muted-foreground">Cloudflare DNS</span>}
    >
      <DomainManager appId={appId} />
    </CollapsibleSection>
  );
}

interface NetworksSectionProps {
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly containerId?: string;
  readonly containerNetworks?: readonly string[];
}

function NetworksSection({
  expanded,
  onToggle,
  containerId,
  containerNetworks,
}: NetworksSectionProps) {
  return (
    <CollapsibleSection
      title="Docker Networks"
      icon={Network}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span className="text-muted-foreground">Manage container networks</span>
      }
    >
      <NetworksManager
        containerId={containerId}
        containerNetworks={containerNetworks}
      />
    </CollapsibleSection>
  );
}

interface VolumesSectionProps {
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly containerVolumes?: readonly string[];
}

function VolumesSection({
  expanded,
  onToggle,
  containerVolumes,
}: VolumesSectionProps) {
  return (
    <CollapsibleSection
      title="Docker Volumes"
      icon={HardDrive}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span className="text-muted-foreground">Manage persistent storage</span>
      }
    >
      <VolumesManager containerVolumes={containerVolumes} />
    </CollapsibleSection>
  );
}

function AppDetailsLoading() {
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

function AppNotFound() {
  return (
    <div className="text-center py-12">
      <p className="text-destructive">Application not found</p>
      <Button asChild variant="link">
        <Link to="/">Go back to dashboard</Link>
      </Button>
    </div>
  );
}

interface HeaderActionsProps {
  readonly allExpanded: boolean;
  readonly toggleAllSections: () => void;
  readonly openAppUrl: string | null;
  readonly app: {
    id: string;
    name: string;
    repositoryUrl: string;
    branch: string;
    workdir: string;
    webhookId: number | null;
  };
  readonly actions: ReturnType<typeof useAppActions>;
  readonly hasSuccessfulDeploy: boolean;
}

function AppDescription({ app }: { readonly app: App }) {
  const showWorkdir = app.workdir && app.workdir !== ".";
  return (
    <div className="flex items-center gap-2 sm:gap-4 text-xs sm:text-sm text-muted-foreground flex-wrap">
      <IconText icon={GitBranch} as="span">
        {app.branch}
      </IconText>
      <a
        href={app.repositoryUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center gap-1 hover:text-foreground truncate max-w-[200px] sm:max-w-none"
      >
        <ExternalLink className="h-3.5 w-3.5 sm:h-4 sm:w-4 shrink-0" />
        <span className="truncate">
          {formatRepositoryUrl(app.repositoryUrl)}
        </span>
      </a>
      {showWorkdir && (
        <IconText icon={Folder} as="span">
          <span className="font-mono text-xs truncate max-w-[100px] sm:max-w-none">
            {app.workdir}
          </span>
        </IconText>
      )}
    </div>
  );
}

function HeaderActions({
  allExpanded,
  toggleAllSections,
  openAppUrl,
  app,
  actions,
  hasSuccessfulDeploy,
}: HeaderActionsProps) {
  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={toggleAllSections}
        className="hidden sm:inline-flex"
      >
        {allExpanded ? (
          <ChevronsDownUp className="h-4 w-4" />
        ) : (
          <ChevronsUpDown className="h-4 w-4" />
        )}
        <span className="hidden lg:inline ml-2">
          {allExpanded ? "Collapse All" : "Expand All"}
        </span>
      </Button>
      {openAppUrl && (
        <Button variant="outline" size="sm" asChild>
          <a href={openAppUrl} target="_blank" rel="noopener noreferrer">
            <Globe className="h-4 w-4" />
            <span className="hidden lg:inline ml-2">Open App</span>
          </a>
        </Button>
      )}
      <AppSettingsDialog app={app} />
      <Button
        variant="outline"
        size="sm"
        onClick={actions.handleRollback}
        disabled={actions.rollback.isPending || !hasSuccessfulDeploy}
      >
        <RotateCcw className="h-4 w-4" />
        <span className="hidden lg:inline ml-2">Rollback</span>
      </Button>
      <Button
        size="sm"
        onClick={() => actions.handleRedeploy()}
        disabled={actions.redeploy.isPending}
      >
        <RefreshCw
          className={`h-4 w-4 ${actions.redeploy.isPending ? "animate-spin" : ""}`}
        />
        <span className="hidden lg:inline ml-2">Redeploy</span>
      </Button>
    </>
  );
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
  const { data: containers } = useContainers(true);
  const { data: customDomains = [] } = useQuery({
    queryKey: ["custom-domains", id],
    queryFn: () => api.domains.list(id ?? ""),
    enabled: !!id,
  });

  const { expandedSections, toggleSection, allExpanded, toggleAllSections } =
    useExpandedSections();
  const actions = useAppActions(id);

  const [selectedDeployId, setSelectedDeployId] = useState<string | null>(null);

  const latestDeploy = deployments?.[0];
  const selectedDeploy = selectedDeployId
    ? deployments?.find((d) => d.id === selectedDeployId)
    : latestDeploy;

  if (appLoading) return <AppDetailsLoading />;
  if (!app) return <AppNotFound />;

  const successfulDeploys =
    deployments?.filter((d) => d.status === "success").length ?? 0;
  const hasSuccessfulDeploy = successfulDeploys > 0;
  const openAppUrl = getOpenAppUrl(customDomains, appUrl?.url ?? null);
  const appContainer = containers?.find(
    (container) =>
      container.labels["paasdeploy.app"] === app.name ||
      container.name === app.name,
  );
  const containerId = appContainer?.id;
  const containerNetworks = appContainer?.networks;
  const containerVolumes = appContainer
    ? appContainer.mounts
        .filter((mount) => mount.type === "volume")
        .map((mount) => mount.source)
    : undefined;

  return (
    <div className="space-y-0">
      <div className="mb-3">
        <PageHeader
          backTo="/"
          title={app.name}
          titleSuffix={
            <div className="flex items-center gap-2">
              <HealthIndicator health={health} />
              {latestDeploy && <StatusBadge status={latestDeploy.status} />}
            </div>
          }
          description={<AppDescription app={app} />}
          actions={
            <HeaderActions
              allExpanded={allExpanded}
              toggleAllSections={toggleAllSections}
              openAppUrl={openAppUrl}
              app={app}
              actions={actions}
              hasSuccessfulDeploy={hasSuccessfulDeploy}
            />
          }
        />
      </div>

      <Card className="border border-border rounded-lg divide-y divide-border">
        <DeploymentsSection
          app={app}
          deployments={deployments}
          expanded={expandedSections.deployments ?? false}
          onToggle={() => toggleSection("deployments")}
          actions={actions}
          selectedDeploy={selectedDeploy}
          onSelectDeploy={setSelectedDeployId}
        />

        <ContainerLogsSection
          appId={app.id}
          appName={app.name}
          health={health}
          expanded={expandedSections.containerLogs ?? false}
          onToggle={() => toggleSection("containerLogs")}
        />

        <ResourceUsageSection
          appId={app.id}
          containerStats={containerStats}
          expanded={expandedSections.metrics ?? false}
          onToggle={() => toggleSection("metrics")}
        />

        <EnvVarsSection
          appId={app.id}
          envVarsCount={envVars?.length ?? 0}
          expanded={expandedSections.envVars ?? false}
          onToggle={() => toggleSection("envVars")}
        />

        <ContainerHealthSection
          appId={app.id}
          health={health}
          actions={actions}
          expanded={expandedSections.health ?? false}
          onToggle={() => toggleSection("health")}
        />

        <DeploymentConfigSection
          appConfig={appConfig}
          expanded={expandedSections.config ?? false}
          onToggle={() => toggleSection("config")}
        />

        <WebhookSection
          webhookId={app.webhookId}
          webhookStatus={webhookStatus}
          actions={actions}
          expanded={expandedSections.webhook ?? false}
          onToggle={() => toggleSection("webhook")}
        />

        <DomainsSection
          appId={app.id}
          expanded={expandedSections.domains ?? false}
          onToggle={() => toggleSection("domains")}
        />

        <NetworksSection
          expanded={expandedSections.networks ?? false}
          onToggle={() => toggleSection("networks")}
          containerId={containerId}
          containerNetworks={containerNetworks}
        />

        <VolumesSection
          expanded={expandedSections.volumes ?? false}
          onToggle={() => toggleSection("volumes")}
          containerVolumes={containerVolumes}
        />
      </Card>
    </div>
  );
}
