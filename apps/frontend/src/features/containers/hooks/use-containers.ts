import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import type { CreateContainerInput } from "@/types";

export function useContainers(all = true, serverId?: string) {
  return useQuery({
    queryKey: ["containers", all, serverId],
    queryFn: () => api.containers.list(all, serverId),
    refetchInterval: 10000,
    refetchOnWindowFocus: true,
  });
}

export function useContainer(id: string | undefined) {
  return useQuery({
    queryKey: ["containers", id],
    queryFn: () => api.containers.get(id!),
    enabled: Boolean(id),
  });
}

export function useContainerLogs(id: string | undefined, tail = 100) {
  return useQuery({
    queryKey: ["containers", id, "logs", tail],
    queryFn: () => api.containers.logs(id!, tail),
    enabled: Boolean(id),
    refetchInterval: 5000,
  });
}

export function useCreateContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateContainerInput) => api.containers.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useStartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.containers.start(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useStopContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.containers.stop(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useRestartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.containers.restart(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useRemoveContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, force = false }: { id: string; force?: boolean }) =>
      api.containers.remove(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
