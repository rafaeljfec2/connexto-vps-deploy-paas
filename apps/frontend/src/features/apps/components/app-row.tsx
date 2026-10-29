import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  AlertTriangle,
  Building2,
  Clock,
  ExternalLink,
  GitBranch,
  Loader2,
  MoreVertical,
  Package,
  Server,
  Tag,
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
import { StatusBadge } from "@/components/status-badge";
import { usePurgeApp } from "@/features/apps/hooks/use-apps";
import { formatDuration, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { App, Deployment } from "@/types";
import { extractRepoOwner } from "../utils/org-helpers";
import { detectTechTags, getRuntimeTag } from "../utils/tech-tags";
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

  const deployment = latestDeploy ?? app.lastDeployment;
  const runtimeTag = app.runtime
    ? getRuntimeTag(app.runtime)
    : (detectTechTags(app.name, app.workdir, app.repositoryUrl)[0] ?? null);

  const repoOwner = extractRepoOwner(app.repositoryUrl);
  const displayOrgOrServer = repoOwner ?? serverName ?? null;

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
  const isFailed = deployment?.status === "failed";

  return (
    <>
      <div
        className={cn(
          "group relative flex items-center justify-between gap-4 rounded-xl border p-3 sm:px-4 sm:py-3 transition-all",
          isFailed
            ? "border-destructive/40 bg-destructive/5 hover:border-destructive/60 hover:bg-destructive/10"
            : isDeploying
              ? "border-primary/50 bg-card/80"
              : "border-border/60 bg-card/50 backdrop-blur-sm hover:border-border hover:bg-card/80 shadow-sm",
        )}
      >
        <Link
          to={`/apps/${app.id}`}
          className="absolute inset-0 z-0 rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          aria-label={`View ${app.name} details`}
        />

        {isDeploying && (
          <div className="absolute inset-0 z-5 bg-card/40 backdrop-blur-[1px] rounded-xl flex items-center justify-center pointer-events-none">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
          </div>
        )}

        {/* Identification & Metadata Area */}
        <div className="flex items-center gap-3.5 min-w-0 flex-1">
          {/* App Icon Container */}
          <div
            className={cn(
              "h-10 w-10 shrink-0 rounded-lg border flex items-center justify-center transition-colors",
              isFailed
                ? "border-destructive/30 bg-destructive/10 text-destructive"
                : "border-border/50 bg-muted/40 text-muted-foreground group-hover:text-foreground group-hover:border-border",
            )}
            aria-hidden="true"
          >
            <Package className="h-5 w-5" />
          </div>

          <div className="min-w-0 flex-1">
            {/* Identification Header: Name, Tech Tag, Version */}
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-semibold text-sm text-foreground group-hover:text-primary transition-colors truncate">
                {app.name}
              </span>

              {runtimeTag && (
                <Badge
                  variant="outline"
                  className={cn(
                    "text-[10px] px-1.5 py-0 h-4.5 font-medium border",
                    runtimeTag.color,
                  )}
                >
                  {runtimeTag.name}
                </Badge>
              )}

              {app.appVersion && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-4.5 font-mono text-muted-foreground border-border/60 bg-muted/30"
                >
                  <Tag className="h-2.5 w-2.5 mr-0.5" aria-hidden="true" />v
                  {app.appVersion}
                </Badge>
              )}
            </div>

            {/* Metadata Row: Org/Server, Branch, Last Deployed, Duration */}
            <div className="flex flex-wrap items-center gap-x-3.5 gap-y-1 text-xs text-muted-foreground mt-1">
              {displayOrgOrServer && (
                <span className="flex items-center gap-1">
                  {serverName && !repoOwner ? (
                    <Server className="h-3 w-3 shrink-0" aria-hidden="true" />
                  ) : (
                    <Building2
                      className="h-3 w-3 shrink-0"
                      aria-hidden="true"
                    />
                  )}
                  <span className="truncate max-w-[120px]">
                    {displayOrgOrServer}
                  </span>
                </span>
              )}

              <span className="flex items-center gap-1">
                <GitBranch className="h-3 w-3 shrink-0" aria-hidden="true" />
                <span className="font-mono text-[11px]">{app.branch}</span>
              </span>

              {app.lastDeployedAt && (
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3 shrink-0" aria-hidden="true" />
                  <span>{formatRelativeTime(app.lastDeployedAt)}</span>
                </span>
              )}

              {deployment?.durationMs && deployment.status === "success" && (
                <span className="hidden sm:flex items-center gap-1">
                  <Timer className="h-3 w-3 shrink-0" aria-hidden="true" />
                  <span>{formatDuration(deployment.durationMs)}</span>
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Actions and Status Area */}
        <div className="flex items-center gap-2 relative z-10 shrink-0">
          {isFailed && (
            <AlertTriangle
              className="h-4 w-4 text-destructive shrink-0"
              aria-label="Deployment failed"
            />
          )}

          {deployment && <StatusBadge status={deployment.status} />}

          {app.repositoryUrl && (
            <Button
              asChild
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-muted-foreground hover:text-foreground transition-colors"
            >
              <a
                href={app.repositoryUrl}
                target="_blank"
                rel="noopener noreferrer"
                title="Open repository"
                onClick={(e) => e.stopPropagation()}
              >
                <ExternalLink className="h-4 w-4" />
                <span className="sr-only">Open repository</span>
              </a>
            </Button>
          )}

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-muted-foreground hover:text-foreground opacity-100 sm:opacity-80 sm:group-hover:opacity-100 transition-opacity"
                onClick={(e) => e.stopPropagation()}
                aria-label={`Actions for ${app.name}`}
              >
                <MoreVertical className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                className="text-destructive focus:text-destructive cursor-pointer"
                onClick={(e) => {
                  e.stopPropagation();
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
