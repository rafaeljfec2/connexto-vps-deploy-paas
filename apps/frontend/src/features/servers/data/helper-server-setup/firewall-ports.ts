export interface FirewallPort {
  readonly port: string;
  readonly protocol: string;
  readonly purpose: string;
  readonly required: string;
  readonly command: string;
}

export const FIREWALL_PORTS: ReadonlyArray<FirewallPort> = [
  {
    port: "22",
    protocol: "TCP",
    purpose: "SSH (provisioning)",
    required: "Yes",
    command: "sudo ufw allow 22/tcp",
  },
  {
    port: "80",
    protocol: "TCP",
    purpose: "HTTP (Traefik)",
    required: "Yes",
    command: "sudo ufw allow 80/tcp",
  },
  {
    port: "443",
    protocol: "TCP",
    purpose: "HTTPS (Traefik + TLS)",
    required: "Yes",
    command: "sudo ufw allow 443/tcp",
  },
  {
    port: "50051",
    protocol: "TCP",
    purpose: "gRPC (Traefik to backend)",
    required: "If gRPC via Traefik",
    command: "sudo ufw allow 50051/tcp",
  },
  {
    port: "50052",
    protocol: "TCP",
    purpose: "gRPC (agent health check)",
    required: "Yes",
    command: "sudo ufw allow 50052/tcp",
  },
  {
    port: "8081",
    protocol: "TCP",
    purpose: "Traefik Dashboard",
    required: "Optional",
    command: "sudo ufw allow 8081/tcp",
  },
];
