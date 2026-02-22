import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  Clock,
  GitBranch,
  Loader2,
  MoreVertical,
  Server,
  Timer,
  Trash2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { HealthIndicator } from "@/components/health-indicator";
import { StatusBadge } from "@/components/status-badge";
import { usePurgeApp } from "@/features/apps/hooks/use-apps";
import { useAppHealth } from "@/hooks/use-sse";
import { cn, formatDuration, formatRelativeTime } from "@/lib/utils";
import type { App, Deployment } from "@/types";
import { getRuntimeTag } from "../utils/tech-tags";
import { AppDeleteDialog } from "./app-delete-dialog";

interface AppRowProps {
  readonly app: App;
  readonly latestDeploy?: Deployment;
  readonly serverName?: string;
}

export function AppRow({ app, latestDeploy, serverName }: AppRowProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const navigate = useNavigate();
  const purgeApp = usePurgeApp();
  const { data: health } = useAppHealth(app.id);

  const deployment = latestDeploy ?? app.lastDeployment;
  const runtimeTag = app.runtime ? getRuntimeTag(app.runtime) : null;

  const handleDelete = () => {
    purgeApp.mutate(app.id, {
      onSuccess: () => {
        setShowDeleteDialog(false);
        navigate("/");
      },
    });
  };

  const isDeploying =
    deployment?.status === "running" || deployment?.status === "pending";

  return (
    <>
      <div
        className={cn(
          "relative flex items-center gap-4 rounded-lg border bg-card p-3 transition-colors group",
          isDeploying ? "border-primary/50" : "hover:bg-accent/50",
        )}
      >
        <Link to={`/apps/${app.id}`} className="absolute inset-0 z-0" />

        {isDeploying && (
          <div className="absolute inset-0 z-5 bg-background/60 backdrop-blur-[1px] rounded-lg flex items-center justify-center pointer-events-none">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
          </div>
        )}

        <div className="flex-1 min-w-0 flex items-center gap-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{app.name}</span>
              {runtimeTag && (
                <Badge
                  variant="outline"
                  className={`text-[10px] px-1.5 py-0 h-5 font-medium border ${runtimeTag.color} hidden sm:inline-flex`}
                >
                  {runtimeTag.name}
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
              {serverName && (
                <span className="flex items-center gap-1">
                  <Server className="h-3 w-3" />
                  <span className="truncate max-w-[120px]">{serverName}</span>
                </span>
              )}
              <span className="flex items-center gap-1">
                <GitBranch className="h-3 w-3" />
                {app.branch}
              </span>
              {app.lastDeployedAt && (
                <span className="hidden sm:flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatRelativeTime(app.lastDeployedAt)}
                </span>
              )}
              {deployment?.durationMs && deployment.status === "success" && (
                <span className="hidden md:flex items-center gap-1">
                  <Timer className="h-3 w-3" />
                  {formatDuration(deployment.durationMs)}
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 relative z-10">
          <HealthIndicator health={health} />
          {deployment && <StatusBadge status={deployment.status} />}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
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

      <AppDeleteDialog
        appName={app.name}
        isOpen={showDeleteDialog}
        isPending={purgeApp.isPending}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
      />
    </>
  );
}
