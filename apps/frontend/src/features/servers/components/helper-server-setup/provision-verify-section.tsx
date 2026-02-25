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

export function ProvisionVerifySection() {
  return (
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
            Progress is streamed in real-time via SSE (Server-Sent Events). You
            will see each step update in the panel. When the agent registers and
            sends heartbeats, the status changes to{" "}
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
  );
}
