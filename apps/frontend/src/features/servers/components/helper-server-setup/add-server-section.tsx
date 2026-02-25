import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { ChevronRight, Globe } from "lucide-react";
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
import { StepNumber } from "@/components/ui/step-number";

export function AddServerSection() {
  return (
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
  );
}
