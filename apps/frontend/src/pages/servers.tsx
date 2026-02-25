import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { HelpCircle, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/page-header";
import { AddServerDialog } from "@/features/servers/components/add-server-dialog";
import { ServerList } from "@/features/servers/components/server-list";

export function ServersPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title="Remote Servers"
        description="Manage servers for remote deploy"
        icon={Server}
        actions={
          <div className="flex flex-wrap gap-2">
            <Button asChild variant="ghost" size="sm">
              <Link
                to={ROUTES.HELPER_SERVER_SETUP}
                className="inline-flex items-center gap-2"
              >
                <HelpCircle className="h-4 w-4" aria-hidden="true" />
                Setup guide
              </Link>
            </Button>
            <AddServerDialog />
          </div>
        }
      />

      <section aria-labelledby="servers-heading">
        <h2 id="servers-heading" className="sr-only">
          Servers
        </h2>
        <ServerList />
      </section>
    </div>
  );
}
