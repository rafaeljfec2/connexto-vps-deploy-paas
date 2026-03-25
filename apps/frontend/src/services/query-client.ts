import { QueryClient } from "@tanstack/react-query";
import { GC_TIMES, STALE_TIMES } from "@/constants/query-config";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: STALE_TIMES.SHORT,
      gcTime: GC_TIMES.DEFAULT,
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});
