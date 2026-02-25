import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";

const REFRESH_INTERVAL_MS = 15_000;

export function useLocalServerStats() {
  return useQuery({
    queryKey: ["system", "stats"],
    queryFn: () => api.system.stats(),
    staleTime: 5000,
    refetchInterval: REFRESH_INTERVAL_MS,
  });
}
