import { useState } from "react";
import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { LayoutTemplate } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/page-header";
import { ServerSelector } from "@/components/server-selector";
import { ContainerList, CreateContainerDialog } from "@/features/containers";

export function ContainersPage() {
  const [serverId, setServerId] = useState<string | undefined>();

  return (
    <div className="space-y-6">
      <PageHeader
        title="Containers"
        description="Manage Docker containers running on your server."
        actions={
          <>
            <ServerSelector value={serverId} onChange={setServerId} />
            <Button asChild variant="outline">
              <Link to={ROUTES.TEMPLATES}>
                <LayoutTemplate className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Templates</span>
              </Link>
            </Button>
            <CreateContainerDialog />
          </>
        }
      />
      <ContainerList serverId={serverId} />
    </div>
  );
}
