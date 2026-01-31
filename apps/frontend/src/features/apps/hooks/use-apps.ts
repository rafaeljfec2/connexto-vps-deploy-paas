import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { DEFAULTS, STALE_TIMES } from "@/constants/routes";
import { api } from "@/services/api";
import type { CreateAppInput, UpdateAppInput } from "@/types";

export function useApps() {
  return useQuery({
    queryKey: ["apps"],
    queryFn: () => api.apps.list(),
  });
}

export function useApp(id: string) {
  return useQuery({
    queryKey: ["app", id],
    queryFn: () => api.apps.get(id),
    enabled: !!id,
  });
}

export function useCreateApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateAppInput) => api.apps.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}

export function useDeleteApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.apps.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}

export function usePurgeApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.apps.purge(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ["apps"] });
      queryClient.removeQueries({ queryKey: ["app", id] });
    },
  });
}

export function useSetupWebhook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.webhooks.setup(appId),
    onSuccess: (_data, appId) => {
      queryClient.invalidateQueries({ queryKey: ["app", appId] });
      queryClient.invalidateQueries({ queryKey: ["apps"] });
      queryClient.invalidateQueries({ queryKey: ["webhookStatus", appId] });
    },
  });
}

export function useRemoveWebhook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.webhooks.remove(appId),
    onSuccess: (_data, appId) => {
      queryClient.invalidateQueries({ queryKey: ["app", appId] });
      queryClient.invalidateQueries({ queryKey: ["apps"] });
      queryClient.invalidateQueries({ queryKey: ["webhookStatus", appId] });
    },
  });
}

export function useWebhookStatus(appId: string) {
  return useQuery({
    queryKey: ["webhookStatus", appId],
    queryFn: () => api.webhooks.status(appId),
    enabled: !!appId,
  });
}

export function useAppURL(appId: string | undefined) {
  return useQuery({
    queryKey: ["appUrl", appId],
    queryFn: () => api.apps.url(appId!),
    enabled: !!appId,
  });
}

export function useAppConfig(appId: string | undefined) {
  return useQuery({
    queryKey: ["appConfig", appId],
    queryFn: () => api.apps.config(appId!),
    enabled: !!appId,
  });
}

export function useUpdateApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAppInput }) =>
      api.apps.update(id, input),
    onSuccess: (_data, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["app", id] });
      queryClient.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}

export function useRestartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.container.restart(appId),
    onSuccess: (_data, appId) => {
      queryClient.invalidateQueries({ queryKey: ["app-health", appId] });
    },
  });
}

export function useStopContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.container.stop(appId),
    onSuccess: (_data, appId) => {
      queryClient.invalidateQueries({ queryKey: ["app-health", appId] });
    },
  });
}

export function useStartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (appId: string) => api.container.start(appId),
    onSuccess: (_data, appId) => {
      queryClient.invalidateQueries({ queryKey: ["app-health", appId] });
    },
  });
}

export function useContainerLogs(
  appId: string | undefined,
  tail = DEFAULTS.LOGS_TAIL,
) {
  return useQuery({
    queryKey: ["containerLogs", appId, tail],
    queryFn: () => api.container.logs(appId!, tail),
    enabled: !!appId,
    refetchInterval: false,
  });
}

export function useContainerStats(appId: string | undefined) {
  return useQuery({
    queryKey: ["containerStats", appId],
    queryFn: () => api.container.stats(appId!),
    enabled: !!appId,
  });
}

export function useCommits(
  appId: string | undefined,
  limit = DEFAULTS.COMMITS_LIMIT,
) {
  return useQuery({
    queryKey: ["commits", appId, limit],
    queryFn: () => api.apps.commits(appId!, limit),
    enabled: !!appId,
    staleTime: STALE_TIMES.NORMAL,
  });
}
