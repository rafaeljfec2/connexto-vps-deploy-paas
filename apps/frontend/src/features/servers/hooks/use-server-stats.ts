import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";
import { SERVERS_QUERY_KEY } from "./use-servers";

const REFRESH_INTERVAL_MS = 15_000;

export function useServerStats(serverId: string | undefined) {
  return useQuery({
    queryKey: [...SERVERS_QUERY_KEY, serverId, "stats"],
    queryFn: () => api.servers.getStats(serverId!),
    enabled: Boolean(serverId),
    staleTime: 5000,
    refetchInterval: REFRESH_INTERVAL_MS,
  });
}
