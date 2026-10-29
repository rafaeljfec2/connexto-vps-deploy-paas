import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  AlertTriangle,
  Building2,
  Clock,
  ExternalLink,
  Folder,
  GitBranch,
  Hammer,
  Loader2,
  MoreVertical,
  Package,
  Rocket,
  Server,
  Tag,
  Timer,
  Trash2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import type { App, Deployment, DeploymentSummary } from "@/types";
import { deriveDeployProgressState } from "../utils/deploy-progress";
import { extractRepoOwner } from "../utils/org-helpers";
import {
  type TechTag,
  detectTechTags,
  getRuntimeTag,
} from "../utils/tech-tags";
import { AppDeleteDialog } from "./app-delete-dialog";

const DEPLOY_ICON_MAP = {
  loader: Loader2,
  rocket: Rocket,
  hammer: Hammer,
} as const;

function DeployProgress({
  deploy,
}: {
  readonly deploy: Deployment | DeploymentSummary;
}) {
  const state = deriveDeployProgressState(deploy.logs, deploy.status);
  if (!state) return null;

  const Icon = DEPLOY_ICON_MAP[state.icon];
  const isRunning = deploy.status === "running";

  return (
    <div className="absolute inset-0 z-5 bg-card/80 backdrop-blur-[2px] rounded-xl flex flex-col items-center justify-center gap-2 p-4 pointer-events-none">
      <div className="flex items-center gap-2 text-primary">
        <Icon className={cn("h-5 w-5", isRunning && "animate-spin")} />
        <span className="font-medium text-sm">{state.phase}</span>
      </div>
      <div className="w-full max-w-[200px] h-1.5 bg-muted rounded-full overflow-hidden">
        <div
          className="h-full bg-primary rounded-full transition-all duration-500 ease-out"
          style={{ width: `${state.progress}%` }}
        />
      </div>
      <span className="text-[10px] text-muted-foreground font-mono">
        {deploy.commitSha?.slice(0, 7)}
      </span>
    </div>
  );
}

function mergeTags(
  runtimeTag: TechTag | null,
  inferredTags: readonly TechTag[],
): readonly TechTag[] {
  if (!runtimeTag) return inferredTags;
  return [
    runtimeTag,
    ...inferredTags.filter((tag) => tag.name !== runtimeTag.name),
  ];
}

interface AppCardProps {
  readonly app: App;
  readonly latestDeploy?: Deployment;
  readonly serverName?: string;
}

export function AppCard({ app, latestDeploy, serverName }: AppCardProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const navigate = useNavigate();
  const purgeApp = usePurgeApp();
  const runtimeTag = app.runtime ? getRuntimeTag(app.runtime) : null;
  const inferredTags = detectTechTags(app.name, app.workdir, app.repositoryUrl);
  const techTags = mergeTags(runtimeTag, inferredTags);

  const deployment = latestDeploy ?? app.lastDeployment;
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
      <Card
        className={cn(
          "transition-all cursor-pointer group relative overflow-hidden rounded-xl border",
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

        {deployment && <DeployProgress deploy={deployment} />}

        <CardHeader className="pb-3 pt-4 px-4">
          <div className="flex items-start justify-between gap-2">
            <div className="flex items-start gap-3 min-w-0 flex-1">
              <div
                className={cn(
                  "h-9 w-9 shrink-0 rounded-lg border flex items-center justify-center transition-colors",
                  isFailed
                    ? "border-destructive/30 bg-destructive/10 text-destructive"
                    : "border-border/50 bg-muted/40 text-muted-foreground group-hover:text-foreground group-hover:border-border",
                )}
                aria-hidden="true"
              >
                <Package className="h-4.5 w-4.5" />
              </div>

              <div className="min-w-0 flex-1">
                <CardTitle className="text-sm font-semibold truncate group-hover:text-primary transition-colors">
                  {app.name}
                </CardTitle>

                <div className="flex flex-wrap items-center gap-1.5 mt-1.5">
                  {techTags[0] && (
                    <Badge
                      variant="outline"
                      className={cn(
                        "text-[10px] px-1.5 py-0 h-4.5 font-medium border",
                        techTags[0].color,
                      )}
                    >
                      {techTags[0].name}
                    </Badge>
                  )}

                  {app.appVersion && (
                    <Badge
                      variant="outline"
                      className="text-[10px] px-1.5 py-0 h-4.5 font-mono text-muted-foreground border-border/60 bg-muted/30"
                    >
                      <Tag className="h-2.5 w-2.5 mr-0.5" />v{app.appVersion}
                    </Badge>
                  )}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-1.5 relative z-10 shrink-0">
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
                  className="h-7 w-7 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <a
                    href={app.repositoryUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Open repository"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                    <span className="sr-only">Open repository</span>
                  </a>
                </Button>
              )}

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-muted-foreground hover:text-foreground transition-opacity"
                    onClick={(e) => e.stopPropagation()}
                    aria-label={`Actions for ${app.name}`}
                  >
                    <MoreVertical className="h-3.5 w-3.5" />
                    <span className="sr-only">Open menu</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    variant="destructive"
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
        </CardHeader>

        <CardContent className="px-4 pb-4 pt-0">
          <div className="grid grid-cols-2 gap-2 text-xs text-muted-foreground border-t border-border/40 pt-3 mt-1">
            {displayOrgOrServer && (
              <span className="flex items-center gap-1.5 truncate">
                {serverName && !repoOwner ? (
                  <Server className="h-3 w-3 shrink-0" aria-hidden="true" />
                ) : (
                  <Building2 className="h-3 w-3 shrink-0" aria-hidden="true" />
                )}
                <span className="truncate">{displayOrgOrServer}</span>
              </span>
            )}

            <span className="flex items-center gap-1.5 truncate">
              <GitBranch className="h-3 w-3 shrink-0" aria-hidden="true" />
              <span className="font-mono text-[11px] truncate">
                {app.branch}
              </span>
            </span>

            {app.lastDeployedAt && (
              <span className="flex items-center gap-1.5 truncate">
                <Clock className="h-3 w-3 shrink-0" aria-hidden="true" />
                <span className="truncate">
                  {formatRelativeTime(app.lastDeployedAt)}
                </span>
              </span>
            )}

            {deployment?.durationMs && deployment.status === "success" && (
              <span className="flex items-center gap-1.5 truncate">
                <Timer className="h-3 w-3 shrink-0" aria-hidden="true" />
                <span className="truncate">
                  {formatDuration(deployment.durationMs)}
                </span>
              </span>
            )}

            {app.workdir && app.workdir !== "." && (
              <span className="flex items-center gap-1.5 truncate col-span-2 text-[11px] font-mono text-muted-foreground/80">
                <Folder className="h-3 w-3 shrink-0" aria-hidden="true" />
                <span className="truncate">{app.workdir}</span>
              </span>
            )}
          </div>
        </CardContent>
      </Card>

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
