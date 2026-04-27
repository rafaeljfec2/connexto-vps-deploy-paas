import { ROUTES } from "@/constants/routes";
import { formatRelativeTime } from "@/lib/format";
import type { DeploymentSummary } from "@/types";
import type { KpiKey, KpiTones } from "./kpi-tone";
import { toneSeverity } from "./kpi-tone";

const NORTH_STAR_PRIORITY: readonly KpiKey[] = [
  "servers",
  "containers",
  "deploys",
  "apps",
];

export function pickNorthStar(tones: KpiTones): KpiKey {
  let best: KpiKey = "apps";
  let bestScore = -1;

  for (const key of NORTH_STAR_PRIORITY) {
    const score = toneSeverity(tones[key]);
    if (score > bestScore) {
      best = key;
      bestScore = score;
    }
  }

  if (bestScore <= toneSeverity("success")) {
    return "servers";
  }
  return best;
}

interface DeployActivityLike {
  readonly appId: string;
  readonly appName: string;
  readonly deployment: DeploymentSummary;
}

export function buildHeroSubtitle(
  recentDeploys: readonly DeployActivityLike[],
  totalApps: number,
  totalServers: number,
): string {
  const [latest] = recentDeploys;
  if (latest) {
    const timestamp =
      latest.deployment.startedAt ?? latest.deployment.finishedAt ?? null;
    const relative = timestamp ? formatRelativeTime(timestamp) : "just now";
    return `Last deploy · ${latest.appName} · ${relative}`;
  }
  if (totalApps > 0) {
    return `${totalApps} apps · ${totalServers} servers · no deploys yet`;
  }
  return "Let's get your first app deployed.";
}

export type NbaSeverity = "destructive" | "warning";

export interface NextBestAction {
  readonly message: string;
  readonly ctaLabel: string;
  readonly ctaHref: string;
  readonly severity: NbaSeverity;
}

interface NbaInput {
  readonly tones: KpiTones;
  readonly onlineServers: number;
  readonly totalServers: number;
  readonly runningContainers: number;
  readonly totalContainers: number;
  readonly successfulDeploys: number;
  readonly failedDeploys: number;
}

export function buildNextBestAction(input: NbaInput): NextBestAction | null {
  const {
    tones,
    onlineServers,
    totalServers,
    runningContainers,
    totalContainers,
    failedDeploys,
  } = input;

  const offlineServers = Math.max(0, totalServers - onlineServers);
  const stoppedContainers = Math.max(0, totalContainers - runningContainers);

  if (tones.servers === "destructive") {
    return {
      message:
        "All servers are offline. Restore connectivity to resume deploys.",
      ctaLabel: "View servers",
      ctaHref: ROUTES.SERVERS,
      severity: "destructive",
    };
  }

  if (tones.containers === "destructive") {
    return {
      message: `${stoppedContainers} of ${totalContainers} containers are stopped. Inspect logs and restart.`,
      ctaLabel: "View containers",
      ctaHref: ROUTES.CONTAINERS,
      severity: "destructive",
    };
  }

  if (tones.servers === "warning") {
    const label = offlineServers === 1 ? "server" : "servers";
    return {
      message: `${offlineServers} of ${totalServers} ${label} offline. Check network reachability.`,
      ctaLabel: "View servers",
      ctaHref: ROUTES.SERVERS,
      severity: "warning",
    };
  }

  if (failedDeploys > 0) {
    const label = failedDeploys === 1 ? "deploy" : "deploys";
    return {
      message: `${failedDeploys} failed ${label} today. Review the audit log.`,
      ctaLabel: "Open audit log",
      ctaHref: ROUTES.AUDIT,
      severity: "warning",
    };
  }

  if (tones.containers === "warning") {
    return {
      message: `${stoppedContainers} of ${totalContainers} containers stopped.`,
      ctaLabel: "View containers",
      ctaHref: ROUTES.CONTAINERS,
      severity: "warning",
    };
  }

  return null;
}
