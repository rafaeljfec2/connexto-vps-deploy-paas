import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  ChevronsDownUp,
  ChevronsUpDown,
  ExternalLink,
  Folder,
  GitBranch,
  Globe,
  HeartPulse,
  Key,
  Play,
  RefreshCw,
  RotateCcw,
  Square,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ContainerMetrics } from "@/components/container-metrics";
import { HealthDetail, HealthIndicator } from "@/components/health-indicator";
import { IconText } from "@/components/icon-text";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { AppSettingsDialog } from "@/features/apps/components/app-settings-dialog";
import { AppSettingsSection } from "@/features/apps/components/app-settings-section";
import { CollapsibleSection } from "@/features/apps/components/collapsible-section";
import { ContainerLogsSection } from "@/features/apps/components/container-logs-section";
import { DeploymentsSection } from "@/features/apps/components/deployments-section";
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
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { useAppHealth } from "@/hooks/use-sse";
import { formatBytes } from "@/lib/format";
import { formatRepositoryUrl } from "@/lib/utils";
import { api } from "@/services/api";
import type { App, CustomDomain } from "@/types";

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
  const { data: containers } = useContainers(true, app?.serverId);
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

        <AppSettingsSection
          appConfig={appConfig}
          webhookId={app.webhookId}
          webhookStatus={webhookStatus}
          actions={actions}
          appId={app.id}
          containerId={containerId}
          containerNetworks={containerNetworks}
          containerVolumes={containerVolumes}
          serverId={app.serverId}
          expandedSections={{
            config: expandedSections.config,
            webhook: expandedSections.webhook,
            domains: expandedSections.domains,
            networks: expandedSections.networks,
            volumes: expandedSections.volumes,
          }}
          toggleSection={toggleSection}
        />
      </Card>
    </div>
  );
}
