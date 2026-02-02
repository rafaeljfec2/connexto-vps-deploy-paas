import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";

export function useImages() {
  return useQuery({
    queryKey: ["images"],
    queryFn: () => api.images.list(),
    refetchInterval: 30000,
  });
}

export function useDanglingImages() {
  return useQuery({
    queryKey: ["images", "dangling"],
    queryFn: () => api.images.listDangling(),
    refetchInterval: 30000,
  });
}

export function useRemoveImage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, force = false }: { id: string; force?: boolean }) =>
      api.images.remove(id, force),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["images"] });
    },
  });
}

export function usePruneImages() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.images.prune(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["images"] });
    },
  });
}
