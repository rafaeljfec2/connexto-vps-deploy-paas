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
  );
}
