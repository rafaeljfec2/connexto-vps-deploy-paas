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
  readonly runtime: string | null;
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
  readonly type: "RUNNING" | "SUCCESS" | "FAILED" | "LOG" | "HEALTH" | "STATS";
  readonly deployId?: string;
  readonly appId: string;
  readonly message?: string;
  readonly health?: HealthStatus;
  readonly stats?: ContainerStats;
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

export interface CloudflareStatus {
  readonly connected: boolean;
  readonly email?: string;
  readonly accountId?: string;
}

export interface CustomDomain {
  readonly id: string;
  readonly appId: string;
  readonly domain: string;
  readonly recordType: string;
  readonly status: string;
  readonly createdAt: string;
}

export interface ProxyStatus {
  readonly type: string;
  readonly running: boolean;
  readonly version?: string;
  readonly pid?: number;
}

export interface ListenDirective {
  readonly port: number;
  readonly ssl: boolean;
  readonly http2: boolean;
  readonly defaultServer: boolean;
  readonly address?: string;
}

export interface SSEConfig {
  readonly bufferingOff: boolean;
  readonly cacheOff: boolean;
  readonly readTimeout?: string;
  readonly sendTimeout?: string;
  readonly chunkedEncoding?: string;
  readonly xAccelBuffering?: string;
}

export interface NginxLocation {
  readonly path: string;
  readonly isRegex: boolean;
  readonly proxyPass?: string;
  readonly proxyPort?: number;
  readonly root?: string;
  readonly tryFiles?: string;
  readonly headers?: Record<string, string>;
  readonly proxyHeaders?: Record<string, string>;
  readonly hasWebSocket: boolean;
  readonly hasSSE: boolean;
  readonly sseConfig?: SSEConfig;
  readonly proxyBuffering?: string;
  readonly proxyCache?: string;
  readonly readTimeout?: string;
  readonly sendTimeout?: string;
  readonly connectTimeout?: string;
}

export interface NginxSite {
  readonly configFile: string;
  readonly serverNames: readonly string[];
  readonly listen: readonly ListenDirective[];
  readonly root?: string;
  readonly locations: readonly NginxLocation[];
  readonly sslEnabled: boolean;
  readonly sslCertPath?: string;
  readonly sslKeyPath?: string;
  readonly sslProvider?: string;
  readonly headers?: Record<string, string>;
  readonly hasWebSocket: boolean;
  readonly hasSSE: boolean;
  readonly rawConfig: string;
}

export interface SSLCertificate {
  readonly domain: string;
  readonly provider: string;
  readonly certPath: string;
  readonly keyPath: string;
  readonly chainPath?: string;
  readonly fullChainPath?: string;
  readonly expiresAt: string;
  readonly daysUntilExpiry: number;
  readonly isExpired: boolean;
  readonly autoRenew: boolean;
  readonly renewalConfig?: string;
  readonly issuer?: string;
  readonly subject?: string;
}

export interface MigrationContainer {
  readonly id: string;
  readonly name: string;
  readonly image: string;
  readonly status: string;
  readonly state: string;
  readonly ports: readonly string[];
  readonly created: string;
  readonly uptime: string;
  readonly networks?: readonly string[];
}

export interface MigrationStatus {
  readonly proxy: ProxyStatus;
  readonly nginxSites: readonly NginxSite[];
  readonly sslCertificates: readonly SSLCertificate[];
  readonly containers: readonly MigrationContainer[];
  readonly traefikReady: boolean;
  readonly migrationNeeded: boolean;
  readonly warnings: readonly string[];
  readonly lastBackupPath?: string;
  readonly lastBackupTime?: string;
}

export interface BackupResult {
  readonly path: string;
  readonly createdAt: string;
  readonly files: readonly string[];
  readonly size: number;
}

export interface TraefikConfig {
  readonly serviceName: string;
  readonly domain: string;
  readonly port: number;
  readonly pathPrefix?: string;
  readonly priority?: number;
  readonly labels: Record<string, string>;
  readonly hasSSE: boolean;
  readonly hasWebSocket: boolean;
  readonly middlewares?: readonly string[];
}

export interface TraefikPreview {
  readonly site: readonly string[];
  readonly configs: readonly TraefikConfig[];
  readonly yaml: string;
}

export interface MigrateResult {
  readonly containerId: string;
  readonly containerName: string;
  readonly domain: string;
  readonly labels: readonly string[];
  readonly success: boolean;
  readonly message: string;
}
