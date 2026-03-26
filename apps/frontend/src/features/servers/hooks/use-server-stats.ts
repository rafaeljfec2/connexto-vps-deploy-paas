import { useQuery } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/query-config";
import { api } from "@/services/api";
import { SERVERS_QUERY_KEY } from "./use-servers";

export function useServerStats(serverId: string | undefined) {
  return useQuery({
    queryKey: [...SERVERS_QUERY_KEY, serverId, "stats"],
    queryFn: () => api.servers.getStats(serverId!),
    enabled: Boolean(serverId),
    staleTime: STALE_TIMES.REALTIME,
    refetchOnWindowFocus: true,
  });
}
