import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { LayoutTemplate } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/page-header";
import { ContainerList, CreateContainerDialog } from "@/features/containers";

export function ContainersPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Containers"
        description="Manage Docker containers running on your server."
        actions={
          <>
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
      <ContainerList />
    </div>
  );
}
