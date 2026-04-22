import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/query-config";
import { api } from "@/services/api";

export interface DockerNetwork {
  readonly name: string;
  readonly id: string;
  readonly driver: string;
  readonly scope: string;
  readonly internal: boolean;
  readonly containers: readonly string[];
}

export function useNetworks(serverId?: string) {
  return useQuery({
    queryKey: ["networks", serverId],
    queryFn: () => api.networks.list(serverId),
    staleTime: STALE_TIMES.SHORT,
  });
}

export function useCreateNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, serverId }: { name: string; serverId?: string }) =>
      api.networks.create(name, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
    },
  });
}

export function useRemoveNetwork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, serverId }: { name: string; serverId?: string }) =>
      api.networks.remove(name, serverId),
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
      serverId,
    }: {
      containerId: string;
      network: string;
      serverId?: string;
    }) => api.networks.connectContainer(containerId, network, serverId),
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
      serverId,
    }: {
      containerId: string;
      network: string;
      serverId?: string;
    }) => api.networks.disconnectContainer(containerId, network, serverId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["networks"] });
      queryClient.invalidateQueries({ queryKey: ["containers"] });
    },
  });
}
