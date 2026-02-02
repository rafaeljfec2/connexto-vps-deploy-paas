import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";

export interface DockerVolume {
  readonly name: string;
  readonly driver: string;
  readonly mountpoint: string;
  readonly createdAt: string;
  readonly labels: Record<string, string>;
}

export function useVolumes() {
  return useQuery({
    queryKey: ["volumes"],
    queryFn: () => api.volumes.list(),
    staleTime: 30_000,
  });
}

export function useCreateVolume() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => api.volumes.create(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["volumes"] });
    },
  });
}

export function useRemoveVolume() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => api.volumes.remove(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["volumes"] });
    },
  });
}
