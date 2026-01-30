import { Link } from "react-router-dom";
import { Box, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import { useApps } from "../hooks/use-apps";
import { AppCard } from "./app-card";

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
