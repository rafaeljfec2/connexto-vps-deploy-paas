import { Container, FolderTree, Network } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CodeBlock } from "@/components/ui/code-block";

export function FilesStructureSection() {
  return (
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
  );
}
