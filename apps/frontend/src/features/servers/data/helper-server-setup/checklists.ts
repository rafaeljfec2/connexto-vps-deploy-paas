export const CHECKLIST_REMOTE: ReadonlyArray<string> = [
  "User created or existing",
  "sudo loginctl enable-linger <user> executed",
  "SSH working (key or password)",
  "systemctl --user status works without error",
  "Passwordless sudo configured (if not root)",
  "Firewall allows ports: 22, 80, 443, 50052",
];

export const CHECKLIST_BACKEND: ReadonlyArray<string> = [
  "TOKEN_ENCRYPTION_KEY set",
  "AGENT_BINARY_PATH points to the agent binary",
  "GRPC_ENABLED=true and GRPC_SERVER_ADDR set (host:50051 reachable by the agent)",
  "Port 50051 published and firewall allows 50051/tcp",
  "Agent binary built (go build or Docker image)",
];

export const CHECKLIST_PANEL: ReadonlyArray<string> = [
  "Server added (Name, Host, Port, User, Key or Password)",
  "ACME Email filled (for automatic TLS via Traefik)",
  "Provision clicked",
  "Status changed to Online",
];
