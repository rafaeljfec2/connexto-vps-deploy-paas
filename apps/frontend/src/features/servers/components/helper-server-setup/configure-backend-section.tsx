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
import { ENV_VARIABLES } from "@/features/servers/data/helper-server-setup";

export function ConfigureBackendSection() {
  return (
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
                    <th className="px-3 py-2 text-left font-medium">Example</th>
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
  );
}
