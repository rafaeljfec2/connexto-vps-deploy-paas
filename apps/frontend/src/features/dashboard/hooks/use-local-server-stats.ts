import { useQuery } from "@tanstack/react-query";
import { STALE_TIMES } from "@/constants/query-config";
import { api } from "@/services/api";

export function useLocalServerStats() {
  return useQuery({
    queryKey: ["system", "stats"],
    queryFn: () => api.system.stats(),
    staleTime: STALE_TIMES.REALTIME,
    refetchOnWindowFocus: true,
  });
}
