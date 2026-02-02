import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";

export interface DockerNetwork {
  readonly name: string;
  readonly id: string;
  readonly driver: string;
  readonly scope: string;
  readonly internal: boolean;
  readonly containers: readonly string[];
}

export function useNetworks() {
  return useQuery({
    queryKey: ["networks"],
    queryFn: () => api.networks.list(),
    staleTime: 30_000,
  });
}

export function useCreateNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => api.networks.create(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
    },
  });
}

export function useRemoveNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => api.networks.remove(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
    },
  });
}

export function useConnectContainerToNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      containerId,
      network,
    }: {
      containerId: string;
      network: string;
    }) => api.networks.connectContainer(containerId, network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}

export function useDisconnectContainerFromNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      containerId,
      network,
    }: {
      containerId: string;
      network: string;
    }) => api.networks.disconnectContainer(containerId, network),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
