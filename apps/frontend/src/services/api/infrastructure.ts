import type {
  BackupResult,
  CertificateStatus,
  CloudflareStatus,
  Container,
  DeployTemplateInput,
  MigrateResult,
  MigrationStatus,
  Template,
  TraefikPreview,
} from "@/types";
import { API_BASE, API_URL, fetchApi, fetchApiList } from "./client";

export const cloudflareApi = {
  status: (): Promise<CloudflareStatus> =>
    fetchApi<CloudflareStatus>(`${API_URL}/auth/cloudflare/status`),

  connect: (apiToken: string): Promise<CloudflareStatus> =>
    fetchApi<CloudflareStatus>(`${API_URL}/auth/cloudflare/connect`, {
      method: "POST",
      body: JSON.stringify({ apiToken }),
    }),

  disconnect: async (): Promise<void> => {
    await fetchApi<void>(`${API_URL}/auth/cloudflare/disconnect`, {
      method: "POST",
    });
  },
};

export const certificatesApi = {
  list: (): Promise<readonly CertificateStatus[]> =>
    fetchApiList<CertificateStatus>(`${API_URL}/api/certificates`),

  getStatus: (domain: string): Promise<CertificateStatus> =>
    fetchApi<CertificateStatus>(
      `${API_URL}/api/certificates/${encodeURIComponent(domain)}`,
    ),
};

export const migrationApi = {
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
      { method: "POST", body: JSON.stringify({ containerIds }) },
    ),

  startContainers: (
    containerIds: readonly string[],
  ): Promise<{ message: string; started: readonly string[] }> =>
    fetchApi<{ message: string; started: readonly string[] }>(
      `${API_BASE}/migration/containers/start`,
      { method: "POST", body: JSON.stringify({ containerIds }) },
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
      { method: "POST", body: JSON.stringify({ containerId }) },
    ),
};

export const templatesApi = {
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
};
