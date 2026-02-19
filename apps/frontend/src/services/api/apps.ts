import type {
  App,
  AppConfig,
  AppURL,
  BulkEnvVarInput,
  CommitInfo,
  ContainerActionResult,
  ContainerLogs,
  ContainerStats,
  CreateAppInput,
  CreateEnvVarInput,
  CustomDomain,
  Deployment,
  EnvVar,
  HealthStatus,
  UpdateAppInput,
  WebhookSetupResult,
  WebhookStatus,
} from "@/types";
import { API_BASE, fetchApi, fetchApiDelete, fetchApiList } from "./client";

export const appsApi = {
  list: (): Promise<readonly App[]> => fetchApiList<App>(`${API_BASE}/apps`),

  get: (id: string): Promise<App> => fetchApi<App>(`${API_BASE}/apps/${id}`),

  health: (id: string): Promise<HealthStatus> =>
    fetchApi<HealthStatus>(`${API_BASE}/apps/${id}/health`),

  url: (id: string): Promise<AppURL> =>
    fetchApi<AppURL>(`${API_BASE}/apps/${id}/url`),

  config: (id: string): Promise<AppConfig> =>
    fetchApi<AppConfig>(`${API_BASE}/apps/${id}/config`),

  update: (id: string, input: UpdateAppInput): Promise<App> =>
    fetchApi<App>(`${API_BASE}/apps/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  create: (input: CreateAppInput): Promise<App> =>
    fetchApi<App>(`${API_BASE}/apps`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  delete: (id: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/apps/${id}`),

  purge: (id: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/apps/${id}?purge=true`),

  commits: (id: string, limit = 20): Promise<readonly CommitInfo[]> =>
    fetchApiList<CommitInfo>(`${API_BASE}/apps/${id}/commits?limit=${limit}`),
};

export const deploymentsApi = {
  list: (appId: string): Promise<readonly Deployment[]> =>
    fetchApiList<Deployment>(`${API_BASE}/apps/${appId}/deployments`),

  redeploy: (appId: string, commitSha?: string): Promise<Deployment> =>
    fetchApi<Deployment>(`${API_BASE}/apps/${appId}/redeploy`, {
      method: "POST",
      body: JSON.stringify({ commitSha }),
    }),

  rollback: (appId: string): Promise<Deployment> =>
    fetchApi<Deployment>(`${API_BASE}/apps/${appId}/rollback`, {
      method: "POST",
    }),
};

export const containerApi = {
  restart: (appId: string): Promise<ContainerActionResult> =>
    fetchApi<ContainerActionResult>(
      `${API_BASE}/apps/${appId}/container/restart`,
      { method: "POST" },
    ),

  stop: (appId: string): Promise<ContainerActionResult> =>
    fetchApi<ContainerActionResult>(
      `${API_BASE}/apps/${appId}/container/stop`,
      { method: "POST" },
    ),

  start: (appId: string): Promise<ContainerActionResult> =>
    fetchApi<ContainerActionResult>(
      `${API_BASE}/apps/${appId}/container/start`,
      { method: "POST" },
    ),

  logs: (appId: string, tail = 100): Promise<ContainerLogs> =>
    fetchApi<ContainerLogs>(
      `${API_BASE}/apps/${appId}/container/logs?tail=${tail}`,
    ),

  logsStreamUrl: (appId: string): string =>
    `${API_BASE}/apps/${appId}/container/logs?follow=true`,

  stats: (appId: string): Promise<ContainerStats> =>
    fetchApi<ContainerStats>(`${API_BASE}/apps/${appId}/container/stats`),
};

export const webhooksApi = {
  setup: (appId: string): Promise<WebhookSetupResult> =>
    fetchApi<WebhookSetupResult>(`${API_BASE}/apps/${appId}/webhook`, {
      method: "POST",
    }),

  remove: (appId: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/apps/${appId}/webhook`),

  status: (appId: string): Promise<WebhookStatus> =>
    fetchApi<WebhookStatus>(`${API_BASE}/apps/${appId}/webhook/status`),
};

export const envVarsApi = {
  list: (appId: string): Promise<readonly EnvVar[]> =>
    fetchApiList<EnvVar>(`${API_BASE}/apps/${appId}/env`),

  create: (appId: string, input: CreateEnvVarInput): Promise<EnvVar> =>
    fetchApi<EnvVar>(`${API_BASE}/apps/${appId}/env`, {
      method: "POST",
      body: JSON.stringify(input),
    }),

  bulkUpsert: (
    appId: string,
    input: BulkEnvVarInput,
  ): Promise<readonly EnvVar[]> =>
    fetchApiList<EnvVar>(`${API_BASE}/apps/${appId}/env/bulk`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),

  update: (
    appId: string,
    varId: string,
    input: Partial<CreateEnvVarInput>,
  ): Promise<EnvVar> =>
    fetchApi<EnvVar>(`${API_BASE}/apps/${appId}/env/${varId}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),

  delete: (appId: string, varId: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/apps/${appId}/env/${varId}`),
};

export const domainsApi = {
  list: (appId: string): Promise<readonly CustomDomain[]> =>
    fetchApiList<CustomDomain>(`${API_BASE}/apps/${appId}/domains`),

  add: (
    appId: string,
    domain: string,
    pathPrefix?: string,
  ): Promise<CustomDomain> =>
    fetchApi<CustomDomain>(`${API_BASE}/apps/${appId}/domains`, {
      method: "POST",
      body: JSON.stringify({ domain, pathPrefix }),
    }),

  remove: (appId: string, domainId: string): Promise<void> =>
    fetchApiDelete(`${API_BASE}/apps/${appId}/domains/${domainId}`),
};
