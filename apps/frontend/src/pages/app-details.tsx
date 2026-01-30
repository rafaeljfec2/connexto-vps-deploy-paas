import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  CheckCircle,
  ChevronDown,
  ChevronRight,
  Cpu,
  ExternalLink,
  Folder,
  GitBranch,
  Globe,
  HardDrive,
  HeartPulse,
  Link2,
  Link2Off,
  Network,
  Play,
  RefreshCw,
  RotateCcw,
  Square,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { HealthDetail, HealthIndicator } from "@/components/health-indicator";
import { IconText } from "@/components/icon-text";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { AppSettingsDialog } from "@/features/apps/components/app-settings-dialog";
import { EnvVarsManager } from "@/features/apps/components/env-vars-manager";
import {
  useApp,
  useAppConfig,
  useAppURL,
  useRemoveWebhook,
  useRestartContainer,
  useSetupWebhook,
  useStartContainer,
  useStopContainer,
  useWebhookStatus,
} from "@/features/apps/hooks/use-apps";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import {
  useDeploys,
  useRedeploy,
  useRollback,
} from "@/features/deploys/hooks/use-deploys";
import { useAppHealth } from "@/hooks/use-sse";
import { formatRepositoryUrl } from "@/lib/utils";

export function AppDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const { data: app, isLoading: appLoading } = useApp(id ?? "");
  const { data: deployments } = useDeploys(id ?? "");
  const { data: webhookStatus } = useWebhookStatus(id ?? "");
  const { data: health } = useAppHealth(id);
  const { data: appUrl } = useAppURL(id);
  const { data: appConfig } = useAppConfig(id);
  const redeploy = useRedeploy();
  const rollback = useRollback();
  const setupWebhook = useSetupWebhook();
  const removeWebhook = useRemoveWebhook();
  const restartContainer = useRestartContainer();
  const stopContainer = useStopContainer();
  const startContainer = useStartContainer();

  const [selectedDeployId, setSelectedDeployId] = useState<string | null>(null);
  const [showCommitInput, setShowCommitInput] = useState(false);
  const [commitSha, setCommitSha] = useState("");
  const [configExpanded, setConfigExpanded] = useState(false);

  const latestDeploy = deployments?.[0];
  const selectedDeploy = selectedDeployId
    ? deployments?.find((d) => d.id === selectedDeployId)
    : latestDeploy;

  const handleRedeploy = (sha?: string) => {
    if (id) {
      redeploy.mutate({ appId: id, commitSha: sha });
      setShowCommitInput(false);
      setCommitSha("");
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
          <div className="flex items-center gap-2">
            <HealthIndicator health={health} />
            {latestDeploy && <StatusBadge status={latestDeploy.status} />}
          </div>
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
            {appUrl?.url && (
              <Button variant="outline" asChild>
                <a href={appUrl.url} target="_blank" rel="noopener noreferrer">
                  <Globe className="h-4 w-4 mr-2" />
                  Open App
                </a>
              </Button>
            )}
            <AppSettingsDialog app={app} />
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
            <Button
              onClick={() => handleRedeploy()}
              disabled={redeploy.isPending}
            >
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
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Deployments</CardTitle>
            <div className="flex items-center gap-2">
              {showCommitInput ? (
                <>
                  <Input
                    placeholder="Commit SHA"
                    value={commitSha}
                    onChange={(e) => setCommitSha(e.target.value)}
                    className="w-40 h-8 text-xs font-mono"
                  />
                  <Button
                    size="sm"
                    onClick={() => handleRedeploy(commitSha)}
                    disabled={!commitSha || redeploy.isPending}
                  >
                    Deploy
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      setShowCommitInput(false);
                      setCommitSha("");
                    }}
                  >
                    Cancel
                  </Button>
                </>
              ) : (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setShowCommitInput(true)}
                >
                  Deploy Commit
                </Button>
              )}
            </div>
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
            <LogViewer
              logs={selectedDeploy?.logs ?? null}
              title={
                selectedDeploy
                  ? `Logs (${selectedDeploy.commitSha.slice(0, 7)})`
                  : "Logs"
              }
            />
          </CardContent>
        </Card>
      </div>

      <EnvVarsManager appId={app.id} />

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Container Health</CardTitle>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => id && restartContainer.mutate(id)}
              disabled={
                restartContainer.isPending || health?.status === "not_found"
              }
            >
              <RefreshCw
                className={`h-4 w-4 mr-1 ${restartContainer.isPending ? "animate-spin" : ""}`}
              />
              Restart
            </Button>
            {health?.status === "running" ? (
              <Button
                variant="outline"
                size="sm"
                onClick={() => id && stopContainer.mutate(id)}
                disabled={stopContainer.isPending}
              >
                <Square className="h-4 w-4 mr-1" />
                Stop
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                onClick={() => id && startContainer.mutate(id)}
                disabled={
                  startContainer.isPending || health?.status === "not_found"
                }
              >
                <Play className="h-4 w-4 mr-1" />
                Start
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <HealthDetail health={health} />
        </CardContent>
      </Card>

      {appConfig && (
        <Card>
          <CardHeader
            className="cursor-pointer select-none"
            onClick={() => setConfigExpanded(!configExpanded)}
          >
            <div className="flex items-center gap-2">
              {configExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              <CardTitle>Deployment Config</CardTitle>
            </div>
          </CardHeader>
          {configExpanded && (
            <CardContent>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <Network className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">Port:</span>
                    <span className="font-mono">
                      {appConfig.hostPort === appConfig.port
                        ? appConfig.port
                        : `${appConfig.hostPort}:${appConfig.port}`}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <HeartPulse className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">Health Check:</span>
                    <span className="font-mono">
                      {appConfig.healthcheck.path}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="ml-6">
                      Interval: {appConfig.healthcheck.interval} | Timeout:{" "}
                      {appConfig.healthcheck.timeout} | Retries:{" "}
                      {appConfig.healthcheck.retries}
                    </span>
                  </div>
                </div>
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <HardDrive className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">Memory:</span>
                    <span className="font-mono">
                      {appConfig.resources.memory}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Cpu className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">CPU:</span>
                    <span className="font-mono">{appConfig.resources.cpu}</span>
                  </div>
                  {appConfig.domains.length > 0 && (
                    <div className="flex items-start gap-2 text-sm">
                      <Globe className="h-4 w-4 text-muted-foreground mt-0.5" />
                      <div>
                        <span className="font-medium">Domains:</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {appConfig.domains.map((domain) => (
                            <span
                              key={domain}
                              className="px-2 py-0.5 bg-muted rounded text-xs font-mono"
                            >
                              {domain}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          )}
        </Card>
      )}

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
