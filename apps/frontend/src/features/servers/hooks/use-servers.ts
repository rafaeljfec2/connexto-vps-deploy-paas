import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/routes";
import { api } from "@/services/api";
import type { CreateServerInput } from "@/types";

const SERVERS_QUERY_KEY = ["servers"] as const;

export function useServers() {
  return useQuery({
    queryKey: SERVERS_QUERY_KEY,
    queryFn: () => api.servers.list(),
    staleTime: STALE_TIMES.NORMAL,
  });
}

export function useServer(id: string | undefined) {
  return useQuery({
    queryKey: [...SERVERS_QUERY_KEY, id],
    queryFn: () => api.servers.get(id!),
    enabled: Boolean(id),
    staleTime: STALE_TIMES.SHORT,
  });
}

export function useCreateServer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateServerInput) => api.servers.create(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SERVERS_QUERY_KEY });
    },
  });
}

export function useDeleteServer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.servers.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SERVERS_QUERY_KEY });
    },
  });
}

export function useProvisionServer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.servers.provision(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SERVERS_QUERY_KEY });
    },
  });
}
