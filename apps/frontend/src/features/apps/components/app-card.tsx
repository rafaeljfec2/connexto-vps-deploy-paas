import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  Clock,
  ExternalLink,
  Folder,
  GitBranch,
  Hammer,
  Loader2,
  MoreVertical,
  Rocket,
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
import { HealthIndicator } from "@/components/health-indicator";
import { IconText } from "@/components/icon-text";
import { StatusBadge } from "@/components/status-badge";
import { usePurgeApp } from "@/features/apps/hooks/use-apps";
import { useAppHealth } from "@/hooks/use-sse";
import {
  cn,
  formatDuration,
  formatRelativeTime,
  formatRepositoryUrl,
} from "@/lib/utils";
import type { App, Deployment, DeploymentSummary } from "@/types";
import {
  type TechTag,
  detectTechTags,
  getRuntimeTag,
} from "../utils/tech-tags";
import { AppDeleteDialog } from "./app-delete-dialog";

function DeployProgress({
  deploy,
}: {
  readonly deploy: Deployment | DeploymentSummary;
}) {
  const isRunning = deploy.status === "running";
  const isPending = deploy.status === "pending";

  if (!isRunning && !isPending) return null;

  const logs = deploy.logs ?? "";
  const isBuildPhase =
    logs.includes("[build]") && !logs.includes("Container deployed");
  const isDeployPhase =
    logs.includes("Deploying container") || logs.includes("[deploy]");
  const isHealthCheck = logs.includes("Waiting for health check");

  let phase = "Initializing";
  let Icon = Loader2;
  let progress = 10;

  if (isPending) {
    phase = "Queued";
    progress = 5;
  } else if (isHealthCheck) {
    phase = "Health check";
    Icon = Rocket;
    progress = 90;
  } else if (isDeployPhase) {
    phase = "Deploying";
    Icon = Rocket;
    progress = 75;
  } else if (isBuildPhase) {
    phase = "Building";
    Icon = Hammer;
    const buildSteps = logs.match(/Step \d+\/\d+/g) ?? [];
    const lastStep = buildSteps.at(-1);
    if (lastStep) {
      const match = /Step (\d+)\/(\d+)/.exec(lastStep);
      const currentStep = match?.[1];
      const totalSteps = match?.[2];
      if (currentStep && totalSteps) {
        const current = Number.parseInt(currentStep, 10);
        const total = Number.parseInt(totalSteps, 10);
        progress = Math.round((current / total) * 60) + 15;
        phase = `Building (${current}/${total})`;
      }
    } else {
      progress = 20;
    }
  }

  return (
    <div className="absolute inset-0 z-5 bg-background/80 backdrop-blur-[2px] rounded-lg flex flex-col items-center justify-center gap-2 p-4 pointer-events-none">
      <div className="flex items-center gap-2 text-primary">
        <Icon className={cn("h-5 w-5", isRunning && "animate-spin")} />
        <span className="font-medium text-sm">{phase}</span>
      </div>
      <div className="w-full max-w-[200px] h-1.5 bg-muted rounded-full overflow-hidden">
        <div
          className="h-full bg-primary rounded-full transition-all duration-500 ease-out"
          style={{ width: `${progress}%` }}
        />
      </div>
      <span className="text-[10px] text-muted-foreground">
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
}

export function AppCard({ app, latestDeploy }: AppCardProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const navigate = useNavigate();
  const purgeApp = usePurgeApp();
  const { data: health } = useAppHealth(app.id);
  const runtimeTag = app.runtime ? getRuntimeTag(app.runtime) : null;
  const inferredTags = detectTechTags(app.name, app.workdir, app.repositoryUrl);
  const techTags = mergeTags(runtimeTag, inferredTags);

  const deployment = latestDeploy ?? app.lastDeployment;

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
      <Card
        className={cn(
          "transition-colors cursor-pointer group relative overflow-hidden",
          isDeploying ? "border-primary/50" : "hover:bg-accent/50",
        )}
      >
        <Link to={`/apps/${app.id}`} className="absolute inset-0 z-0" />

        {deployment && <DeployProgress deploy={deployment} />}

        <CardHeader className="pb-2">
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{app.name}</CardTitle>
            <div className="flex items-center gap-2">
              <HealthIndicator health={health} />
              {deployment && <StatusBadge status={deployment.status} />}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 relative z-10 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
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
          {techTags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 pb-1">
              {techTags.map((tag) => (
                <Badge
                  key={tag.name}
                  variant="outline"
                  className={`text-[10px] px-1.5 py-0 h-5 font-medium border ${tag.color}`}
                >
                  {tag.name}
                </Badge>
              ))}
            </div>
          )}

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

          {deployment?.durationMs && deployment.status === "success" && (
            <IconText icon={Timer}>
              <span>Build time: {formatDuration(deployment.durationMs)}</span>
            </IconText>
          )}
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
