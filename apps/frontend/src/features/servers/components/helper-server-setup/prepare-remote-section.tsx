import { KeyRound } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CodeBlock } from "@/components/ui/code-block";
import { StepNumber } from "@/components/ui/step-number";

export function PrepareRemoteSection() {
  return (
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
            <h4 className="mb-1 font-medium">
              1.1 Create a dedicated user for the agent
            </h4>
            <p className="mb-2 text-sm text-muted-foreground">
              Create a dedicated user to run the deploy agent. This user needs
              access to Docker and systemd user services. You can also use an
              existing user (e.g. ubuntu, root).
            </p>
            <CodeBlock>{`# Create user with home directory and bash shell\nsudo useradd -m -s /bin/bash deploy\n\n# Add user to docker group (required for Docker access)\nsudo usermod -aG docker deploy\n\n# Enable linger so systemd user services survive logout\nsudo loginctl enable-linger deploy\n\n# Set up SSH directory with correct permissions\nsudo mkdir -p /home/deploy/.ssh\nsudo chmod 700 /home/deploy/.ssh\nsudo touch /home/deploy/.ssh/authorized_keys\nsudo chmod 600 /home/deploy/.ssh/authorized_keys\nsudo chown -R deploy:deploy /home/deploy/.ssh`}</CodeBlock>
            <p className="mt-1 text-sm text-muted-foreground">
              Replace <code>deploy</code> with your preferred username. If using
              an existing user, just ensure it belongs to the{" "}
              <code>docker</code> group and has linger enabled.
            </p>
          </div>

          <div>
            <h4 className="mb-1 font-medium">1.2 Verify user setup</h4>
            <p className="mb-1 text-sm text-muted-foreground">
              Confirm the user is correctly configured before proceeding.
            </p>
            <CodeBlock>{`# Check user exists and belongs to docker group\nid deploy\n# Expected: uid=...(deploy) gid=...(deploy) groups=...,docker\n\n# Check linger is enabled\nls /var/lib/systemd/linger/ | grep deploy\n\n# Test systemd user access\nsudo -u deploy XDG_RUNTIME_DIR=/run/user/$(id -u deploy) systemctl --user status`}</CodeBlock>
          </div>

          <div>
            <h4 className="mb-1 font-medium">
              1.3 Configure passwordless sudo (or use SSH password){" "}
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
                  <KeyRound className="h-4 w-4" /> Option B {"\u2014"} Password
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
              If you see &quot;Failed to connect to bus&quot;, ensure linger is
              enabled (step 1.1) and the user has a valid login session.
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
  );
}
