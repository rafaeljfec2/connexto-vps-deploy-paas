import { useCallback, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import { sseClient } from "@/services/sse";
import type { App, Deployment, HealthStatus, SSEEvent } from "@/types";

export function useSSE() {
  const queryClient = useQueryClient();

  const handleEvent = useCallback(
    (event: SSEEvent) => {
      switch (event.type) {
        case "RUNNING":
        case "SUCCESS":
        case "FAILED":
          queryClient.invalidateQueries({ queryKey: ["apps"] });
          queryClient.invalidateQueries({ queryKey: ["app", event.appId] });
          queryClient.invalidateQueries({
            queryKey: ["deployments", event.appId],
          });

          queryClient.setQueryData<App[]>(["apps"], (old) => {
            if (!old) return old;
            return old.map((app) => {
              if (app.id === event.appId) {
                return {
                  ...app,
                  lastDeployedAt:
                    event.type === "SUCCESS"
                      ? event.timestamp
                      : app.lastDeployedAt,
                };
              }
              return app;
            });
          });
          break;

        case "LOG":
          queryClient.setQueryData<Deployment[]>(
            ["deployments", event.appId],
            (old) => {
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
            },
          );
          break;

        case "HEALTH":
          if (event.health) {
            queryClient.setQueryData<HealthStatus>(
              ["app-health", event.appId],
              event.health,
            );
          }
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
    staleTime: 60 * 1000,
  });
}
