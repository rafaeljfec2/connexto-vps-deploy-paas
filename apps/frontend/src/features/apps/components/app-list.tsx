import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Box, Grid3X3, List, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import { cn } from "@/lib/utils";
import { useApps } from "../hooks/use-apps";
import { AppCard } from "./app-card";
import { AppRow } from "./app-row";

type ViewMode = "grid" | "list";

const VIEW_MODE_KEY = "flowdeploy:app-list-view";

export function AppList() {
  const { data: apps, isLoading, error } = useApps();
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    const saved = localStorage.getItem(VIEW_MODE_KEY);
    return saved === "list" ? "list" : "grid";
  });

  useEffect(() => {
    localStorage.setItem(VIEW_MODE_KEY, viewMode);
  }, [viewMode]);

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
    <div className="space-y-4">
      <div className="flex justify-end">
        <div className="flex items-center rounded-md border bg-muted p-1">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 w-7 p-0",
              viewMode === "grid" && "bg-background shadow-sm",
            )}
            onClick={() => setViewMode("grid")}
          >
            <Grid3X3 className="h-4 w-4" />
            <span className="sr-only">Grid view</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 w-7 p-0",
              viewMode === "list" && "bg-background shadow-sm",
            )}
            onClick={() => setViewMode("list")}
          >
            <List className="h-4 w-4" />
            <span className="sr-only">List view</span>
          </Button>
        </div>
      </div>

      {viewMode === "grid" ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {apps.map((app) => (
            <AppCard key={app.id} app={app} />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {apps.map((app) => (
            <AppRow key={app.id} app={app} />
          ))}
        </div>
      )}
    </div>
  );
}
