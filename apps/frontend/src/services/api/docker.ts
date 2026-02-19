import type { Container, ContainerLogs, CreateContainerInput } from "@/types";
import {
  API_BASE,
  API_URL,
  buildUrl,
  fetchApi,
  fetchApiDelete,
  fetchApiList,
} from "./client";

export interface DockerImageInfo {
  readonly id: string;
  readonly repository: string;
  readonly tag: string;
  readonly size: number;
  readonly created: string;
  readonly containers: number;
  readonly dangling: boolean;
  readonly labels: readonly string[];
}

export interface DockerNetworkInfo {
  readonly name: string;
  readonly id: string;
  readonly driver: string;
  readonly scope: string;
  readonly internal: boolean;
  readonly containers: readonly string[];
}

export interface DockerVolumeInfo {
  readonly name: string;
  readonly driver: string;
  readonly mountpoint: string;
  readonly createdAt: string;
  readonly labels: Record<string, string>;
}

function normalizeNetwork(
  item: string | Partial<DockerNetworkInfo>,
): DockerNetworkInfo {
  if (typeof item === "string") {
    return {
      name: item,
      id: item,
      driver: "unknown",
      scope: "unknown",
      internal: false,
      containers: [],
    };
  }
  return {
    name: item.name ?? "",
    id: item.id ?? item.name ?? "",
    driver: item.driver ?? "unknown",
    scope: item.scope ?? "unknown",
    internal: item.internal ?? false,
    containers: item.containers ?? [],
  };
}

function normalizeVolume(
  item: string | Partial<DockerVolumeInfo>,
): DockerVolumeInfo {
  if (typeof item === "string") {
    return {
      name: item,
      driver: "unknown",
      mountpoint: "",
      createdAt: "",
      labels: {},
    };
  }
  return {
    name: item.name ?? "",
    driver: item.driver ?? "unknown",
    mountpoint: item.mountpoint ?? "",
    createdAt: item.createdAt ?? "",
    labels: item.labels ?? {},
  };
}

export const containersApi = {
  list: (all = true, serverId?: string): Promise<readonly Container[]> =>
    fetchApiList<Container>(
      buildUrl(`${API_BASE}/containers`, { all, serverId }),
    ),

  get: (id: string): Promise<Container> =>
    fetchApi<Container>(`${API_BASE}/containers/${id}`),

  create: (input: CreateContainerInput): Promise<Container> =>
    fetchApi<Container>(`${API_BASE}/containers`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  start: (id: string): Promise<{ message: string; id: string }> =>
    fetchApi<{ message: string; id: string }>(
      `${API_BASE}/containers/${id}/start`,
      { method: "POST" },
    ),

  stop: (id: string): Promise<{ message: string; id: string }> =>
    fetchApi<{ message: string; id: string }>(
      `${API_BASE}/containers/${id}/stop`,
      { method: "POST" },
    ),

  restart: (id: string): Promise<{ message: string; id: string }> =>
    fetchApi<{ message: string; id: string }>(
      `${API_BASE}/containers/${id}/restart`,
      { method: "POST" },
    ),

  remove: (id: string, force = false): Promise<void> =>
    fetchApiDelete(buildUrl(`${API_BASE}/containers/${id}`, { force })),

  logs: (id: string, tail = 100): Promise<ContainerLogs> =>
    fetchApi<ContainerLogs>(`${API_BASE}/containers/${id}/logs?tail=${tail}`),

  logsStreamUrl: (id: string): string =>
    `${API_BASE}/containers/${id}/logs?follow=true`,

  consoleUrl: (id: string, shell = "sh"): string => {
    const base = API_URL.replace(/^http/, "ws");
    return `${base}/paas-deploy/v1/containers/${id}/console?shell=${encodeURIComponent(shell)}`;
  },
};

export const imagesApi = {
  list: (serverId?: string): Promise<readonly DockerImageInfo[]> =>
    fetchApiList<DockerImageInfo>(buildUrl(`${API_BASE}/images`, { serverId })),

  listDangling: (): Promise<readonly DockerImageInfo[]> =>
    fetchApiList<DockerImageInfo>(`${API_BASE}/images/dangling`),

  remove: (
    id: string,
    force = false,
    ref?: string,
    serverId?: string,
  ): Promise<void> =>
    fetchApiDelete(
      buildUrl(`${API_BASE}/images/${encodeURIComponent(id)}`, {
        force,
        ref,
        serverId,
      }),
    ),

  prune: (
    serverId?: string,
  ): Promise<{ imagesDeleted: number; spaceReclaimed: number }> =>
    fetchApi<{ imagesDeleted: number; spaceReclaimed: number }>(
      buildUrl(`${API_BASE}/images/prune`, { serverId }),
      { method: "POST" },
    ),
};

export const networksApi = {
  list: async (serverId?: string): Promise<readonly DockerNetworkInfo[]> => {
    const data = await fetchApiList<string | Partial<DockerNetworkInfo>>(
      buildUrl(`${API_BASE}/networks`, { serverId }),
    );
    return data.map(normalizeNetwork);
  },

  create: (
    name: string,
    serverId?: string,
  ): Promise<{ name: string; id: string }> =>
    fetchApi<{ name: string; id: string }>(
      buildUrl(`${API_BASE}/networks`, { serverId }),
      { method: "POST", body: JSON.stringify({ name }) },
    ),

  remove: (name: string, serverId?: string): Promise<void> =>
    fetchApiDelete(
      buildUrl(`${API_BASE}/networks/${encodeURIComponent(name)}`, {
        serverId,
      }),
    ),

  connectContainer: (
    containerId: string,
    network: string,
  ): Promise<{ message: string }> =>
    fetchApi<{ message: string }>(
      `${API_BASE}/containers/${containerId}/networks`,
      { method: "POST", body: JSON.stringify({ network }) },
    ),

  disconnectContainer: (containerId: string, network: string): Promise<void> =>
    fetchApiDelete(
      `${API_BASE}/containers/${containerId}/networks/${encodeURIComponent(network)}`,
    ),
};

export const volumesApi = {
  list: async (serverId?: string): Promise<readonly DockerVolumeInfo[]> => {
    const data = await fetchApiList<string | Partial<DockerVolumeInfo>>(
      buildUrl(`${API_BASE}/volumes`, { serverId }),
    );
    return data.map(normalizeVolume);
  },

  create: (name: string, serverId?: string): Promise<{ name: string }> =>
    fetchApi<{ name: string }>(buildUrl(`${API_BASE}/volumes`, { serverId }), {
      method: "POST",
      body: JSON.stringify({ name }),
    }),

  remove: (name: string, serverId?: string): Promise<void> =>
    fetchApiDelete(
      buildUrl(`${API_BASE}/volumes/${encodeURIComponent(name)}`, { serverId }),
    ),
};
