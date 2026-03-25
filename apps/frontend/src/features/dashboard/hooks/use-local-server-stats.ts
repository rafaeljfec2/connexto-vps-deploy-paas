import { useQuery } from "@tanstack/react-query";
import { REFETCH_INTERVALS, STALE_TIMES } from "@/constants/query-config";
import { api } from "@/services/api";

export function useLocalServerStats() {
  return useQuery({
    queryKey: ["system", "stats"],
    queryFn: () => api.system.stats(),
    staleTime: STALE_TIMES.REALTIME,
    refetchInterval: REFETCH_INTERVALS.STATS,
  });
}
