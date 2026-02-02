import type { User } from "@/contexts/auth-context";
import type {
  ApiEnvelope,
  App,
  AppConfig,
  AppURL,
  BackupResult,
  BulkEnvVarInput,
  CertificateStatus,
  CloudflareStatus,
  CommitInfo,
  Container,
  ContainerActionResult,
  ContainerLogs,
  ContainerStats,
  CreateAppInput,
  CreateContainerInput,
  CreateEnvVarInput,
  CustomDomain,
  DeployTemplateInput,
  Deployment,
  EnvVar,
  HealthStatus,
  MigrateResult,
  MigrationStatus,
  Template,
  TraefikPreview,
  UpdateAppInput,
  WebhookSetupResult,
  WebhookStatus,
} from "@/types";
import { ApiError, isApiError } from "@/types";

const API_URL = import.meta.env.VITE_API_URL ?? "";
const API_BASE = `${API_URL}/paas-deploy/v1`;

export interface GitHubInstallation {
  readonly id: string;
  readonly installationId: number;
  readonly accountType: string;
  readonly accountLogin: string;
  readonly repositorySelection: string;
}

export interface GitHubRepository {
  readonly id: number;
  readonly name: string;
  readonly fullName: string;
  readonly private: boolean;
  readonly description: string;
  readonly htmlUrl: string;
  readonly cloneUrl: string;
  readonly defaultBranch: string;
  readonly language: string;
  readonly owner: {
    readonly login: string;
    readonly avatarUrl: string;
    readonly type: string;
  };
}

export interface ReposResponse {
  readonly repositories: readonly GitHubRepository[];
  readonly needInstall: boolean;
  readonly installMessage?: string;
}

async function fetchApi<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const envelope: ApiEnvelope<T> = await response.json();

  if (!response.ok || isApiError(envelope)) {
    throw ApiError.fromResponse(envelope, response.status);
  }

  return envelope.data as T;
}

async function fetchApiList<T>(
  url: string,
  options?: RequestInit,
): Promise<readonly T[]> {
  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  const envelope: ApiEnvelope<readonly T[]> = await response.json();

  if (!response.ok || isApiError(envelope)) {
    throw ApiError.fromResponse(envelope, response.status);
  }

  return envelope.data ?? [];
}

export const api = {
  auth: {
    me: (): Promise<User> => fetchApi<User>(`${API_URL}/auth/me`),

    logout: async (): Promise<void> => {
      const response = await fetch(`${API_URL}/auth/logout`, {
        method: "POST",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },

  github: {
    installations: (): Promise<readonly GitHubInstallation[]> =>
      fetchApiList<GitHubInstallation>(`${API_URL}/api/github/installations`),

    repos: (installationId?: string): Promise<ReposResponse> => {
      const url = installationId
        ? `${API_URL}/api/github/repos?installation_id=${installationId}`
        : `${API_URL}/api/github/repos`;
      return fetchApi<ReposResponse>(url);
    },

    repo: (owner: string, repo: string): Promise<GitHubRepository> =>
      fetchApi<GitHubRepository>(
        `${API_URL}/api/github/repos/${owner}/${repo}`,
      ),
  },

  apps: {
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

    delete: async (id: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    purge: async (id: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${id}?purge=true`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    commits: (id: string, limit = 20): Promise<readonly CommitInfo[]> =>
      fetchApiList<CommitInfo>(`${API_BASE}/apps/${id}/commits?limit=${limit}`),
  },

  deployments: {
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
  },

  container: {
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
  },

  webhooks: {
    setup: (appId: string): Promise<WebhookSetupResult> =>
      fetchApi<WebhookSetupResult>(`${API_BASE}/apps/${appId}/webhook`, {
        method: "POST",
      }),

    remove: async (appId: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${appId}/webhook`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    status: (appId: string): Promise<WebhookStatus> =>
      fetchApi<WebhookStatus>(`${API_BASE}/apps/${appId}/webhook/status`),
  },

  envVars: {
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

    delete: async (appId: string, varId: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${appId}/env/${varId}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },

  cloudflare: {
    status: (): Promise<CloudflareStatus> =>
      fetchApi<CloudflareStatus>(`${API_URL}/auth/cloudflare/status`),

    connect: (apiToken: string): Promise<CloudflareStatus> =>
      fetchApi<CloudflareStatus>(`${API_URL}/auth/cloudflare/connect`, {
        method: "POST",
        body: JSON.stringify({ apiToken }),
      }),

    disconnect: async (): Promise<void> => {
      const response = await fetch(`${API_URL}/auth/cloudflare/disconnect`, {
        method: "POST",
        credentials: "include",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },

  domains: {
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

    remove: async (appId: string, domainId: string): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/apps/${appId}/domains/${domainId}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },

  certificates: {
    list: (): Promise<readonly CertificateStatus[]> =>
      fetchApiList<CertificateStatus>(`${API_URL}/api/certificates`),

    getStatus: (domain: string): Promise<CertificateStatus> =>
      fetchApi<CertificateStatus>(
        `${API_URL}/api/certificates/${encodeURIComponent(domain)}`,
      ),
  },

  migration: {
    status: (): Promise<MigrationStatus> =>
      fetchApi<MigrationStatus>(`${API_BASE}/migration/status`),

    backup: (): Promise<BackupResult> =>
      fetchApi<BackupResult>(`${API_BASE}/migration/backup`, {
        method: "POST",
      }),

    stopContainers: (
      containerIds: readonly string[],
    ): Promise<{ message: string; stopped: readonly string[] }> =>
      fetchApi<{ message: string; stopped: readonly string[] }>(
        `${API_BASE}/migration/containers/stop`,
        {
          method: "POST",
          body: JSON.stringify({ containerIds }),
        },
      ),

    startContainers: (
      containerIds: readonly string[],
    ): Promise<{ message: string; started: readonly string[] }> =>
      fetchApi<{ message: string; started: readonly string[] }>(
        `${API_BASE}/migration/containers/start`,
        {
          method: "POST",
          body: JSON.stringify({ containerIds }),
        },
      ),

    stopNginx: (): Promise<{ message: string }> =>
      fetchApi<{ message: string }>(`${API_BASE}/migration/proxy/stop-nginx`, {
        method: "POST",
      }),

    getTraefikConfig: (siteIndex: number): Promise<TraefikPreview> =>
      fetchApi<TraefikPreview>(
        `${API_BASE}/migration/sites/${siteIndex}/traefik`,
      ),

    migrateSite: (
      siteIndex: number,
      containerId: string,
    ): Promise<MigrateResult> =>
      fetchApi<MigrateResult>(
        `${API_BASE}/migration/sites/${siteIndex}/migrate`,
        {
          method: "POST",
          body: JSON.stringify({ containerId }),
        },
      ),
  },

  containers: {
    list: (all = true): Promise<readonly Container[]> =>
      fetchApiList<Container>(`${API_BASE}/containers?all=${all}`),

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

    remove: async (id: string, force = false): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/containers/${id}?force=${force}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    logs: (id: string, tail = 100): Promise<ContainerLogs> =>
      fetchApi<ContainerLogs>(`${API_BASE}/containers/${id}/logs?tail=${tail}`),
  },

  templates: {
    list: (category?: string): Promise<readonly Template[]> => {
      const url = category
        ? `${API_BASE}/templates?category=${category}`
        : `${API_BASE}/templates`;
      return fetchApiList<Template>(url);
    },

    get: (id: string): Promise<Template> =>
      fetchApi<Template>(`${API_BASE}/templates/${id}`),

    deploy: (id: string, input: DeployTemplateInput): Promise<Container> =>
      fetchApi<Container>(`${API_BASE}/templates/${id}/deploy`, {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },

  images: {
    list: (): Promise<
      readonly {
        id: string;
        repository: string;
        tag: string;
        size: number;
        created: string;
        containers: number;
        dangling: boolean;
        labels: readonly string[];
      }[]
    > => fetchApiList(`${API_BASE}/images`),

    listDangling: (): Promise<
      readonly {
        id: string;
        repository: string;
        tag: string;
        size: number;
        created: string;
        containers: number;
        dangling: boolean;
        labels: readonly string[];
      }[]
    > => fetchApiList(`${API_BASE}/images/dangling`),

    remove: async (id: string, force = false): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/images/${encodeURIComponent(id)}?force=${force}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    prune: (): Promise<{ imagesDeleted: number; spaceReclaimed: number }> =>
      fetchApi<{ imagesDeleted: number; spaceReclaimed: number }>(
        `${API_BASE}/images/prune`,
        { method: "POST" },
      ),
  },

  networks: {
    list: (): Promise<
      readonly {
        name: string;
        id: string;
        driver: string;
        scope: string;
        internal: boolean;
        containers: readonly string[];
      }[]
    > => fetchApiList(`${API_BASE}/networks`),

    create: (name: string): Promise<{ name: string; id: string }> =>
      fetchApi<{ name: string; id: string }>(`${API_BASE}/networks`, {
        method: "POST",
        body: JSON.stringify({ name }),
      }),

    remove: async (name: string): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/networks/${encodeURIComponent(name)}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },

    connectContainer: (
      containerId: string,
      network: string,
    ): Promise<{ message: string }> =>
      fetchApi<{ message: string }>(
        `${API_BASE}/containers/${containerId}/networks`,
        {
          method: "POST",
          body: JSON.stringify({ network }),
        },
      ),

    disconnectContainer: async (
      containerId: string,
      network: string,
    ): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/containers/${containerId}/networks/${encodeURIComponent(network)}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },

  volumes: {
    list: (): Promise<
      readonly {
        name: string;
        driver: string;
        mountpoint: string;
        createdAt: string;
        labels: Record<string, string>;
      }[]
    > => fetchApiList(`${API_BASE}/volumes`),

    create: (name: string): Promise<{ name: string }> =>
      fetchApi<{ name: string }>(`${API_BASE}/volumes`, {
        method: "POST",
        body: JSON.stringify({ name }),
      }),

    remove: async (name: string): Promise<void> => {
      const response = await fetch(
        `${API_BASE}/volumes/${encodeURIComponent(name)}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },
};
