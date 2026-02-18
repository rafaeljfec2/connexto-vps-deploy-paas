import { FileText, GitCommit, History, Rocket } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StatusBadge } from "@/components/status-badge";
import { DeployTimeline } from "@/features/deploys/components/deploy-timeline";
import { LogViewer } from "@/features/deploys/components/log-viewer";
import type { App, Deployment } from "@/types";
import { CollapsibleSection } from "./collapsible-section";
import { CommitSelectorInline } from "./commit-selector";

export interface UseAppActionsReturn {
  readonly handleRedeploy: (sha?: string) => void;
  readonly redeploy: { isPending: boolean };
}

interface DeploymentsSectionProps {
  readonly app: App;
  readonly deployments: readonly Deployment[] | undefined;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly actions: UseAppActionsReturn;
  readonly selectedDeploy: Deployment | undefined;
  readonly onSelectDeploy: (id: string | null) => void;
}

export function DeploymentsSection({
  app,
  deployments,
  expanded,
  onToggle,
  actions,
  selectedDeploy,
  onSelectDeploy,
}: DeploymentsSectionProps) {
  const successfulDeploys =
    deployments?.filter((d) => d.status === "success").length ?? 0;
  const totalDeploys = deployments?.length ?? 0;
  const latestDeploy = deployments?.[0];

  return (
    <CollapsibleSection
      title="Deployments"
      icon={Rocket}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span>
          {totalDeploys} deploys ({successfulDeploys} successful)
          {latestDeploy && (
            <>
              {" "}
              • Latest: <StatusBadge status={latestDeploy.status} size="sm" />
            </>
          )}
        </span>
      }
    >
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 overflow-hidden">
        <Tabs defaultValue="history" className="w-full min-w-0 overflow-hidden">
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger
              value="history"
              className="gap-1 sm:gap-2 text-xs sm:text-sm"
            >
              <History className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
              <span className="hidden xs:inline">Deploy</span> History
            </TabsTrigger>
            <TabsTrigger
              value="commits"
              className="gap-1 sm:gap-2 text-xs sm:text-sm"
            >
              <GitCommit className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
              Commits
            </TabsTrigger>
          </TabsList>
          <TabsContent value="history" className="mt-0">
            <DeployTimeline appId={app.id} onSelectDeploy={onSelectDeploy} />
          </TabsContent>
          <TabsContent value="commits" className="mt-0">
            <CommitSelectorInline
              appId={app.id}
              onSelect={actions.handleRedeploy}
              disabled={actions.redeploy.isPending}
            />
          </TabsContent>
        </Tabs>
        <div className="space-y-4 min-w-0 overflow-hidden">
          <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
            <FileText className="h-4 w-4" />
            Deploy Logs
            {selectedDeploy && (
              <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                {selectedDeploy.commitSha.slice(0, 7)}
              </span>
            )}
          </div>
          <LogViewer
            logs={selectedDeploy?.logs ?? null}
            title={
              selectedDeploy
                ? `Logs (${selectedDeploy.commitSha.slice(0, 7)})`
                : "Logs"
            }
          />
        </div>
      </div>
    </CollapsibleSection>
  );
}
