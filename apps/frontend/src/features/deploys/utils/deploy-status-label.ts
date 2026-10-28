import type { DeployStatus } from "@/types";

export type DeployStatusLabel = "Ready" | "Error" | "Building" | "Cancelled";

export type DeployStatusTone = "success" | "error" | "running" | "muted";

interface DeployStatusPresentation {
  readonly label: DeployStatusLabel;
  readonly tone: DeployStatusTone;
}

const STATUS_PRESENTATION: Record<DeployStatus, DeployStatusPresentation> = {
  success: { label: "Ready", tone: "success" },
  failed: { label: "Error", tone: "error" },
  running: { label: "Building", tone: "running" },
  pending: { label: "Building", tone: "running" },
  cancelled: { label: "Cancelled", tone: "muted" },
};

export function getDeployStatusPresentation(
  status: DeployStatus,
): DeployStatusPresentation {
  return STATUS_PRESENTATION[status];
}

export function getDeployDurationMs(
  startedAt: string | null,
  finishedAt: string | null,
  nowMs: number = Date.now(),
): number | null {
  if (!startedAt) return null;
  const start = new Date(startedAt).getTime();
  if (Number.isNaN(start)) return null;
  if (finishedAt) {
    const end = new Date(finishedAt).getTime();
    if (Number.isNaN(end) || end < start) return null;
    return end - start;
  }
  return Math.max(0, nowMs - start);
}
