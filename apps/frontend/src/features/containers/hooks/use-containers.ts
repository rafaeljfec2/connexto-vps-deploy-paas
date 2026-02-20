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

export function useContainerLogs(
  id: string | undefined,
  tail = 100,
  serverId?: string,
) {
  return useQuery({
    queryKey: ["containers", id, "logs", tail, serverId],
    queryFn: () => api.containers.logs(id!, tail, serverId),
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

interface ContainerActionInput {
  readonly id: string;
  readonly serverId?: string;
}

export function useStartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, serverId }: ContainerActionInput) =>
      api.containers.start(id, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useStopContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, serverId }: ContainerActionInput) =>
      api.containers.stop(id, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useRestartContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, serverId }: ContainerActionInput) =>
      api.containers.restart(id, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

interface RemoveContainerInput {
  readonly id: string;
  readonly force?: boolean;
  readonly serverId?: string;
}

export function useRemoveContainer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, force = false, serverId }: RemoveContainerInput) =>
      api.containers.remove(id, force, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
