import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { GitBranch, ExternalLink, RefreshCw, RotateCcw } from "lucide-react";
import { useApp } from "@/features/apps/hooks/use-apps";
import { formatRepositoryUrl } from "@/lib/utils";
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/status-badge";
import { PageHeader } from "@/components/page-header";
import { IconText } from "@/components/icon-text";

export function AppDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const { data: app, isLoading: appLoading } = useApp(id ?? "");
  const { data: deployments } = useDeploys(id ?? "");
  const redeploy = useRedeploy();
  const rollback = useRollback();

  const [selectedDeployId, setSelectedDeployId] = useState<string | null>(null);

  const latestDeploy = deployments?.[0];
  const selectedDeploy = selectedDeployId
    ? deployments?.find((d) => d.id === selectedDeployId)
    : latestDeploy;

  const handleRedeploy = () => {
    if (id) {
      redeploy.mutate({ appId: id });
    }
  };

  const handleRollback = () => {
    if (id) {
      rollback.mutate(id);
    }
  };

  if (appLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-96" />
          <Skeleton className="h-96" />
        </div>
      </div>
    );
  }

  if (!app) {
    return (
      <div className="text-center py-12">
        <p className="text-destructive">Application not found</p>
        <Button asChild variant="link">
          <Link to="/">Go back to dashboard</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title={app.name}
        titleSuffix={
          latestDeploy && <StatusBadge status={latestDeploy.status} />
        }
        description={
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <IconText icon={GitBranch} as="span">
              {app.branch}
            </IconText>
            <a
              href={app.repositoryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 hover:text-foreground"
            >
              <ExternalLink className="h-4 w-4" />
              {formatRepositoryUrl(app.repositoryUrl)}
            </a>
          </div>
        }
        actions={
          <>
            <Button
              variant="outline"
              onClick={handleRollback}
              disabled={
                rollback.isPending ||
                !deployments?.some((d) => d.status === "success")
              }
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Rollback
            </Button>
            <Button onClick={handleRedeploy} disabled={redeploy.isPending}>
              <RefreshCw
                className={`h-4 w-4 mr-2 ${redeploy.isPending ? "animate-spin" : ""}`}
              />
              Redeploy
            </Button>
          </>
        }
      />

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Deployments</CardTitle>
          </CardHeader>
          <CardContent>
            <DeployTimeline
              appId={app.id}
              onSelectDeploy={setSelectedDeployId}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              Logs
              {selectedDeploy && (
                <span className="text-sm font-normal text-muted-foreground ml-2">
                  ({selectedDeploy.commitSha.slice(0, 7)})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <LogViewer logs={selectedDeploy?.logs ?? null} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
