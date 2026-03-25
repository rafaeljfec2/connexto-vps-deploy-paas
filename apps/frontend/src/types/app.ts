import type { DeploymentSummary } from "./deployment";

export type AppStatus = "active" | "inactive" | "deleted";

export interface App {
  readonly id: string;
  readonly name: string;
  readonly repositoryUrl: string;
  readonly branch: string;
  readonly workdir: string;
  readonly runtime: string | null;
  readonly config: Record<string, unknown>;
  readonly status: AppStatus;
  readonly webhookId: number | null;
  readonly appVersion?: string;
  readonly serverId?: string;
  readonly lastDeployedAt: string | null;
  readonly lastDeployment?: DeploymentSummary | null;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface CreateAppInput {
  readonly name: string;
  readonly repositoryUrl: string;
  readonly branch?: string;
  readonly workdir?: string;
  readonly serverId?: string;
}

export interface UpdateAppInput {
  readonly name?: string;
  readonly branch?: string;
  readonly workdir?: string;
}

export interface AppURL {
  readonly url: string;
  readonly port: number;
  readonly hostPort: number;
}

export interface AppVolumeConfig {
  readonly name?: string;
  readonly source?: string;
  readonly target: string;
  readonly readOnly?: boolean;
}

export interface AppConfig {
  readonly name: string;
  readonly port: number;
  readonly hostPort: number;
  readonly healthcheck: {
    readonly path: string;
    readonly interval: string;
    readonly timeout: string;
    readonly retries: number;
    readonly startPeriod: string;
  };
  readonly resources: {
    readonly memory: string;
    readonly cpu: string;
  };
  readonly domains: readonly string[];
  readonly volumes: readonly AppVolumeConfig[];
}

export interface EnvVar {
  readonly id: string;
  readonly appId: string;
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface CreateEnvVarInput {
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
}

export interface BulkEnvVarInput {
  readonly vars: readonly CreateEnvVarInput[];
}

export interface CommitInfo {
  readonly sha: string;
  readonly message: string;
  readonly author: string;
  readonly date: string;
  readonly url: string;
}

export interface WebhookSetupResult {
  readonly webhookId: number;
  readonly provider: string;
  readonly active: boolean;
}

export interface WebhookStatus {
  readonly exists: boolean;
  readonly active: boolean;
  readonly lastPingAt: string | null;
  readonly error: string | null;
  readonly configuredUrl?: string | null;
}

export type CertificateStatusType =
  | "active"
  | "pending"
  | "no_tls"
  | "unknown"
  | "error";

export interface CertificateStatus {
  readonly domain: string;
  readonly status: CertificateStatusType;
  readonly expiresAt?: string | null;
  readonly issuedAt?: string | null;
  readonly issuer?: string | null;
  readonly error?: string | null;
}

export interface CloudflareStatus {
  readonly connected: boolean;
  readonly email?: string;
  readonly accountId?: string;
}

export interface CustomDomain {
  readonly id: string;
  readonly appId: string;
  readonly domain: string;
  readonly pathPrefix: string;
  readonly recordType: string;
  readonly status: string;
  readonly createdAt: string;
}

export type NotificationChannelType = "slack" | "discord" | "email";

export interface NotificationChannel {
  readonly id: string;
  readonly type: NotificationChannelType;
  readonly name: string;
  readonly config: Record<string, unknown>;
  readonly appId?: string;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface NotificationRule {
  readonly id: string;
  readonly eventType: string;
  readonly channelId: string;
  readonly appId?: string;
  readonly enabled: boolean;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export type NotificationEventType =
  | "deploy_running"
  | "deploy_success"
  | "deploy_failed"
  | "container_down"
  | "health_unhealthy";

export interface CreateNotificationChannelInput {
  readonly type: NotificationChannelType;
  readonly name: string;
  readonly config: Record<string, unknown>;
  readonly appId?: string;
}

export interface CreateNotificationRuleInput {
  readonly eventType: NotificationEventType;
  readonly channelId: string;
  readonly appId?: string;
  readonly enabled: boolean;
}
