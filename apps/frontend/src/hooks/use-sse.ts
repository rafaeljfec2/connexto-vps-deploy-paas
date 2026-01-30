import { useCallback, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { sseClient } from "@/services/sse";
import type { App, Deployment, SSEEvent } from "@/types";

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
