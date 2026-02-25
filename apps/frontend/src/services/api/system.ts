import type { ServerStats } from "@/types";
import { API_BASE, fetchApi } from "./client";

export const systemApi = {
  stats: (): Promise<ServerStats> =>
    fetchApi<ServerStats>(`${API_BASE}/system/stats`),
};
