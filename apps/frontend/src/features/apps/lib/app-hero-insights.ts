import type { KpiTone } from "@/features/dashboard/lib/kpi-tone";
import { toneSeverity } from "@/features/dashboard/lib/kpi-tone";
import { formatRelativeTime } from "@/lib/format";
import type { ContainerStats, Deployment, HealthStatus } from "@/types";

export type AppKpiKey = "lastDeploy" | "health" | "cpu" | "memory";

export type AppKpiTones = Record<AppKpiKey, KpiTone>;

export type AppHealthState =
  | "healthy"
  | "unhealthy"
  | "starting"
  | "offline"
  | "unknown";

const CPU_WARNING_THRESHOLD = 60;
const CPU_DESTRUCTIVE_THRESHOLD = 85;
const MEMORY_WARNING_THRESHOLD = 60;
const MEMORY_DESTRUCTIVE_THRESHOLD = 85;

export function parseHealthState(
  health: HealthStatus | null | undefined,
): AppHealthState {
  if (!health) return "unknown";
  if (health.status === "not_found" || health.status === "exited") {
    return "offline";
  }
  if (health.status === "paused") return "offline";
  if (health.status === "restarting") return "starting";
  if (health.status === "running") {
    if (health.health === "healthy") return "healthy";
    if (health.health === "unhealthy") return "unhealthy";
    if (health.health === "starting") return "starting";
    return "healthy";
  }
  return "unknown";
}

function deriveLastDeployTone(latest: Deployment | null | undefined): KpiTone {
  if (!latest) return "default";
  switch (latest.status) {
    case "success":
      return "success";
    case "failed":
      return "destructive";
    case "pending":
    case "running":
      return "warning";
    default:
      return "default";
  }
}

function deriveHealthTone(state: AppHealthState): KpiTone {
  switch (state) {
    case "healthy":
      return "success";
    case "unhealthy":
      return "destructive";
    case "starting":
      return "warning";
    case "offline":
      return "destructive";
    default:
      return "default";
  }
}

function deriveUsageTone(
  percent: number | null | undefined,
  warning: number,
  destructive: number,
): KpiTone {
  if (percent == null || Number.isNaN(percent)) return "default";
  if (percent >= destructive) return "destructive";
  if (percent >= warning) return "warning";
  return "success";
}

interface AppKpiInput {
  readonly latestDeploy: Deployment | null | undefined;
  readonly healthState: AppHealthState;
  readonly containerStats: ContainerStats | null | undefined;
}

export function deriveAppKpiTones(input: AppKpiInput): AppKpiTones {
  const cpuPercent = input.containerStats?.cpuPercent;
  const memoryPercent = input.containerStats?.memoryPercent;
  return {
    lastDeploy: deriveLastDeployTone(input.latestDeploy),
    health: deriveHealthTone(input.healthState),
    cpu: deriveUsageTone(
      cpuPercent,
      CPU_WARNING_THRESHOLD,
      CPU_DESTRUCTIVE_THRESHOLD,
    ),
    memory: deriveUsageTone(
      memoryPercent,
      MEMORY_WARNING_THRESHOLD,
      MEMORY_DESTRUCTIVE_THRESHOLD,
    ),
  };
}

const NORTH_STAR_PRIORITY: readonly AppKpiKey[] = [
  "health",
  "lastDeploy",
  "cpu",
  "memory",
];

export function pickAppNorthStar(tones: AppKpiTones): AppKpiKey {
  let best: AppKpiKey = "lastDeploy";
  let bestScore = -1;
  for (const key of NORTH_STAR_PRIORITY) {
    const score = toneSeverity(tones[key]);
    if (score > bestScore) {
      best = key;
      bestScore = score;
    }
  }
  if (bestScore <= toneSeverity("success")) {
    return "lastDeploy";
  }
  return best;
}

interface SubtitleInput {
  readonly latestDeploy: Deployment | null | undefined;
  readonly openAppUrl: string | null;
  readonly branch: string;
}

export function buildAppSubtitle(input: SubtitleInput): string {
  const { latestDeploy, openAppUrl, branch } = input;
  if (!latestDeploy) {
    return "Never deployed yet — trigger your first deploy.";
  }
  const timestamp = latestDeploy.finishedAt ?? latestDeploy.startedAt ?? null;
  const relative = timestamp ? formatRelativeTime(timestamp) : "just now";
  if (openAppUrl) {
    return `Deployed ${relative} · serving at ${openAppUrl}`;
  }
  return `Deployed ${relative} from ${branch}`;
}

export type AppNbaSeverity = "destructive" | "warning";

export type AppNbaAction =
  | "view-logs"
  | "view-health"
  | "view-metrics"
  | "redeploy";

export interface AppNextBestAction {
  readonly message: string;
  readonly ctaLabel: string;
  readonly action: AppNbaAction;
  readonly severity: AppNbaSeverity;
}

interface NbaInput {
  readonly latestDeploy: Deployment | null | undefined;
  readonly healthState: AppHealthState;
  readonly tones: AppKpiTones;
}

export function buildAppNextBestAction(
  input: NbaInput,
): AppNextBestAction | null {
  const { latestDeploy, healthState, tones } = input;

  if (latestDeploy?.status === "failed") {
    return {
      message: "Last deploy failed. Review the build logs or roll back.",
      ctaLabel: "View logs",
      action: "view-logs",
      severity: "destructive",
    };
  }

  if (healthState === "unhealthy") {
    return {
      message: "Container is running but reports as unhealthy.",
      ctaLabel: "View health",
      action: "view-health",
      severity: "destructive",
    };
  }

  if (healthState === "offline") {
    return {
      message: "Container is stopped. Redeploy to bring it back up.",
      ctaLabel: "Redeploy",
      action: "redeploy",
      severity: "warning",
    };
  }

  if (tones.cpu === "destructive" || tones.memory === "destructive") {
    return {
      message: "Resource usage is critical. Inspect metrics before scaling.",
      ctaLabel: "View metrics",
      action: "view-metrics",
      severity: "warning",
    };
  }

  return null;
}
