export * from "./api";

export type AppStatus = "active" | "inactive" | "deleted";

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
  readonly startedAt?: string | null;
  readonly finishedAt?: string | null;
  readonly durationMs?: number | null;
  readonly logs?: string | null;
}

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
  readonly lastDeployment?: DeploymentSummary | null;
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
  readonly pathPrefix: string;
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

export type ContainerState =
  | "running"
  | "exited"
  | "paused"
  | "restarting"
  | "dead"
  | "created";

export interface ContainerPort {
  readonly privatePort: number;
  readonly publicPort?: number;
  readonly type: "tcp" | "udp";
}

export interface ContainerMount {
  readonly type: string;
  readonly source: string;
  readonly destination: string;
  readonly readOnly: boolean;
}

export interface Container {
  readonly id: string;
  readonly name: string;
  readonly image: string;
  readonly state: ContainerState;
  readonly status: string;
  readonly health: string;
  readonly created: string;
  readonly ipAddress: string;
  readonly ports: readonly ContainerPort[];
  readonly labels: Record<string, string>;
  readonly networks: readonly string[];
  readonly mounts: readonly ContainerMount[];
  readonly isFlowDeployManaged: boolean;
}

export interface DockerImage {
  readonly id: string;
  readonly repository: string;
  readonly tag: string;
  readonly size: number;
  readonly created: string;
  readonly containers: number;
  readonly dangling: boolean;
  readonly labels: readonly string[];
}

export interface PruneResult {
  readonly imagesDeleted: number;
  readonly spaceReclaimed: number;
}

export interface PortMappingInput {
  readonly hostPort: number;
  readonly containerPort: number;
  readonly protocol?: "tcp" | "udp";
}

export interface VolumeMappingInput {
  readonly hostPath: string;
  readonly containerPath: string;
  readonly readOnly?: boolean;
}

export interface CreateContainerInput {
  readonly name: string;
  readonly image: string;
  readonly ports?: readonly PortMappingInput[];
  readonly env?: Record<string, string>;
  readonly volumes?: readonly VolumeMappingInput[];
  readonly network?: string;
  readonly restartPolicy?: "no" | "always" | "unless-stopped" | "on-failure";
  readonly command?: readonly string[];
}

export interface TemplateEnvVar {
  readonly name: string;
  readonly label: string;
  readonly description?: string;
  readonly default?: string;
  readonly required: boolean;
}

export interface Template {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly image: string;
  readonly category: string;
  readonly type: "container" | "stack";
  readonly logo?: string;
  readonly env?: readonly TemplateEnvVar[];
  readonly ports?: readonly number[];
  readonly volumes?: readonly string[];
}

export interface DeployTemplateInput {
  readonly name?: string;
  readonly env?: Record<string, string>;
  readonly ports?: readonly PortMappingInput[];
  readonly network?: string;
  readonly restartPolicy?: string;
}
