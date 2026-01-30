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
