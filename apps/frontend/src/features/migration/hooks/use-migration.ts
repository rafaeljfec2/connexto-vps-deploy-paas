import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";

const QUERY_KEYS = {
  status: ["migration-status"] as const,
} as const;

export function useMigrationStatus() {
  return useQuery({
    queryKey: QUERY_KEYS.status,
    queryFn: () => api.migration.status(),
    refetchInterval: 10000,
  });
}

export function useBackupMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.migration.backup(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status });
    },
  });
}

export function useStopContainersMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ids: readonly string[]) => api.migration.stopContainers(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status });
    },
  });
}

export function useStartContainersMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ids: readonly string[]) => api.migration.startContainers(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status });
    },
  });
}

export function useStopNginxMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.migration.stopNginx(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status });
    },
  });
}

interface MigrateSiteParams {
  readonly siteIndex: number;
  readonly containerId: string;
}

export function useMigrateSiteMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ siteIndex, containerId }: MigrateSiteParams) =>
      api.migration.migrateSite(siteIndex, containerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status });
    },
  });
}
