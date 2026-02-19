import type { ApiEnvelope } from "@/types";
import { ApiError, isApiError } from "@/types";

export const API_URL = import.meta.env.VITE_API_URL ?? "";
export const API_BASE = `${API_URL}/paas-deploy/v1`;

export async function fetchApi<T>(
  url: string,
  options?: RequestInit,
): Promise<T> {
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

export async function fetchApiList<T>(
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

export async function fetchApiDelete(url: string): Promise<void> {
  const response = await fetch(url, {
    method: "DELETE",
    credentials: "include",
  });

  if (!response.ok && response.status !== 204) {
    const envelope: ApiEnvelope<null> = await response.json();
    throw ApiError.fromResponse(envelope, response.status);
  }
}

export function buildUrl(
  base: string,
  params?: Record<string, string | boolean | number | undefined>,
): string {
  if (!params) return base;

  const entries = Object.entries(params).filter(
    ([, v]) => v !== undefined && v !== "",
  );
  if (entries.length === 0) return base;

  const separator = base.includes("?") ? "&" : "?";
  const query = entries
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
    .join("&");

  return `${base}${separator}${query}`;
}
