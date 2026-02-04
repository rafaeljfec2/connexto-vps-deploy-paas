import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  Server,
  Terminal,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PageHeader } from "@/components/page-header";

function CodeBlock({ children }: { readonly children: string }) {
  return (
    <pre className="overflow-x-auto rounded-md border bg-muted/50 p-3 text-sm">
      <code>{children}</code>
    </pre>
  );
}

const TROUBLESHOOTING: ReadonlyArray<{
  problem: string;
  cause: string;
  solution: string;
}> = [
  {
    problem: "agent stopped / dial tcp ... 50051: i/o timeout",
    cause:
      "Agent cannot reach the backend gRPC server. Port 50051 not listening on the backend host or blocked by firewall.",
    solution:
      'On the backend server: (1) Ensure Docker Compose publishes port 50051 (Traefik: ports include "50051:50051"). (2) Run docker compose up -d. (3) Open firewall: sudo ufw allow 50051/tcp && sudo ufw reload (or firewalld: sudo firewall-cmd --permanent --add-port=50051/tcp && sudo firewall-cmd --reload). (4) From the agent server test: nc -vz BACKEND_IP 50051.',
  },
  {
    problem: "no such host (e.g. paasdeploy.connexto.com.br)",
    cause: "DNS on the agent server cannot resolve the backend hostname.",
    solution:
      "Use a hostname that resolves from the agent server, or use the backend IP in GRPC_SERVER_ADDR and in the agent -server-addr. Ensure the agent is started with the same address the backend is reachable at.",
  },
  {
    problem: "ssh connect: i/o timeout",
    cause: "Firewall blocking SSH port.",
    solution:
      "Allow the SSH port (e.g. 22) on the remote server: sudo ufw allow 22/tcp (or the port you use); sudo ufw reload.",
  },
  {
    problem: "unable to authenticate (SSH)",
    cause: "Wrong key or password.",
    solution:
      "Check credentials in the panel. Test manually: ssh -p PORT USER@HOST. Ensure the key has no passphrase or provide the correct password.",
  },
  {
    problem:
      "provision failed: EOF / Unit ... could not be found / Failed to connect to bus",
    cause:
      "Linger not enabled for the user, so systemd user services do not run after logout.",
    solution:
      "On the remote server run: sudo loginctl enable-linger USER (e.g. deploy or oab-api). Then log in as that user and run systemctl --user status to confirm.",
  },
  {
    problem: 'Status stays "Provisioning" or "Pending"',
    cause:
      "SSH connection dropped, process stuck, or agent not connecting back.",
    solution:
      'Check backend logs for provision errors. On the remote server check: systemctl --user status paasdeploy-agent and journalctl --user -u paasdeploy-agent -n 50. Fix gRPC/firewall if you see "agent stopped" or connection errors.',
  },
];

const CHECKLIST_REMOTE: ReadonlyArray<string> = [
  "User created or existing",
  "sudo loginctl enable-linger <user> executed",
  "SSH working (key or password)",
  "systemctl --user status works without error",
];

const CHECKLIST_BACKEND: ReadonlyArray<string> = [
  "TOKEN_ENCRYPTION_KEY set",
  "AGENT_BINARY_PATH points to the agent binary",
  "GRPC_ENABLED=true and GRPC_SERVER_ADDR set (host:50051 reachable by the agent)",
  "Port 50051 published (e.g. in Docker Compose for Traefik) and firewall allows 50051/tcp",
];

const CHECKLIST_PANEL: ReadonlyArray<string> = [
  "Server added (Name, Host, Port, User, Key or Password)",
  "Provision clicked",
  "Status changed to Online",
];

export function HelperServerSetupPage() {
  return (
    <div className="space-y-8">
      <PageHeader
        backTo={ROUTES.SERVERS}
        title="Server setup guide"
        description="Step-by-step tutorial to create and connect a remote server for deploy"
        icon={Server}
      />

      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Overview</AlertTitle>
        <AlertDescription>
          Prepare the remote VPS, configure the backend, add the server in the
          panel, then provision. The agent will connect to the backend via gRPC
          (port 50051).
        </AlertDescription>
      </Alert>

      <section aria-labelledby="part1-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part1-heading" className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                1
              </span>{" "}
              Prepare the remote server (VPS)
            </CardTitle>
            <CardDescription>
              Run these commands on the remote server (SSH as root or with
              sudo).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="mb-1 font-medium">1.1 Create user (if needed)</h4>
              <CodeBlock>sudo adduser deploy</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Or use an existing user (e.g. oab-api).
              </p>
            </div>
            <div>
              <h4 className="mb-1 font-medium">1.2 Enable linger</h4>
              <p className="mb-1 text-sm text-muted-foreground">
                Required so systemd user services keep running after logout.
              </p>
              <CodeBlock>sudo loginctl enable-linger &lt;USER&gt;</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Example: <code>sudo loginctl enable-linger oab-api</code>
              </p>
            </div>
            <div>
              <h4 className="mb-1 font-medium">1.3 SSH access</h4>
              <p className="text-sm text-muted-foreground">
                User must log in via SSH with private key or password. Add
                public key to <code>~/.ssh/authorized_keys</code> or set
                password. Test: <code>ssh deploy@IP -p PORT</code>.
              </p>
            </div>
            <div>
              <h4 className="mb-1 font-medium">1.4 Check systemd user</h4>
              <p className="mb-1 text-sm text-muted-foreground">
                Log in as the chosen user and run:
              </p>
              <CodeBlock>{`systemctl --user status`}</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                If you see &quot;Failed to connect to bus&quot;, redo step 1.2.
              </p>
            </div>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part2-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part2-heading" className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                2
              </span>{" "}
              Configure the backend
            </CardTitle>
            <CardDescription>
              On the host where the backend runs (or in Docker).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="mb-1 font-medium">2.1 Encryption key</h4>
              <CodeBlock>openssl rand -base64 32</CodeBlock>
              <p className="mt-1 text-sm text-muted-foreground">
                Put the result in <code>TOKEN_ENCRYPTION_KEY</code> in .env.
              </p>
            </div>
            <div>
              <h4 className="mb-1 font-medium">2.2 Agent binary</h4>
              <p className="mb-1 text-sm text-muted-foreground">
                Backend image usually includes the agent. Set{" "}
                <code>AGENT_BINARY_PATH</code> (e.g. /app/agent). For Docker
                Compose ensure port 50051 is published for Traefik.
              </p>
            </div>
            <div>
              <h4 className="mb-1 font-medium">2.3 .env variables</h4>
              <div className="overflow-x-auto rounded-md border bg-muted/30 p-3 text-sm">
                <table className="w-full min-w-[280px]">
                  <thead>
                    <tr className="border-b text-left">
                      <th className="pb-2 pr-2 font-medium">Variable</th>
                      <th className="pb-2 font-medium">Example</th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr className="border-b">
                      <td className="py-1.5 pr-2">TOKEN_ENCRYPTION_KEY</td>
                      <td className="py-1.5">(output of openssl)</td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-1.5 pr-2">GRPC_ENABLED</td>
                      <td className="py-1.5">true</td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-1.5 pr-2">GRPC_PORT</td>
                      <td className="py-1.5">50051</td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-1.5 pr-2">GRPC_SERVER_ADDR</td>
                      <td className="py-1.5">
                        host:50051 (reachable by agent)
                      </td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-1.5 pr-2">AGENT_BINARY_PATH</td>
                      <td className="py-1.5">/app/agent or path to binary</td>
                    </tr>
                    <tr>
                      <td className="py-1.5 pr-2">AGENT_GRPC_PORT</td>
                      <td className="py-1.5">50052</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              Restart the backend after changing .env. If using Docker Compose,
              ensure <code>50051:50051</code> is in the Traefik ports and run{" "}
              <code>./start.sh</code> (it can open the firewall for 50051
              automatically when ufw is active).
            </p>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part3-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part3-heading" className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                3
              </span>{" "}
              Add server in the panel
            </CardTitle>
            <CardDescription>
              Servers → Add Server. Fill name, host, SSH port, user, and key or
              password.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              You must provide either SSH key or password. After saving, the
              server appears as Pending until you provision it.
            </p>
            <Button asChild variant="outline" className="mt-4">
              <Link to={ROUTES.SERVERS}>
                Open Servers
                <ChevronRight className="ml-1 h-4 w-4" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="part4-heading">
        <Card>
          <CardHeader>
            <CardTitle id="part4-heading" className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">
                4
              </span>{" "}
              Provision and verify
            </CardTitle>
            <CardDescription>
              Click Provision on the server card. Backend will SSH in, install
              the agent and start the systemd user service.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              When the agent registers and sends heartbeats, status becomes
              Online. On the remote server you can check:
            </p>
            <CodeBlock>
              {`systemctl --user status paasdeploy-agent
journalctl --user -u paasdeploy-agent -f`}
            </CodeBlock>
            <p className="text-sm text-muted-foreground">
              Health: <code>GET /paas-deploy/v1/servers/:id/health</code> should
              return <code>{`{ "status": "ok", "latencyMs": ... }`}</code>.
            </p>
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
              Common problems and how to fix them.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {TROUBLESHOOTING.map((item) => (
              <div
                key={item.problem}
                className="rounded-lg border bg-muted/20 p-4 space-y-2"
              >
                <p className="font-medium text-sm">{item.problem}</p>
                <p className="text-sm text-muted-foreground">
                  <span className="font-medium">Cause:</span> {item.cause}
                </p>
                <p className="text-sm text-muted-foreground">
                  <span className="font-medium">Solution:</span> {item.solution}
                </p>
              </div>
            ))}
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
    </div>
  );
}
