export * from "./api";

export type AppStatus = "active" | "inactive" | "deleted";

export type DeployStatus =
  | "pending"
  | "running"
  | "success"
  | "failed"
  | "cancelled";

export interface App {
  readonly id: string;
  readonly name: string;
  readonly repositoryUrl: string;
  readonly branch: string;
  readonly workdir: string;
  readonly config: Record<string, unknown>;
  readonly status: AppStatus;
  readonly webhookId: number | null;
  readonly lastDeployedAt: string | null;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface Deployment {
  readonly id: string;
  readonly appId: string;
  readonly commitSha: string;
  readonly commitMessage: string;
  readonly status: DeployStatus;
  readonly startedAt: string | null;
  readonly finishedAt: string | null;
  readonly errorMessage: string | null;
  readonly logs: string | null;
  readonly previousImageTag: string | null;
  readonly currentImageTag: string | null;
  readonly createdAt: string;
}

export interface CreateAppInput {
  readonly name: string;
  readonly repositoryUrl: string;
  readonly branch?: string;
  readonly workdir?: string;
}

export interface HealthStatus {
  readonly status: "running" | "exited" | "paused" | "restarting" | "not_found";
  readonly health: "healthy" | "unhealthy" | "starting" | "none";
  readonly startedAt?: string;
  readonly uptime?: string;
}

export interface SSEEvent {
  readonly type: "RUNNING" | "SUCCESS" | "FAILED" | "LOG" | "HEALTH";
  readonly deployId?: string;
  readonly appId: string;
  readonly message?: string;
  readonly health?: HealthStatus;
  readonly timestamp: string;
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

export interface AppURL {
  readonly url: string;
  readonly port: number;
  readonly hostPort: number;
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
}

export interface ContainerActionResult {
  readonly success: boolean;
  readonly message: string;
}

export interface UpdateAppInput {
  readonly branch?: string;
  readonly workdir?: string;
}

export interface ContainerLogs {
  readonly logs: string;
}

export interface ContainerStats {
  readonly cpuPercent: number;
  readonly memoryUsage: number;
  readonly memoryLimit: number;
  readonly memoryPercent: number;
  readonly networkRx: number;
  readonly networkTx: number;
  readonly pids: number;
}

export interface CommitInfo {
  readonly sha: string;
  readonly message: string;
  readonly author: string;
  readonly date: string;
  readonly url: string;
}
