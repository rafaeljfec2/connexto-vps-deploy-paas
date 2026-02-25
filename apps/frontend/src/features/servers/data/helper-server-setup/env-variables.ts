export interface EnvVariable {
  readonly name: string;
  readonly example: string;
  readonly required: boolean;
}

export const ENV_VARIABLES: ReadonlyArray<EnvVariable> = [
  {
    name: "TOKEN_ENCRYPTION_KEY",
    example: "(output of openssl rand -base64 32)",
    required: true,
  },
  { name: "GRPC_ENABLED", example: "true", required: true },
  { name: "GRPC_PORT", example: "50051", required: true },
  {
    name: "GRPC_SERVER_ADDR",
    example: "host:50051 (reachable from VPS)",
    required: true,
  },
  {
    name: "AGENT_BINARY_PATH",
    example: "/app/agent or path to binary",
    required: true,
  },
  { name: "AGENT_GRPC_PORT", example: "50052", required: false },
];
