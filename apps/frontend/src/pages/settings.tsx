import { Link } from "react-router-dom";
import { ArrowRight, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PageHeader } from "@/components/page-header";
import { CloudflareConnection } from "@/features/settings/components/cloudflare-connection";
import { GitHubLinkCard } from "@/features/settings/components/github-link-card";
import { NotificationSettings } from "@/features/settings/components/notification-settings";

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title="Settings"
        description="Manage your FlowDeploy integrations and preferences"
      />

      <div className="space-y-6">
        <section>
          <h2 className="text-lg font-semibold mb-4">Account</h2>
          <GitHubLinkCard />
        </section>

        <section>
          <h2 className="text-lg font-semibold mb-4">Server</h2>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                Migration Center
              </CardTitle>
              <CardDescription>
                Migrate from Nginx to Traefik and manage your server
                configuration
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild>
                <Link to="/settings/migration">
                  Open Migration Center
                  <ArrowRight className="h-4 w-4 ml-2" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        </section>

        <section>
          <h2 className="text-lg font-semibold mb-4">Integrations</h2>
          <div className="space-y-6">
            <CloudflareConnection />
            <NotificationSettings />
          </div>
        </section>
      </div>
    </div>
  );
}
