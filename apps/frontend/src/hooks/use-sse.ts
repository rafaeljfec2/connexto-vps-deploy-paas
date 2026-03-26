import { useCallback, useEffect } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/query-config";
import { applyAgentUpdateEvent } from "@/features/servers/agent-update-store";
import { applyProvisionEvent } from "@/features/servers/provision-progress-store";
import { api } from "@/services/api";
import { sseClient } from "@/services/sse";
import type {
  App,
  ContainerStats,
  DeployStatus,
  Deployment,
  HealthStatus,
  SSEEvent,
  ServerStats,
} from "@/types";

function deployStatusFromEvent(type: string): DeployStatus | undefined {
  switch (type) {
    case "RUNNING":
      return "running";
    case "SUCCESS":
      return "success";
    case "FAILED":
      return "failed";
    default:
      return undefined;
  }
}

function handleDeployEvent(qc: QueryClient, event: SSEEvent) {
  qc.invalidateQueries({ queryKey: ["apps"] });
  qc.invalidateQueries({ queryKey: ["app", event.appId] });
  qc.invalidateQueries({ queryKey: ["app-health", event.appId] });

  const status = deployStatusFromEvent(event.type);

  qc.setQueryData<Deployment[]>(["deployments", event.appId], (old) => {
    if (!old) return old;
    const exists = old.some((d) => d.id === event.deployId);
    if (!exists) return old;

    return old.map((deploy) => {
      if (deploy.id !== event.deployId) return deploy;
      return {
        ...deploy,
        ...(status ? { status } : {}),
        ...(event.type === "SUCCESS" ? { finishedAt: event.timestamp } : {}),
        ...(event.type === "FAILED"
          ? { finishedAt: event.timestamp, errorMessage: event.message }
          : {}),
      };
    });
  });

  qc.invalidateQueries({ queryKey: ["deployments", event.appId] });

  qc.setQueryData<App[]>(["apps"], (old) => {
    if (!old) return old;
    return old.map((app) => {
      if (app.id === event.appId) {
        return {
          ...app,
          lastDeployedAt:
            event.type === "SUCCESS" ? event.timestamp : app.lastDeployedAt,
        };
      }
      return app;
    });
  });
}

function handleLogEvent(qc: QueryClient, event: SSEEvent) {
  const current = qc.getQueryData<Deployment[]>(["deployments", event.appId]);

  if (!current?.some((d) => d.id === event.deployId)) {
    qc.invalidateQueries({ queryKey: ["deployments", event.appId] });
    return;
  }

  qc.setQueryData<Deployment[]>(["deployments", event.appId], (old) => {
    if (!old) return old;
    return old.map((deploy) => {
      if (deploy.id === event.deployId) {
        return {
          ...deploy,
          logs: (deploy.logs ?? "") + (event.message ?? "") + "\n",
        };
      }
      return deploy;
    });
  });
}

function handleProvisionEvent(qc: QueryClient, event: SSEEvent) {
  if (!event.serverId) return;
  applyProvisionEvent(event.serverId, event);

  const isTerminal =
    event.type === "PROVISION_COMPLETED" || event.type === "PROVISION_FAILED";
  if (isTerminal) {
    qc.invalidateQueries({ queryKey: ["servers"] });
  }
}

function handleAgentUpdateEvent(qc: QueryClient, event: SSEEvent) {
  if (!event.serverId) return;
  applyAgentUpdateEvent(event.serverId, event);

  if (event.step === "updated") {
    qc.invalidateQueries({ queryKey: ["server", event.serverId] });
    qc.invalidateQueries({ queryKey: ["servers"] });
  }
}

export function useSSE() {
  const queryClient = useQueryClient();

  const handleEvent = useCallback(
    (event: SSEEvent) => {
      switch (event.type) {
        case "RUNNING":
        case "SUCCESS":
        case "FAILED":
          handleDeployEvent(queryClient, event);
          break;

        case "LOG":
          handleLogEvent(queryClient, event);
          break;

        case "HEALTH":
          if (event.health) {
            queryClient.setQueryData<HealthStatus>(
              ["app-health", event.appId],
              event.health,
            );
          }
          break;

        case "STATS":
          if (event.stats && event.appId) {
            queryClient.setQueryData<ContainerStats>(
              ["containerStats", event.appId],
              event.stats,
            );
          }
          break;

        case "SYSTEM_STATS":
          if (event.systemStats) {
            queryClient.setQueryData<ServerStats>(
              ["system", "stats"],
              event.systemStats,
            );
          }
          break;

        case "SERVER_STATS":
          if (event.systemStats && event.serverId) {
            queryClient.setQueryData<ServerStats>(
              ["servers", event.serverId, "stats"],
              event.systemStats,
            );
          }
          break;

        case "INVALIDATE":
          if (event.resource) {
            queryClient.invalidateQueries({ queryKey: [event.resource] });
          }
          break;

        case "PROVISION_STEP":
        case "PROVISION_LOG":
        case "PROVISION_COMPLETED":
        case "PROVISION_FAILED":
          handleProvisionEvent(queryClient, event);
          break;

        case "AGENT_UPDATE_STEP":
          handleAgentUpdateEvent(queryClient, event);
          break;
      }
    },
    [queryClient],
  );

  useEffect(() => {
    sseClient.connect();
    const unsubscribe = sseClient.subscribe(handleEvent);

    return () => {
      unsubscribe();
    };
  }, [handleEvent]);

  return { isConnected: sseClient.isConnected };
}

export function useAppHealth(appId: string | undefined) {
  return useQuery<HealthStatus | null>({
    queryKey: ["app-health", appId],
    queryFn: async () => {
      if (!appId) return null;
      return api.apps.health(appId);
    },
    enabled: !!appId,
    staleTime: STALE_TIMES.NORMAL,
  });
}
