export type DeployStatus =
  | "pending"
  | "running"
  | "success"
  | "failed"
  | "cancelled";

export interface DeploymentSummary {
  readonly id: string;
  readonly status: DeployStatus;
  readonly commitSha: string;
  readonly commitMessage?: string;
  readonly appVersion?: string;
  readonly startedAt?: string | null;
  readonly finishedAt?: string | null;
  readonly durationMs?: number | null;
  readonly logs?: string | null;
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
  readonly appVersion?: string;
  readonly durationMs?: number | null;
  readonly createdAt: string;
}
