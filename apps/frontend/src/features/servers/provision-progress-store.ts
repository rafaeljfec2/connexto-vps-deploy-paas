import type {
  ProvisionProgressState,
  ProvisionStepState,
  SSEEvent,
} from "@/types";

const STEP_ORDER = [
  "ssh_connect",
  "remote_env",
  "sftp_client",
  "install_dir",
  "agent_certs",
  "agent_binary",
  "systemd_unit",
  "start_agent",
] as const;

const progress = new Map<string, ProvisionProgressState>();
const listeners = new Set<() => void>();

function getStepIndex(step: string): number {
  const i = STEP_ORDER.indexOf(step as (typeof STEP_ORDER)[number]);
  return i >= 0 ? i : 999;
}

function sortSteps(steps: ProvisionStepState[]): ProvisionStepState[] {
  return [...steps].sort((a, b) => getStepIndex(a.step) - getStepIndex(b.step));
}

function updateProgress(
  serverId: string,
  updater: (prev: ProvisionProgressState) => ProvisionProgressState,
): void {
  const prev = progress.get(serverId) ?? {
    steps: [],
    logs: [],
    status: "running" as const,
  };
  const next = updater(prev);
  progress.set(serverId, next);
  listeners.forEach((cb) => cb());
}

export function applyProvisionEvent(serverId: string, event: SSEEvent): void {
  switch (event.type) {
    case "PROVISION_STEP":
      updateProgress(serverId, (prev) => {
        const steps = [...prev.steps];
        const idx = steps.findIndex((s) => s.step === event.step);
        const newStep: ProvisionStepState = {
          step: event.step ?? "",
          status: event.status ?? "",
          message: event.message ?? "",
        };
        if (idx >= 0) {
          steps[idx] = newStep;
        } else {
          steps.push(newStep);
        }
        return {
          ...prev,
          steps: sortSteps(steps),
        };
      });
      break;
    case "PROVISION_LOG":
      updateProgress(serverId, (prev) => ({
        ...prev,
        logs: [...prev.logs, event.message ?? ""],
      }));
      break;
    case "PROVISION_COMPLETED":
      updateProgress(serverId, (prev) => ({
        ...prev,
        status: "completed",
      }));
      break;
    case "PROVISION_FAILED":
      updateProgress(serverId, (prev) => ({
        ...prev,
        status: "failed",
        logs: [...prev.logs, event.message ?? "Provision failed"],
      }));
      break;
  }
}

export function getProvisionProgress(
  serverId: string,
): ProvisionProgressState | undefined {
  return progress.get(serverId);
}

export function clearProvisionProgress(serverId: string): void {
  progress.delete(serverId);
  listeners.forEach((cb) => cb());
}

export function subscribeProvisionProgress(callback: () => void): () => void {
  listeners.add(callback);
  return () => listeners.delete(callback);
}
