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

export interface SSEEvent {
  readonly type: "RUNNING" | "SUCCESS" | "FAILED" | "LOG";
  readonly deployId: string;
  readonly appId: string;
  readonly message?: string;
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
