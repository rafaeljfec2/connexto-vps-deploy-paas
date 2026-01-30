import { Link } from "react-router-dom";
import { Plus, Box } from "lucide-react";
import { useApps } from "../hooks/use-apps";
import { AppCard } from "./app-card";
import { Button } from "@/components/ui/button";
import { LoadingGrid } from "@/components/loading-grid";
import { ErrorMessage } from "@/components/error-message";
import { EmptyState } from "@/components/empty-state";

export function AppList() {
  const { data: apps, isLoading, error } = useApps();

  if (isLoading) {
    return <LoadingGrid count={6} columns={3} />;
  }

  if (error) {
    return <ErrorMessage message="Failed to load applications" />;
  }

  if (!apps || apps.length === 0) {
    return (
      <EmptyState
        icon={Box}
        title="No applications yet"
        description="Connect your first GitHub repository to start deploying."
        action={
          <Button asChild>
            <Link to="/apps/new">
              <Plus className="h-4 w-4 mr-2" />
              Connect Repository
            </Link>
          </Button>
        }
      />
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {apps.map((app) => (
        <AppCard key={app.id} app={app} />
      ))}
    </div>
  );
}
