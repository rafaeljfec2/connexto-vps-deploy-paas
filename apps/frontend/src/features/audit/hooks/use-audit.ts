import type { UseQueryOptions } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";

const API_URL = import.meta.env.VITE_API_URL ?? "";
const API_BASE = `${API_URL}/paas-deploy/v1`;

export interface AuditLog {
  readonly id: string;
  readonly eventType: string;
  readonly resourceType: string;
  readonly resourceId?: string;
  readonly resourceName?: string;
  readonly userId?: string;
  readonly userName?: string;
  readonly details?: Record<string, unknown>;
  readonly ipAddress?: string;
  readonly createdAt: string;
}

export interface AuditLogsResponse {
  readonly logs: readonly AuditLog[];
  readonly total: number;
  readonly limit: number;
  readonly offset: number;
}

export interface AuditFilter {
  eventType?: string;
  resourceType?: string;
  resourceId?: string;
  userId?: string;
  startDate?: string;
  endDate?: string;
  limit?: number;
  offset?: number;
}

async function fetchAuditLogs(filter: AuditFilter): Promise<AuditLogsResponse> {
  const params = new URLSearchParams();
  if (filter.eventType) params.set("eventType", filter.eventType);
  if (filter.resourceType) params.set("resourceType", filter.resourceType);
  if (filter.resourceId) params.set("resourceId", filter.resourceId);
  if (filter.userId) params.set("userId", filter.userId);
  if (filter.startDate) params.set("startDate", filter.startDate);
  if (filter.endDate) params.set("endDate", filter.endDate);
  if (filter.limit) params.set("limit", filter.limit.toString());
  if (filter.offset) params.set("offset", filter.offset.toString());

  const response = await fetch(`${API_BASE}/audit/logs?${params.toString()}`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to fetch audit logs");
  }

  const data = await response.json();
  return data.data;
}

export function useAuditLogs(
  filter: AuditFilter = {},
  options?: Omit<
    UseQueryOptions<AuditLogsResponse, Error>,
    "queryKey" | "queryFn"
  >,
) {
  return useQuery({
    queryKey: ["audit-logs", filter],
    queryFn: () => fetchAuditLogs(filter),
    refetchInterval: 30000,
    ...options,
  });
}

export interface WebhookPayload {
  readonly id: string;
  readonly deliveryId: string;
  readonly eventType: string;
  readonly provider: string;
  readonly outcome: string;
  readonly errorMessage?: string;
  readonly createdAt: string;
}

export interface WebhookPayloadsResponse {
  readonly payloads: readonly WebhookPayload[];
  readonly total: number;
  readonly limit: number;
  readonly offset: number;
}

export interface WebhookPayloadsFilter {
  readonly limit?: number;
  readonly offset?: number;
}

async function fetchWebhookPayloads(
  filter: WebhookPayloadsFilter = {},
): Promise<WebhookPayloadsResponse> {
  const params = new URLSearchParams();
  if (filter.limit != null) params.set("limit", filter.limit.toString());
  if (filter.offset != null) params.set("offset", filter.offset.toString());

  const response = await fetch(
    `${API_BASE}/audit/webhook-payloads?${params.toString()}`,
    { credentials: "include" },
  );

  if (!response.ok) {
    throw new Error("Failed to fetch webhook payloads");
  }

  const data = await response.json();
  return data.data;
}

export function useWebhookPayloads(
  filter: WebhookPayloadsFilter = {},
  options?: Omit<
    UseQueryOptions<WebhookPayloadsResponse, Error>,
    "queryKey" | "queryFn"
  >,
) {
  return useQuery({
    queryKey: ["webhook-payloads", filter],
    queryFn: () => fetchWebhookPayloads(filter),
    refetchInterval: 30000,
    ...options,
  });
}
