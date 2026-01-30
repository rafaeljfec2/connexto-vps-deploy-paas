import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  Clock,
  ExternalLink,
  Folder,
  GitBranch,
  MoreVertical,
  Trash2,
} from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { HealthIndicator } from "@/components/health-indicator";
import { IconText } from "@/components/icon-text";
import { StatusBadge } from "@/components/status-badge";
import { usePurgeApp } from "@/features/apps/hooks/use-apps";
import { useAppHealth } from "@/hooks/use-sse";
import { formatRelativeTime, formatRepositoryUrl } from "@/lib/utils";
import type { App, Deployment } from "@/types";

interface AppCardProps {
  readonly app: App;
  readonly latestDeploy?: Deployment;
}

export function AppCard({ app, latestDeploy }: AppCardProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const navigate = useNavigate();
  const purgeApp = usePurgeApp();
  const { data: health } = useAppHealth(app.id);

  const handleDelete = () => {
    purgeApp.mutate(app.id, {
      onSuccess: () => {
        setShowDeleteDialog(false);
        navigate("/");
      },
    });
  };

  return (
    <>
      <Card className="hover:bg-accent/50 transition-colors cursor-pointer group relative">
        <Link to={`/apps/${app.id}`} className="absolute inset-0 z-0" />

        <CardHeader className="pb-2">
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{app.name}</CardTitle>
            <div className="flex items-center gap-2">
              <HealthIndicator health={health} />
              {latestDeploy && <StatusBadge status={latestDeploy.status} />}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 relative z-10 opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={(e) => e.preventDefault()}
                  >
                    <MoreVertical className="h-4 w-4" />
                    <span className="sr-only">Open menu</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    className="text-destructive focus:text-destructive cursor-pointer"
                    onClick={(e) => {
                      e.preventDefault();
                      setShowDeleteDialog(true);
                    }}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-2">
          <IconText icon={GitBranch}>
            <span>{app.branch}</span>
          </IconText>

          <IconText icon={ExternalLink}>
            <span className="truncate">
              {formatRepositoryUrl(app.repositoryUrl)}
            </span>
          </IconText>

          {app.workdir && app.workdir !== "." && (
            <IconText icon={Folder}>
              <span className="truncate font-mono text-xs">{app.workdir}</span>
            </IconText>
          )}

          {app.lastDeployedAt && (
            <IconText icon={Clock}>
              <span>Deployed {formatRelativeTime(app.lastDeployedAt)}</span>
            </IconText>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {app.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the
              application, remove all containers, images, files, environment
              variables, and deployment history from the server.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={purgeApp.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={purgeApp.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {purgeApp.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
