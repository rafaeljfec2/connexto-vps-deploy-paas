import { Link } from "react-router-dom";
import { GitBranch, Loader2, Plus, Server as ServerIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useServerApps } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";
import type { App } from "@/types";

function getDeployBadgeVariant(
  status: string,
): "default" | "destructive" | "secondary" {
  if (status === "success") return "default";
  if (status === "failed") return "destructive";
  return "secondary";
}

interface ServerAppsSectionProps {
  readonly serverId: string;
}

export function ServerAppsSection({ serverId }: ServerAppsSectionProps) {
  const { data: apps, isLoading } = useServerApps(serverId);

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-sm font-semibold">Apps</h2>
        <Button variant="outline" size="sm" className="h-7 text-xs" asChild>
          <Link to={`/apps/new?serverId=${serverId}`}>
            <Plus className="h-3.5 w-3.5 mr-1" />
            New App
          </Link>
        </Button>
      </div>

      {isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      )}

      {!isLoading && !apps?.length && (
        <Card>
          <CardContent className="py-8 text-center">
            <ServerIcon className="h-8 w-8 mx-auto mb-2 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">
              No apps deployed on this server
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3 text-xs"
              asChild
            >
              <Link to={`/apps/new?serverId=${serverId}`}>
                <Plus className="h-3.5 w-3.5 mr-1" />
                Deploy your first app
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}

      {!isLoading && apps && apps.length > 0 && (
        <div className="space-y-2">
          {apps.map((app) => (
            <ServerAppRow key={app.id} app={app} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ServerAppRowProps {
  readonly app: App;
}

function ServerAppRow({ app }: ServerAppRowProps) {
  const deployment = app.lastDeployment;
  const isDeploying =
    deployment?.status === "running" || deployment?.status === "pending";

  return (
    <Link
      to={`/apps/${app.id}`}
      className={cn(
        "relative flex items-center gap-4 rounded-lg border bg-card p-3 transition-colors",
        isDeploying ? "border-primary/50" : "hover:bg-accent/50",
      )}
    >
      {isDeploying && (
        <div className="absolute inset-0 z-[5] bg-background/60 backdrop-blur-[1px] rounded-lg flex items-center justify-center pointer-events-none">
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
        </div>
      )}

      <div className="flex-1 min-w-0">
        <span className="font-medium text-sm truncate">{app.name}</span>
        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
          <span className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            {app.branch}
          </span>
          {deployment?.commitSha && (
            <span className="font-mono text-[11px]">
              {deployment.commitSha.slice(0, 7)}
            </span>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        {deployment && (
          <Badge
            variant={getDeployBadgeVariant(deployment.status)}
            className="text-[10px] px-1.5 h-5"
          >
            {deployment.status}
          </Badge>
        )}
      </div>
    </Link>
  );
}
