import type { AgentUpdateState, AgentUpdateStep, SSEEvent } from "@/types";

const state = new Map<string, AgentUpdateState>();
const listeners = new Set<() => void>();

function notify(): void {
  listeners.forEach((cb) => cb());
}

export function applyAgentUpdateEvent(serverId: string, event: SSEEvent): void {
  if (event.type !== "AGENT_UPDATE_STEP") return;

  const step = (event.step ?? "enqueued") as AgentUpdateStep;
  const isCompleted = step === "updated";
  const isError = step === "error";

  const prev = state.get(serverId);
  const startedAt = prev?.startedAt ?? Date.now();

  const resolveStatus = (): AgentUpdateState["status"] => {
    if (isCompleted) return "completed";
    if (isError) return "error";
    return "running";
  };

  state.set(serverId, {
    step,
    status: resolveStatus(),
    version: isCompleted ? event.message : prev?.version,
    errorMessage: isError ? event.message : undefined,
    startedAt,
  });

  notify();
}

export function getAgentUpdateState(
  serverId: string,
): AgentUpdateState | undefined {
  return state.get(serverId);
}

export function clearAgentUpdateState(serverId: string): void {
  state.delete(serverId);
  notify();
}

export function subscribeAgentUpdate(callback: () => void): () => void {
  listeners.add(callback);
  return () => listeners.delete(callback);
}
