import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { HealthIndicator } from "@/components/health-indicator";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import {
  AppDescription,
  ContainerHealthSection,
  EnvVarsSection,
  HeaderActions,
  MobileActionBar,
  ResourceUsageSection,
} from "@/features/apps/components/app-details";
import { AppSettingsSection } from "@/features/apps/components/app-settings-section";
import { ContainerLogsSection } from "@/features/apps/components/container-logs-section";
import { DeploymentsSection } from "@/features/apps/components/deployments-section";
import { useAppActions } from "@/features/apps/hooks/use-app-actions";
import {
  useApp,
  useAppConfig,
  useAppURL,
  useContainerStats,
} from "@/features/apps/hooks/use-apps";
import { useEnvVars } from "@/features/apps/hooks/use-env-vars";
import { useExpandedSections } from "@/features/apps/hooks/use-expanded-sections";
import { getOpenAppUrl } from "@/features/apps/utils/domain-helpers";
import { useContainers } from "@/features/containers/hooks/use-containers";
import { useDeploys } from "@/features/deploys/hooks/use-deploys";
import { useAppHealth } from "@/hooks/use-sse";
import { api } from "@/services/api";

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

export function AppDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const { data: app, isLoading: appLoading } = useApp(id ?? "");
  const { data: deployments } = useDeploys(id ?? "");
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
  const { data: webhookStatus } = useQuery({
    queryKey: ["webhookStatus", id],
    queryFn: () => api.webhooks.status(id ?? ""),
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
    <div className="space-y-0 pb-16 md:pb-0">
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

      <MobileActionBar
        onRedeploy={() => actions.handleRedeploy()}
        onRollback={actions.handleRollback}
        openUrl={openAppUrl}
        isRedeploying={actions.redeploy.isPending}
        isRollingBack={actions.rollback.isPending}
        hasSuccessfulDeploy={hasSuccessfulDeploy}
      />
    </div>
  );
}
