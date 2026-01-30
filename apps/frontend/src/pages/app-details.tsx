import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  CheckCircle,
  ExternalLink,
  Folder,
  GitBranch,
  Link2,
  Link2Off,
  RefreshCw,
  RotateCcw,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { IconText } from "@/components/icon-text";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import {
  useApp,
  useRemoveWebhook,
  useSetupWebhook,
  useWebhookStatus,
} from "@/features/apps/hooks/use-apps";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { formatRepositoryUrl } from "@/lib/utils";

export function AppDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const { data: app, isLoading: appLoading } = useApp(id ?? "");
  const { data: deployments } = useDeploys(id ?? "");
  const { data: webhookStatus } = useWebhookStatus(id ?? "");
  const redeploy = useRedeploy();
  const rollback = useRollback();
  const setupWebhook = useSetupWebhook();
  const removeWebhook = useRemoveWebhook();

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

  const handleSetupWebhook = () => {
    if (id) {
      setupWebhook.mutate(id);
    }
  };

  const handleRemoveWebhook = () => {
    if (id) {
      removeWebhook.mutate(id);
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
          <div className="flex items-center gap-4 text-sm text-muted-foreground flex-wrap">
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
            {app.workdir && app.workdir !== "." && (
              <IconText icon={Folder} as="span">
                <span className="font-mono text-xs">{app.workdir}</span>
              </IconText>
            )}
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

      <Card>
        <CardHeader>
          <CardTitle>GitHub Webhook</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {app.webhookId ? (
                <>
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <div>
                    <p className="font-medium">Webhook configured</p>
                    <p className="text-sm text-muted-foreground">
                      Auto-deploy enabled for push events
                      {webhookStatus?.active === false && (
                        <span className="text-yellow-500 ml-2">(inactive)</span>
                      )}
                    </p>
                  </div>
                </>
              ) : (
                <>
                  <XCircle className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="font-medium">Webhook not configured</p>
                    <p className="text-sm text-muted-foreground">
                      Configure to enable auto-deploy on push
                    </p>
                  </div>
                </>
              )}
            </div>
            {app.webhookId ? (
              <Button
                variant="outline"
                onClick={handleRemoveWebhook}
                disabled={removeWebhook.isPending}
              >
                <Link2Off className="h-4 w-4 mr-2" />
                Remove Webhook
              </Button>
            ) : (
              <Button
                onClick={handleSetupWebhook}
                disabled={setupWebhook.isPending}
              >
                <Link2 className="h-4 w-4 mr-2" />
                Setup Webhook
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
