import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";

export function useDeploys(appId: string) {
  return useQuery({
    queryKey: ["deployments", appId],
    queryFn: () => api.deployments.list(appId),
    enabled: !!appId,
  });
}

export function useRedeploy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ appId, commitSha }: { appId: string; commitSha?: string }) =>
      api.deployments.redeploy(appId, commitSha),
    onSuccess: (_, { appId }) => {
      queryClient.invalidateQueries({ queryKey: ["deployments", appId] });
      queryClient.invalidateQueries({ queryKey: ["app", appId] });
    },
  });
}

export function useRollback() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.deployments.rollback(appId),
    onSuccess: (_, appId) => {
      queryClient.invalidateQueries({ queryKey: ["deployments", appId] });
      queryClient.invalidateQueries({ queryKey: ["app", appId] });
    },
  });
}
