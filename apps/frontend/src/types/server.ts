import type { ContainerStats } from "./docker";

export interface HealthStatus {
  readonly status: "running" | "exited" | "paused" | "restarting" | "not_found";
  readonly health: "healthy" | "unhealthy" | "starting" | "none";
  readonly startedAt?: string;
  readonly uptime?: string;
}

export type SSEEventType =
  | "RUNNING"
  | "SUCCESS"
  | "FAILED"
  | "LOG"
  | "HEALTH"
  | "STATS"
  | "PROVISION_STEP"
  | "PROVISION_LOG"
  | "PROVISION_COMPLETED"
  | "PROVISION_FAILED"
  | "AGENT_UPDATE_STEP";

export type AgentUpdateStep = "enqueued" | "delivered" | "updated" | "error";

export interface AgentUpdateState {
  readonly step: AgentUpdateStep;
  readonly status: "running" | "completed" | "error";
  readonly version?: string;
  readonly errorMessage?: string;
  readonly startedAt: number;
}

export interface SSEEvent {
  readonly type: SSEEventType;
  readonly deployId?: string;
  readonly appId?: string;
  readonly serverId?: string;
  readonly step?: string;
  readonly status?: string;
  readonly message?: string;
  readonly health?: HealthStatus;
  readonly stats?: ContainerStats;
  readonly timestamp: string;
}

export interface ProvisionStepState {
  readonly step: string;
  readonly status: string;
  readonly message: string;
}

export interface ProvisionProgressState {
  readonly steps: readonly ProvisionStepState[];
  readonly logs: readonly string[];
  readonly status: "running" | "completed" | "failed";
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

export type ServerStatus =
  | "pending"
  | "provisioning"
  | "online"
  | "offline"
  | "error";

export type AgentUpdateMode = "grpc" | "https";

export interface Server {
  readonly id: string;
  readonly name: string;
  readonly host: string;
  readonly sshPort: number;
  readonly sshUser: string;
  readonly acmeEmail?: string;
  readonly status: ServerStatus;
  readonly agentVersion?: string;
  readonly agentUpdateMode: AgentUpdateMode;
  readonly latestAgentVersion: string;
  readonly lastHeartbeatAt?: string;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface CreateServerInput {
  readonly name: string;
  readonly host: string;
  readonly sshPort?: number;
  readonly sshUser: string;
  readonly sshKey?: string;
  readonly sshPassword?: string;
  readonly acmeEmail?: string;
}

export interface ServerSystemInfo {
  readonly hostname?: string;
  readonly os?: string;
  readonly os_version?: string;
  readonly architecture?: string;
  readonly cpu_cores?: number;
  readonly memory_total_bytes?: number;
  readonly disk_total_bytes?: number;
  readonly kernel_version?: string;
}

export interface ServerSystemMetrics {
  readonly cpu_usage_percent?: number;
  readonly memory_used_bytes?: number;
  readonly memory_available_bytes?: number;
  readonly disk_used_bytes?: number;
  readonly disk_available_bytes?: number;
  readonly load_average_1m?: number;
  readonly load_average_5m?: number;
  readonly load_average_15m?: number;
  readonly network_rx_bytes?: number;
  readonly network_tx_bytes?: number;
}

export interface ServerStats {
  readonly systemInfo: ServerSystemInfo;
  readonly systemMetrics: ServerSystemMetrics;
}
