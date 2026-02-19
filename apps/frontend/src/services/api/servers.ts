import type { App, CreateServerInput, Server, ServerStats } from "@/types";
import { API_BASE, fetchApi, fetchApiDelete, fetchApiList } from "./client";

export const serversApi = {
  list: (): Promise<readonly Server[]> =>
    fetchApiList<Server>(`${API_BASE}/servers`),

  get: (id: string): Promise<Server> =>
    fetchApi<Server>(`${API_BASE}/servers/${id}`),

  create: (input: CreateServerInput): Promise<Server> =>
    fetchApi<Server>(`${API_BASE}/servers`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  update: (
    id: string,
    input: {
      name?: string;
      host?: string;
      sshPort?: number;
      sshUser?: string;
      sshKey?: string;
      sshPassword?: string;
      acmeEmail?: string;
    },
  ): Promise<Server> =>
    fetchApi<Server>(`${API_BASE}/servers/${id}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),

  delete: (id: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/servers/${id}`),

  provision: (id: string): Promise<{ message: string }> =>
    fetchApi<{ message: string }>(`${API_BASE}/servers/${id}/provision`, {
      method: "POST",
    }),

  getStats: (id: string): Promise<ServerStats> =>
    fetchApi<ServerStats>(`${API_BASE}/servers/${id}/stats`),

  updateAgent: (id: string): Promise<{ message: string }> =>
    fetchApi<{ message: string }>(`${API_BASE}/servers/${id}/update-agent`, {
      method: "POST",
    }),

  apps: (id: string): Promise<readonly App[]> =>
    fetchApiList<App>(`${API_BASE}/servers/${id}/apps`),
};
