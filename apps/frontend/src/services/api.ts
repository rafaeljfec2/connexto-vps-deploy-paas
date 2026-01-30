import type {
  ApiEnvelope,
  App,
  BulkEnvVarInput,
  CreateAppInput,
  CreateEnvVarInput,
  Deployment,
  EnvVar,
  WebhookSetupResult,
  WebhookStatus,
} from "@/types";
import { ApiError, isApiError } from "@/types";

const API_BASE = "/paas-deploy/v1";

async function fetchApi<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
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
  apps: {
    list: (): Promise<readonly App[]> => fetchApiList<App>(`${API_BASE}/apps`),

    get: (id: string): Promise<App> => fetchApi<App>(`${API_BASE}/apps/${id}`),

    create: (input: CreateAppInput): Promise<App> =>
      fetchApi<App>(`${API_BASE}/apps`, {
        method: "POST",
        body: JSON.stringify(input),
      }),

    delete: async (id: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${id}`, {
        method: "DELETE",
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
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

  webhooks: {
    setup: (appId: string): Promise<WebhookSetupResult> =>
      fetchApi<WebhookSetupResult>(`${API_BASE}/apps/${appId}/webhook`, {
        method: "POST",
      }),

    remove: async (appId: string): Promise<void> => {
      const response = await fetch(`${API_BASE}/apps/${appId}/webhook`, {
        method: "DELETE",
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
      });

      if (!response.ok && response.status !== 204) {
        const envelope: ApiEnvelope<null> = await response.json();
        throw ApiError.fromResponse(envelope, response.status);
      }
    },
  },
};
