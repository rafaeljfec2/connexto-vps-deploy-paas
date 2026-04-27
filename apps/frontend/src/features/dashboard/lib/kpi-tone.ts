export type KpiTone = "default" | "success" | "warning" | "destructive";

export type KpiKey = "apps" | "servers" | "containers" | "deploys";

export interface StatsSnapshot {
  readonly totalApps: number;
  readonly totalServers: number;
  readonly onlineServers: number;
  readonly totalContainers: number;
  readonly runningContainers: number;
  readonly successfulDeploys: number;
  readonly failedDeploys: number;
}

export type KpiTones = Record<KpiKey, KpiTone>;

const TONE_SEVERITY: Record<KpiTone, number> = {
  default: 0,
  success: 1,
  warning: 2,
  destructive: 3,
};

export function toneSeverity(tone: KpiTone): number {
  return TONE_SEVERITY[tone];
}

function deriveServersTone(online: number, total: number): KpiTone {
  if (total === 0) return "default";
  if (online === total) return "success";
  if (online === 0) return "destructive";
  return "warning";
}

function deriveContainersTone(running: number, total: number): KpiTone {
  if (total === 0) return "default";
  const ratio = running / total;
  if (ratio >= 0.8) return "success";
  if (ratio >= 0.5) return "warning";
  return "destructive";
}

function deriveDeploysTone(successful: number, failed: number): KpiTone {
  const total = successful + failed;
  if (total === 0) return "default";
  if (failed === 0) return "success";
  if (successful === 0) return "destructive";
  return "warning";
}

export function deriveKpiTones(stats: StatsSnapshot): KpiTones {
  return {
    apps: "default",
    servers: deriveServersTone(stats.onlineServers, stats.totalServers),
    containers: deriveContainersTone(
      stats.runningContainers,
      stats.totalContainers,
    ),
    deploys: deriveDeploysTone(stats.successfulDeploys, stats.failedDeploys),
  };
}
