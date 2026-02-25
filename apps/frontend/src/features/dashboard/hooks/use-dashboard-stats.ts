import { useMemo } from "react";
import { useApps } from "@/features/apps/hooks/use-apps";
import { useContainers } from "@/features/containers/hooks/use-containers";
import { useServers } from "@/features/servers/hooks/use-servers";
import type { App, DeploymentSummary } from "@/types";

interface DeployActivity {
  readonly appId: string;
  readonly appName: string;
  readonly deployment: DeploymentSummary;
}

interface DashboardStats {
  readonly totalApps: number;
  readonly totalServers: number;
  readonly onlineServers: number;
  readonly totalContainers: number;
  readonly runningContainers: number;
  readonly recentDeploys: readonly DeployActivity[];
  readonly successfulDeploys: number;
  readonly failedDeploys: number;
  readonly isLoading: boolean;
}

function extractRecentDeploys(apps: readonly App[]): DeployActivity[] {
  return apps
    .filter(
      (app): app is App & { lastDeployment: DeploymentSummary } =>
        app.lastDeployment != null,
    )
    .map((app) => ({
      appId: app.id,
      appName: app.name,
      deployment: app.lastDeployment,
    }))
    .sort((a, b) => {
      const dateA = a.deployment.startedAt ?? a.deployment.finishedAt ?? "";
      const dateB = b.deployment.startedAt ?? b.deployment.finishedAt ?? "";
      return new Date(dateB).getTime() - new Date(dateA).getTime();
    });
}

export function useDashboardStats(): DashboardStats {
  const { data: apps, isLoading: appsLoading } = useApps();
  const { data: servers, isLoading: serversLoading } = useServers();
  const { data: containers, isLoading: containersLoading } = useContainers();

  const recentDeploys = useMemo(() => extractRecentDeploys(apps ?? []), [apps]);

  const successfulDeploys = useMemo(
    () => recentDeploys.filter((d) => d.deployment.status === "success").length,
    [recentDeploys],
  );

  const failedDeploys = useMemo(
    () => recentDeploys.filter((d) => d.deployment.status === "failed").length,
    [recentDeploys],
  );

  return {
    totalApps: apps?.length ?? 0,
    totalServers: servers?.length ?? 0,
    onlineServers: servers?.filter((s) => s.status === "online").length ?? 0,
    totalContainers: containers?.length ?? 0,
    runningContainers:
      containers?.filter((c) => c.state === "running").length ?? 0,
    recentDeploys,
    successfulDeploys,
    failedDeploys,
    isLoading: appsLoading || serversLoading || containersLoading,
  };
}
