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
import { cn, formatRelativeTime, formatRepositoryUrl } from "@/lib/utils";
import type { App, Deployment } from "@/types";

function DeployProgress({ deploy }: { readonly deploy: Deployment }) {
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
    <div className="absolute inset-0 z-5 bg-background/80 backdrop-blur-[2px] rounded-lg flex flex-col items-center justify-center gap-2 p-4">
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

interface TechTag {
  readonly name: string;
  readonly color: string;
}

function getRuntimeTag(runtime: string): TechTag | null {
  const tags: Record<string, TechTag> = {
    go: {
      name: "Go",
      color: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
    },
    node: {
      name: "Node.js",
      color: "bg-green-500/20 text-green-400 border-green-500/30",
    },
    python: {
      name: "Python",
      color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    },
    rust: {
      name: "Rust",
      color: "bg-orange-500/20 text-orange-400 border-orange-500/30",
    },
    java: {
      name: "Java",
      color: "bg-red-500/20 text-red-400 border-red-500/30",
    },
    ruby: {
      name: "Ruby",
      color: "bg-red-400/20 text-red-300 border-red-400/30",
    },
    php: {
      name: "PHP",
      color: "bg-indigo-500/20 text-indigo-400 border-indigo-500/30",
    },
    dotnet: {
      name: ".NET",
      color: "bg-violet-500/20 text-violet-400 border-violet-500/30",
    },
    elixir: {
      name: "Elixir",
      color: "bg-purple-500/20 text-purple-400 border-purple-500/30",
    },
  };
  return tags[runtime] ?? null;
}

function detectTechTags(app: App): readonly TechTag[] {
  const tags: TechTag[] = [];
  const nameAndWorkdir =
    `${app.name} ${app.workdir} ${app.repositoryUrl}`.toLowerCase();

  if (app.runtime) {
    const runtimeTag = getRuntimeTag(app.runtime);
    if (runtimeTag) {
      tags.push(runtimeTag);
    }
  } else if (
    nameAndWorkdir.includes("go") ||
    nameAndWorkdir.includes("golang") ||
    app.workdir.includes("cmd/")
  ) {
    tags.push({
      name: "Go",
      color: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
    });
  } else if (
    nameAndWorkdir.includes("node") ||
    nameAndWorkdir.includes("express") ||
    nameAndWorkdir.includes("nest")
  ) {
    tags.push({
      name: "Node.js",
      color: "bg-green-500/20 text-green-400 border-green-500/30",
    });
  } else if (
    nameAndWorkdir.includes("python") ||
    nameAndWorkdir.includes("django") ||
    nameAndWorkdir.includes("flask")
  ) {
    tags.push({
      name: "Python",
      color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    });
  } else if (nameAndWorkdir.includes("rust")) {
    tags.push({
      name: "Rust",
      color: "bg-orange-500/20 text-orange-400 border-orange-500/30",
    });
  } else if (
    nameAndWorkdir.includes("java") ||
    nameAndWorkdir.includes("spring")
  ) {
    tags.push({
      name: "Java",
      color: "bg-red-500/20 text-red-400 border-red-500/30",
    });
  }

  if (nameAndWorkdir.includes("api")) {
    tags.push({
      name: "API",
      color: "bg-purple-500/20 text-purple-400 border-purple-500/30",
    });
  }

  if (nameAndWorkdir.includes("frontend") || nameAndWorkdir.includes("react")) {
    tags.push({
      name: "Frontend",
      color: "bg-blue-500/20 text-blue-400 border-blue-500/30",
    });
  }

  if (nameAndWorkdir.includes("worker") || nameAndWorkdir.includes("job")) {
    tags.push({
      name: "Worker",
      color: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    });
  }

  return tags;
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
  const techTags = detectTechTags(app);

  const handleDelete = () => {
    purgeApp.mutate(app.id, {
      onSuccess: () => {
        setShowDeleteDialog(false);
        navigate("/");
      },
    });
  };

  const isDeploying =
    latestDeploy?.status === "running" || latestDeploy?.status === "pending";

  return (
    <>
      <Card
        className={cn(
          "transition-colors cursor-pointer group relative overflow-hidden",
          isDeploying ? "border-primary/50" : "hover:bg-accent/50",
        )}
      >
        <Link to={`/apps/${app.id}`} className="absolute inset-0 z-0" />

        {latestDeploy && <DeployProgress deploy={latestDeploy} />}

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
