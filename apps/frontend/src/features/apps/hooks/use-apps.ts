import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { CreateAppInput } from "@/types";

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
