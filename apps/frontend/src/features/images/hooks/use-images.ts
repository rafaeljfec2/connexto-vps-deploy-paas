import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { REFETCH_INTERVALS } from "@/constants/query-config";
import { api } from "@/services/api";

export function useImages(serverId?: string) {
  return useQuery({
    queryKey: ["images", serverId],
    queryFn: () => api.images.list(serverId),
    refetchInterval: REFETCH_INTERVALS.SLOW,
  });
}

export function useDanglingImages() {
  return useQuery({
    queryKey: ["images", "dangling"],
    queryFn: () => api.images.listDangling(),
    refetchInterval: REFETCH_INTERVALS.SLOW,
  });
}

interface RemoveImageInput {
  readonly id: string;
  readonly force?: boolean;
  readonly ref?: string;
  readonly serverId?: string;
}

export function useRemoveImage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, force = false, ref, serverId }: RemoveImageInput) =>
      api.images.remove(id, force, ref, serverId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["images"] });
    },
  });
}

export function usePruneImages() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (serverId?: string) => api.images.prune(serverId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["images"] });
    },
  });
}
