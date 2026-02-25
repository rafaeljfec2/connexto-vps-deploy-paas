export interface TroubleshootItem {
  readonly problem: string;
  readonly cause: string;
  readonly solution: string;
}

export const TROUBLE_SSH: ReadonlyArray<TroubleshootItem> = [
  {
    problem: "ssh connect: i/o timeout",
    cause: "Firewall blocking SSH port.",
    solution:
      "Allow the SSH port: sudo ufw allow 22/tcp && sudo ufw reload. Test: nc -vz HOST PORT.",
  },
  {
    problem: "unable to authenticate (SSH)",
    cause: "Wrong key or password.",
    solution:
      "Check credentials in the panel. Test manually: ssh -p PORT USER@HOST.",
  },
  {
    problem: "provision failed: EOF",
    cause: "Connection dropped during file transfer.",
    solution: "Re-provision the server. Check network stability.",
  },
];

export const TROUBLE_DOCKER: ReadonlyArray<TroubleshootItem> = [
  {
    problem: "install docker: command timed out after 10m",
    cause: "Slow internet or no access to get.docker.com.",
    solution:
      "Check internet access on the remote server. Install Docker manually if needed, then re-provision.",
  },
  {
    problem: "start docker daemon: ... sudo -n",
    cause: "Passwordless sudo not configured.",
    solution:
      "Configure sudoers (see step 1.3). Ensure the user can run: sudo -n systemctl start docker.",
  },
  {
    problem: "create docker network: permission denied",
    cause: "User not in the docker group.",
    solution:
      "Run: sudo usermod -aG docker USER, then disconnect and reconnect SSH.",
  },
];

export const TROUBLE_TRAEFIK: ReadonlyArray<TroubleshootItem> = [
  {
    problem: "Traefik not installed",
    cause: "ACME Email not provided.",
    solution: "Edit the server and fill in the ACME Email field. Re-provision.",
  },
  {
    problem: "write traefik config: ... sudo -n",
    cause: "Passwordless sudo not configured for /opt/traefik.",
    solution: "Configure sudoers (see step 1.3).",
  },
  {
    problem: "Port 80/443 already in use",
    cause: "Another service (nginx, apache) using those ports.",
    solution:
      "Stop the conflicting service: sudo systemctl stop nginx. Then re-provision.",
  },
  {
    problem: "TLS not working",
    cause: "DNS not pointing to the server.",
    solution: "Configure a DNS A record pointing your domain to the server IP.",
  },
];

export const TROUBLE_AGENT: ReadonlyArray<TroubleshootItem> = [
  {
    problem: "Unit ... could not be found / Failed to connect to bus",
    cause: "Linger not enabled for the user.",
    solution:
      "Run: sudo loginctl enable-linger USER. Then: systemctl --user status.",
  },
  {
    problem: "Status stays Provisioning or Pending",
    cause: "SSH stuck, process hung, or agent not connecting back.",
    solution:
      "Check backend logs. On remote: systemctl --user status paasdeploy-agent.",
  },
  {
    problem: "Agent not connecting (dial tcp ... 50051: i/o timeout)",
    cause: "Port 50051 not open on the backend host.",
    solution:
      "Ensure port 50051 is published. Open firewall: sudo ufw allow 50051/tcp. Test: nc -vz BACKEND_IP 50051.",
  },
  {
    problem: "no such host (DNS resolution failure)",
    cause: "DNS on the agent server cannot resolve the backend hostname.",
    solution: "Use the backend IP in GRPC_SERVER_ADDR or fix DNS.",
  },
];

export const TROUBLE_MULTITENANCY: ReadonlyArray<TroubleshootItem> = [
  {
    problem: "Server not showing in the list",
    cause: "Server belongs to another user.",
    solution:
      "Each user only sees their own servers. Verify you are logged in with the correct account.",
  },
  {
    problem: "server not found when accessing",
    cause: "Trying to access another user's server.",
    solution: "Only the owner can access a server. Check the server ID.",
  },
];
