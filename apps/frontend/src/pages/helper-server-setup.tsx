import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  Container,
  FolderTree,
  Globe,
  KeyRound,
  Network,
  RefreshCw,
  Server,
  Shield,
  Terminal,
  Workflow,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PageHeader } from "@/components/page-header";

function CodeBlock({ children }: { readonly children: string }) {
  return (
    <pre className="overflow-x-auto rounded-md border bg-muted/50 p-3 text-sm">
      <code>{children}</code>
    </pre>
  );
}

interface FirewallPort {
  readonly port: string;
  readonly protocol: string;
  readonly purpose: string;
  readonly required: string;
  readonly command: string;
}

const FIREWALL_PORTS: ReadonlyArray<FirewallPort> = [
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

interface ProvisionStep {
  readonly id: string;
  readonly label: string;
}

const PROVISION_STEPS: ReadonlyArray<ProvisionStep> = [
  { id: "ssh_connect", label: "Connecting via SSH" },
  { id: "remote_env", label: "Detecting environment (home dir, UID)" },
  {
    id: "docker_check / docker_install",
    label: "Checking / installing Docker",
  },
  { id: "docker_start", label: "Checking / starting Docker daemon" },
  { id: "docker_network", label: "Creating Docker network" },
  {
    id: "traefik_check / traefik_install",
    label: "Checking / installing Traefik",
  },
  { id: "sftp_client", label: "Connecting SFTP" },
  { id: "install_dir", label: "Creating directories" },
  { id: "agent_certs", label: "Generating and installing mTLS certificates" },
  { id: "agent_binary", label: "Copying agent binary" },
  { id: "systemd_unit", label: "Configuring systemd service" },
  { id: "start_agent", label: "Starting agent" },
];

interface EnvVariable {
  readonly name: string;
  readonly example: string;
  readonly required: boolean;
}

const ENV_VARIABLES: ReadonlyArray<EnvVariable> = [
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

interface TroubleshootItem {
  readonly problem: string;
  readonly cause: string;
  readonly solution: string;
}

const TROUBLE_SSH: ReadonlyArray<TroubleshootItem> = [
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

const TROUBLE_DOCKER: ReadonlyArray<TroubleshootItem> = [
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

const TROUBLE_TRAEFIK: ReadonlyArray<TroubleshootItem> = [
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

const TROUBLE_AGENT: ReadonlyArray<TroubleshootItem> = [
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

const TROUBLE_MULTITENANCY: ReadonlyArray<TroubleshootItem> = [
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

function TroubleshootSection({
  items,
}: {
  readonly items: ReadonlyArray<TroubleshootItem>;
}) {
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div
          key={item.problem}
          className="rounded-lg border bg-muted/20 p-3 sm:p-4 space-y-2"
        >
          <p className="font-medium text-sm break-words">{item.problem}</p>
          <p className="text-sm text-muted-foreground">
            <span className="font-medium">Cause:</span> {item.cause}
          </p>
          <p className="text-sm text-muted-foreground">
            <span className="font-medium">Solution:</span> {item.solution}
          </p>
        </div>
      ))}
    </div>
  );
}

const CHECKLIST_REMOTE: ReadonlyArray<string> = [
  "User created or existing",
  "sudo loginctl enable-linger <user> executed",
  "SSH working (key or password)",
  "systemctl --user status works without error",
  "Passwordless sudo configured (if not root)",
  "Firewall allows ports: 22, 80, 443, 50052",
];

const CHECKLIST_BACKEND: ReadonlyArray<string> = [
  "TOKEN_ENCRYPTION_KEY set",
  "AGENT_BINARY_PATH points to the agent binary",
  "GRPC_ENABLED=true and GRPC_SERVER_ADDR set (host:50051 reachable by the agent)",
  "Port 50051 published and firewall allows 50051/tcp",
  "Agent binary built (go build or Docker image)",
];

const CHECKLIST_PANEL: ReadonlyArray<string> = [
  "Server added (Name, Host, Port, User, Key or Password)",
  "ACME Email filled (for automatic TLS via Traefik)",
  "Provision clicked",
  "Status changed to Online",
];

function StepNumber({ n }: { readonly n: number }) {
  return (
    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
      {n}
    </span>
  );
}

export function HelperServerSetupPage() {
  return (
    <div className="space-y-8">
      <PageHeader
        backTo={ROUTES.SERVERS}
        title="Server setup guide"
        description="Step-by-step tutorial to prepare a remote server, provision Docker, Traefik, and the deploy agent"
        icon={Server}
      />

      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Overview</AlertTitle>
        <AlertDescription>
          Prepare the remote VPS, configure the backend, add the server in the
          panel, then provision. The provisioning{" "}
          <strong>automatically installs Docker, Traefik, and the agent</strong>{" "}
          via SSH. The agent connects to the backend via gRPC with mTLS.
        </AlertDescription>
      </Alert>

      <section aria-labelledby="provisioning-heading">
        <Card>
          <CardHeader>
            <CardTitle
              id="provisioning-heading"
              className="flex items-center gap-2"
            >
              <Workflow className="h-5 w-5" />
              How provisioning works
            </CardTitle>
            <CardDescription>
              When you click <strong>Provision</strong>, the backend
              automatically executes these steps via SSH.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="overflow-x-auto rounded-md border bg-muted/30 p-3">
              <ol className="space-y-2 text-sm">
                {PROVISION_STEPS.map((step, i) => (
                  <li key={step.id} className="flex items-start gap-2">
                    <Badge
                      variant="outline"
                      className="mt-0.5 shrink-0 font-mono text-xs"
                    >
                      {i + 1}
                    </Badge>
                    <div>
                      <code className="text-xs text-muted-foreground">
                        {step.id}
                      </code>
                      <span className="mx-1.5 text-muted-foreground">
                        {"\u2014"}
                      </span>
                      <span>{step.label}</span>
                    </div>
                  </li>
                ))}
              </ol>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-md border p-3 space-y-1">
                <p className="text-sm font-medium flex items-center gap-1.5">
                  <Container className="h-4 w-4" /> Automatic installs
                </p>
                <ul className="text-sm text-muted-foreground space-y-0.5">
                  <li>
                    <strong>Docker</strong> {"\u2014"} via get.docker.com (if
                    not installed)
                  </li>
                  <li>
                    <strong>Docker network</strong> {"\u2014"}{" "}
                    &quot;paasdeploy&quot; (if not exists)
                  </li>
                  <li>
                    <strong>Traefik</strong> {"\u2014"} reverse proxy + TLS (if
                    ACME Email provided)
                  </li>
                  <li>
                    <strong>Agent</strong> {"\u2014"} binary + mTLS certs +
                    systemd service
                  </li>
                </ul>
              </div>
              <div className="rounded-md border p-3 space-y-1">
                <p className="text-sm font-medium flex items-center gap-1.5">
                  <RefreshCw className="h-4 w-4" /> Timeouts per step
                </p>
                <ul className="text-sm text-muted-foreground space-y-0.5">
                  <li>Docker install: 10 minutes</li>
                  <li>Docker start: 30 seconds</li>
                  <li>Docker network: 30 seconds</li>
                  <li>Traefik setup: 5 minutes</li>
                </ul>
              </div>
            </div>

            <Alert variant="default">
              <RefreshCw className="h-4 w-4" />
              <AlertTitle>Idempotent</AlertTitle>
              <AlertDescription>
                You can re-provision safely. Docker is skipped if already
                running. The network is skipped if it exists. Traefik is skipped
                if running with the correct version (or upgraded automatically).
                The agent binary and certs are always overwritten.
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="firewall-heading">
        <Card>
          <CardHeader>
            <CardTitle
              id="firewall-heading"
              className="flex items-center gap-2"
            >
              <Shield className="h-5 w-5" />
              Firewall ports
            </CardTitle>
            <CardDescription>
              Open these ports on the correct servers for provisioning and
              communication.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="overflow-x-auto rounded-md border">
              <table className="w-full min-w-[480px] text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-3 py-2 text-left font-medium">Port</th>
                    <th className="px-3 py-2 text-left font-medium">
                      Protocol
                    </th>
                    <th className="px-3 py-2 text-left font-medium">Purpose</th>
                    <th className="px-3 py-2 text-left font-medium">
                      Required
                    </th>
                    <th className="px-3 py-2 text-left font-medium">Command</th>
                  </tr>
                </thead>
                <tbody className="text-muted-foreground">
                  {FIREWALL_PORTS.map((p) => (
                    <tr key={p.port} className="border-b last:border-0">
                      <td className="px-3 py-2 font-medium text-foreground">
                        {p.port}
                      </td>
                      <td className="px-3 py-2">{p.protocol}</td>
                      <td className="px-3 py-2">{p.purpose}</td>
                      <td className="px-3 py-2">{p.required}</td>
                      <td className="px-3 py-2">
                        <code className="text-xs">{p.command}</code>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="text-sm text-muted-foreground">
              After adding rules, run <code>sudo ufw reload</code>. With
              firewalld:{" "}
              <code>sudo firewall-cmd --permanent --add-port=PORT/tcp</code>{" "}
              then <code>sudo firewall-cmd --reload</code>.
            </p>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part1-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part1-heading" className="flex items-center gap-2">
              <StepNumber n={1} /> Prepare the remote server (VPS)
            </CardTitle>
            <CardDescription>
              Run these commands on the remote server via SSH.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div>
              <h4 className="mb-1 font-medium">1.1 Create user (if needed)</h4>
              <CodeBlock>sudo adduser deploy</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Or use an existing user (e.g. ubuntu, root).
              </p>
            </div>

            <div>
              <h4 className="mb-1 font-medium">1.2 Enable linger</h4>
              <p className="mb-1 text-sm text-muted-foreground">
                Required so systemd user services keep running after logout.
              </p>
              <CodeBlock>{`sudo loginctl enable-linger <USER>\n# Example: sudo loginctl enable-linger deploy`}</CodeBlock>
            </div>

            <div>
              <h4 className="mb-1 font-medium">
                1.3 Configure passwordless sudo{" "}
                <Badge variant="pending" className="ml-1 text-xs">
                  non-root only
                </Badge>
              </h4>
              <p className="mb-2 text-sm text-muted-foreground">
                If SSH user is not root, provisioning needs <code>sudo</code> to
                install Docker and Traefik. The provisioner uses{" "}
                <code>sudo -n</code> (no password prompt).
              </p>
              <p className="mb-1 text-sm font-medium">
                Option A {"\u2014"} Full sudo (simplest):
              </p>
              <CodeBlock>{`echo "deploy ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/deploy`}</CodeBlock>
              <p className="mt-3 mb-1 text-sm font-medium">
                Option B {"\u2014"} Restricted sudo (more secure):
              </p>
              <CodeBlock>{`cat << 'EOF' | sudo tee /etc/sudoers.d/deploy\ndeploy ALL=(ALL) NOPASSWD: /usr/bin/sh -c *get.docker.com*\ndeploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl start docker\ndeploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable docker\ndeploy ALL=(ALL) NOPASSWD: /usr/sbin/usermod -aG docker *\ndeploy ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /opt/traefik*\ndeploy ALL=(ALL) NOPASSWD: /usr/bin/tee /opt/traefik/*\nEOF`}</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                If connecting as <strong>root</strong>, skip this step.
              </p>
            </div>

            <div>
              <h4 className="mb-1 font-medium">1.4 SSH access</h4>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-md border p-3">
                  <p className="text-sm font-medium flex items-center gap-1.5">
                    <KeyRound className="h-4 w-4" /> Option A {"\u2014"} SSH Key
                    (recommended)
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Add the public key to <code>~/.ssh/authorized_keys</code>.
                    Test: <code>ssh deploy@IP -p PORT</code>
                  </p>
                </div>
                <div className="rounded-md border p-3">
                  <p className="text-sm font-medium flex items-center gap-1.5">
                    <KeyRound className="h-4 w-4" /> Option B {"\u2014"}{" "}
                    Password
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Ensure the user has a password set. flowDeploy stores it
                    encrypted.
                  </p>
                </div>
              </div>
            </div>

            <div>
              <h4 className="mb-1 font-medium">1.5 Check systemd user</h4>
              <CodeBlock>{`ssh deploy@IP -p PORT\nsystemctl --user status`}</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                If you see &quot;Failed to connect to bus&quot;, redo step 1.2.
              </p>
            </div>

            <div>
              <h4 className="mb-1 font-medium">1.6 Open firewall ports</h4>
              <CodeBlock>{`sudo ufw allow 22/tcp\nsudo ufw allow 80/tcp\nsudo ufw allow 443/tcp\nsudo ufw allow 50052/tcp\nsudo ufw reload`}</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Port 8081 (Traefik Dashboard) is optional. Port 50051 only if
                exposing gRPC via Traefik on this server.
              </p>
            </div>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part2-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part2-heading" className="flex items-center gap-2">
              <StepNumber n={2} /> Configure the backend
            </CardTitle>
            <CardDescription>
              On the host where the flowDeploy backend runs.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="mb-1 font-medium">2.1 Generate encryption key</h4>
              <CodeBlock>openssl rand -base64 32</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Put the result in <code>TOKEN_ENCRYPTION_KEY</code> in .env.
              </p>
            </div>

            <div>
              <h4 className="mb-1 font-medium">2.2 Build agent binary</h4>
              <p className="mb-1 text-sm font-medium">Development:</p>
              <CodeBlock>{`cd apps/agent\ngo build -o ../../dist/agent ./cmd/agent`}</CodeBlock>
              <p className="mt-2 text-sm text-muted-foreground">
                <strong>Production (Docker):</strong> The backend image already
                includes the agent at <code>/app/agent</code>. Set{" "}
                <code>AGENT_BINARY_PATH=/app/agent</code>.
              </p>
            </div>

            <div>
              <h4 className="mb-2 font-medium">2.3 Environment variables</h4>
              <div className="overflow-x-auto rounded-md border">
                <table className="w-full min-w-[380px] text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-3 py-2 text-left font-medium">
                        Variable
                      </th>
                      <th className="px-3 py-2 text-left font-medium">
                        Example
                      </th>
                      <th className="px-3 py-2 text-left font-medium">
                        Required
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    {ENV_VARIABLES.map((v) => (
                      <tr key={v.name} className="border-b last:border-0">
                        <td className="px-3 py-1.5 font-mono text-xs text-foreground">
                          {v.name}
                        </td>
                        <td className="px-3 py-1.5 text-xs">{v.example}</td>
                        <td className="px-3 py-1.5">
                          {v.required ? (
                            <Badge variant="default" className="text-xs">
                              Yes
                            </Badge>
                          ) : (
                            <span className="text-xs">No</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <p className="text-sm text-muted-foreground">
              Restart the backend after changing .env. If using Docker Compose,
              ensure <code>50051:50051</code> is in the Traefik ports.
            </p>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part3-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part3-heading" className="flex items-center gap-2">
              <StepNumber n={3} /> Add server in the panel
            </CardTitle>
            <CardDescription>
              Go to Servers {"\u2192"} Add Server and fill in the form.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="overflow-x-auto rounded-md border">
              <table className="w-full min-w-[380px] text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-3 py-2 text-left font-medium">Field</th>
                    <th className="px-3 py-2 text-left font-medium">
                      Description
                    </th>
                  </tr>
                </thead>
                <tbody className="text-muted-foreground">
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      Name
                    </td>
                    <td className="px-3 py-2">
                      Friendly name (e.g. production, staging)
                    </td>
                  </tr>
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      Host
                    </td>
                    <td className="px-3 py-2">VPS IP or hostname</td>
                  </tr>
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      SSH Port
                    </td>
                    <td className="px-3 py-2">SSH port (default: 22)</td>
                  </tr>
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      SSH User
                    </td>
                    <td className="px-3 py-2">
                      User from step 1.1 (e.g. deploy, root)
                    </td>
                  </tr>
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      SSH Key
                    </td>
                    <td className="px-3 py-2">
                      Full private key (optional if using password)
                    </td>
                  </tr>
                  <tr className="border-b">
                    <td className="px-3 py-2 font-medium text-foreground">
                      SSH Password
                    </td>
                    <td className="px-3 py-2">
                      User password (optional if using key)
                    </td>
                  </tr>
                  <tr>
                    <td className="px-3 py-2 font-medium text-foreground">
                      ACME Email{" "}
                      <Badge variant="pending" className="ml-1.5 text-xs">
                        important
                      </Badge>
                    </td>
                    <td className="px-3 py-2">
                      {
                        "Email for automatic TLS certificates via Let's Encrypt (Traefik). Without this, Traefik will not be installed and apps won't have HTTPS."
                      }
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <Alert>
              <Globe className="h-4 w-4" />
              <AlertTitle>ACME Email</AlertTitle>
              <AlertDescription>
                {
                  "The ACME Email is required for Traefik to obtain TLS certificates from Let's Encrypt. Without it, Traefik will not be provisioned and deployed applications will not have automatic HTTPS."
                }
              </AlertDescription>
            </Alert>

            <p className="text-sm text-muted-foreground">
              You must provide either an SSH key or password. Both can be used
              together. After saving, the server appears as{" "}
              <Badge variant="pending" className="text-xs">
                Pending
              </Badge>{" "}
              until provisioned.
            </p>

            <Button asChild variant="outline" className="mt-2">
              <Link to={ROUTES.SERVERS}>
                Open Servers <ChevronRight className="ml-1 h-4 w-4" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part4-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part4-heading" className="flex items-center gap-2">
              <StepNumber n={4} /> Provision and verify
            </CardTitle>
            <CardDescription>
              Click Provision on the server card. The backend will automatically
              install Docker, Traefik, and the agent via SSH.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Progress is streamed in real-time via SSE (Server-Sent Events).
              You will see each step update in the panel. When the agent
              registers and sends heartbeats, the status changes to{" "}
              <Badge variant="success" className="text-xs">
                Online
              </Badge>
              .
            </p>

            <div>
              <h4 className="mb-1 font-medium">Verify on the remote server</h4>
              <CodeBlock>{`# Check agent\nsystemctl --user status paasdeploy-agent\njournalctl --user -u paasdeploy-agent -f\n\n# Check Docker\ndocker --version\ndocker ps                    # should show Traefik running\ndocker network ls            # should show "paasdeploy"\n\n# Check Traefik\ndocker inspect traefik --format '{{.Config.Image}}'\ncurl -s http://localhost:8081/api/overview | head -c 200`}</CodeBlock>
            </div>

            <div>
              <h4 className="mb-1 font-medium">Health check via API</h4>
              <CodeBlock>{`GET /paas-deploy/v1/servers/:id/health\n# Expected: { "status": "ok", "latencyMs": ... }`}</CodeBlock>
            </div>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="troubleshooting-heading">
        <Card>
          <CardHeader>
            <CardTitle
              id="troubleshooting-heading"
              className="flex items-center gap-2"
            >
              <Terminal className="h-5 w-5" />
              Troubleshooting
            </CardTitle>
            <CardDescription>
              Common problems organized by category.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="ssh" className="w-full">
              <TabsList className="mb-4 flex flex-wrap h-auto gap-1">
                <TabsTrigger value="ssh" className="text-xs sm:text-sm">
                  SSH
                </TabsTrigger>
                <TabsTrigger value="docker" className="text-xs sm:text-sm">
                  Docker
                </TabsTrigger>
                <TabsTrigger value="traefik" className="text-xs sm:text-sm">
                  Traefik
                </TabsTrigger>
                <TabsTrigger value="agent" className="text-xs sm:text-sm">
                  Agent
                </TabsTrigger>
                <TabsTrigger
                  value="multitenancy"
                  className="text-xs sm:text-sm"
                >
                  Access
                </TabsTrigger>
              </TabsList>
              <TabsContent value="ssh">
                <TroubleshootSection items={TROUBLE_SSH} />
              </TabsContent>
              <TabsContent value="docker">
                <TroubleshootSection items={TROUBLE_DOCKER} />
              </TabsContent>
              <TabsContent value="traefik">
                <TroubleshootSection items={TROUBLE_TRAEFIK} />
              </TabsContent>
              <TabsContent value="agent">
                <TroubleshootSection items={TROUBLE_AGENT} />
              </TabsContent>
              <TabsContent value="multitenancy">
                <TroubleshootSection items={TROUBLE_MULTITENANCY} />
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="checklist-heading">
        <Card>
          <CardHeader>
            <CardTitle
              id="checklist-heading"
              className="flex items-center gap-2"
            >
              <CheckCircle2 className="h-5 w-5" />
              Quick checklist
            </CardTitle>
            <CardDescription>
              Use this before and after provisioning.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div>
              <h4 className="mb-2 font-medium">On the remote server</h4>
              <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                {CHECKLIST_REMOTE.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
            <div>
              <h4 className="mb-2 font-medium">Backend</h4>
              <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                {CHECKLIST_BACKEND.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
            <div>
              <h4 className="mb-2 font-medium">Panel</h4>
              <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                {CHECKLIST_PANEL.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="filestructure-heading">
        <Card>
          <CardHeader>
            <CardTitle
              id="filestructure-heading"
              className="flex items-center gap-2"
            >
              <FolderTree className="h-5 w-5" />
              Files installed on the remote server
            </CardTitle>
            <CardDescription>
              After a successful provision, these files and containers exist on
              the remote server.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="mb-1 font-medium flex items-center gap-1.5">
                <Network className="h-4 w-4" /> Agent files
              </h4>
              <CodeBlock>{`~/paasdeploy-agent/\n  agent          # agent binary\n  ca.pem         # CA certificate\n  cert.pem       # agent certificate (mTLS)\n  key.pem        # agent private key\n\n~/.config/systemd/user/\n  paasdeploy-agent.service  # systemd service`}</CodeBlock>
            </div>
            <div>
              <h4 className="mb-1 font-medium flex items-center gap-1.5">
                <Container className="h-4 w-4" /> Traefik (if ACME Email
                configured)
              </h4>
              <CodeBlock>{`/opt/traefik/\n  traefik.yml              # static configuration\n  letsencrypt/\n    acme.json              # TLS certificates (auto-generated)\n\nDocker containers:\n  traefik    # reverse proxy (ports 80, 443, 50051, 8081)\n\nDocker network:\n  paasdeploy  # shared network between Traefik and apps`}</CodeBlock>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
