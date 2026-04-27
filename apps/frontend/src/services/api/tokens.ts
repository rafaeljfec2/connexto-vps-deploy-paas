import { API_BASE, fetchApi, fetchApiDelete } from "./client";

export type TokenScope =
  | "read"
  | "deploy"
  | "containers:write"
  | "config:write"
  | "resources:write"
  | "servers:write"
  | "destructive"
  | "admin";

export const TOKEN_SCOPES: readonly TokenScope[] = [
  "read",
  "deploy",
  "containers:write",
  "config:write",
  "resources:write",
  "servers:write",
  "destructive",
  "admin",
] as const;

export interface PersonalAccessToken {
  readonly id: string;
  readonly name: string;
  readonly tokenPrefix: string;
  readonly scopes: readonly TokenScope[];
  readonly lastUsedAt?: string;
  readonly expiresAt?: string;
  readonly revokedAt?: string;
  readonly createdAt: string;
}

export interface ListTokensResponse {
  readonly tokens: readonly PersonalAccessToken[];
}

export interface CreateTokenPayload {
  readonly name: string;
  readonly scopes: readonly TokenScope[];
  readonly expiresAt?: string;
}

export interface CreateTokenResponse {
  readonly token: PersonalAccessToken;
  readonly plaintextToken: string;
}

const TOKENS_URL = `${API_BASE}/tokens`;

export function listTokens(): Promise<ListTokensResponse> {
  return fetchApi<ListTokensResponse>(TOKENS_URL);
}

export function createToken(
  payload: CreateTokenPayload,
): Promise<CreateTokenResponse> {
  return fetchApi<CreateTokenResponse>(TOKENS_URL, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function revokeToken(tokenId: string): Promise<void> {
  return fetchApiDelete(`${TOKENS_URL}/${tokenId}`);
}
