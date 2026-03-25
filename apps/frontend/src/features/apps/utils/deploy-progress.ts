import type { DeployStatus } from "@/types";

type DeployIcon = "loader" | "rocket" | "hammer";

interface DeployProgressState {
  readonly phase: string;
  readonly progress: number;
  readonly icon: DeployIcon;
}

export function deriveDeployProgressState(
  logs: string | null | undefined,
  status: DeployStatus,
): DeployProgressState | null {
  const isRunning = status === "running";
  const isPending = status === "pending";

  if (!isRunning && !isPending) return null;

  const rawLogs = logs ?? "";
  const isBuildPhase =
    rawLogs.includes("[build]") && !rawLogs.includes("Container deployed");
  const isDeployPhase =
    rawLogs.includes("Deploying container") || rawLogs.includes("[deploy]");
  const isHealthCheck = rawLogs.includes("Waiting for health check");

  let phase = "Initializing";
  let icon: DeployIcon = "loader";
  let progress = 10;

  if (isPending) {
    phase = "Queued";
    progress = 5;
  } else if (isHealthCheck) {
    phase = "Health check";
    icon = "rocket";
    progress = 90;
  } else if (isDeployPhase) {
    phase = "Deploying";
    icon = "rocket";
    progress = 75;
  } else if (isBuildPhase) {
    phase = "Building";
    icon = "hammer";
    const buildSteps = rawLogs.match(/Step \d+\/\d+/g) ?? [];
    const lastStep = buildSteps.at(-1);
    if (lastStep) {
      const match = /Step (\d+)\/(\d+)/.exec(lastStep);
      const currentStep = match?.[1];
      const totalSteps = match?.[2];
      if (currentStep && totalSteps) {
        const current = Number.parseInt(currentStep, 10);
        const total = Number.parseInt(totalSteps, 10);
        progress = Math.round((current / total) * 60) + 15;
        phase = `Building (${current}/${total})`;
      }
    } else {
      progress = 20;
    }
  }

  return { phase, progress, icon };
}
