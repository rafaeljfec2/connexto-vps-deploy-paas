import { Box, Rocket, Server, Zap } from "lucide-react";
import { useDashboardStats } from "../hooks/use-dashboard-stats";
import { StatCard } from "./stat-card";

export function StatsOverview() {
  const {
    totalApps,
    totalServers,
    onlineServers,
    runningContainers,
    totalContainers,
    successfulDeploys,
    failedDeploys,
    isLoading,
  } = useDashboardStats();

  const deploySubtitle =
    failedDeploys > 0
      ? `${successfulDeploys} succeeded, ${failedDeploys} failed`
      : `${successfulDeploys} succeeded`;

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatCard
        icon={Rocket}
        title="Applications"
        value={totalApps}
        subtitle={`${totalApps} total`}
        accentColor="default"
        isLoading={isLoading}
      />
      <StatCard
        icon={Server}
        title="Servers"
        value={totalServers}
        subtitle={`${onlineServers} online`}
        accentColor={onlineServers === totalServers ? "success" : "warning"}
        isLoading={isLoading}
      />
      <StatCard
        icon={Box}
        title="Containers"
        value={runningContainers}
        subtitle={`${totalContainers} total`}
        accentColor="success"
        isLoading={isLoading}
      />
      <StatCard
        icon={Zap}
        title="Deployments"
        value={successfulDeploys + failedDeploys}
        subtitle={deploySubtitle}
        accentColor={failedDeploys > 0 ? "warning" : "success"}
        isLoading={isLoading}
      />
    </div>
  );
}
