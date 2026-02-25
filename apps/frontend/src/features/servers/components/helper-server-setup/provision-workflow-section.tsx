import { Container, RefreshCw, Workflow } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PROVISION_STEPS } from "@/features/servers/data/helper-server-setup";

export function ProvisionWorkflowSection() {
  return (
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
            When you click <strong>Provision</strong>, the backend automatically
            executes these steps via SSH.
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
                  <strong>Docker</strong> {"\u2014"} via get.docker.com (if not
                  installed)
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
              You can re-provision safely. Docker is skipped if already running.
              The network is skipped if it exists. Traefik is skipped if running
              with the correct version (or upgraded automatically). The agent
              binary and certs are always overwritten.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </section>
  );
}
