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

export interface ContainerActionResult {
  readonly success: boolean;
  readonly message: string;
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

export interface DockerNetwork {
  readonly name: string;
  readonly id: string;
  readonly driver: string;
  readonly scope: string;
  readonly internal: boolean;
  readonly containers: readonly string[];
}

export interface DockerVolume {
  readonly name: string;
  readonly driver: string;
  readonly mountpoint: string;
  readonly createdAt: string;
  readonly labels: Record<string, string>;
}
