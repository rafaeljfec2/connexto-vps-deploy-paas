import { useState } from "react";
import { GitCommit, MoreHorizontal, Rocket } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DeployList } from "@/features/deploys/components/deploy-list";
import { DeployLogsModal } from "@/features/deploys/components/deploy-logs-modal";
import { getDeployStatusPresentation } from "@/features/deploys/utils/deploy-status-label";
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
  readonly deploymentsLoading?: boolean;
  readonly deploymentsError?: Error | null;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly actions: UseAppActionsReturn;
  readonly selectedDeploy: Deployment | undefined;
  readonly onSelectDeploy: (id: string | null) => void;
}

export function DeploymentsSection({
  app,
  deployments,
  deploymentsLoading = false,
  deploymentsError = null,
  expanded,
  onToggle,
  actions,
  selectedDeploy,
  onSelectDeploy,
}: DeploymentsSectionProps) {
  const [commitsOpen, setCommitsOpen] = useState(false);
  const successfulDeploys =
    deployments?.filter((d) => d.status === "success").length ?? 0;
  const totalDeploys = deployments?.length ?? 0;
  const latestDeploy = deployments?.[0];

  const handleRedeployFromCommit = (sha: string) => {
    setCommitsOpen(false);
    actions.handleRedeploy(sha);
  };

  return (
    <>
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
                • Latest:{" "}
                <Badge variant="outline" className="text-xs px-1.5 py-0">
                  {getDeployStatusPresentation(latestDeploy.status).label}
                </Badge>
              </>
            )}
          </span>
        }
        actions={
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                aria-label="Deployment section actions"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => setCommitsOpen(true)}>
                <GitCommit className="mr-2 h-4 w-4" />
                Redeploy from commit
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        }
      >
        <DeployList
          deployments={deployments}
          branch={app.branch}
          isLoading={deploymentsLoading}
          error={deploymentsError}
          onSelectDeploy={onSelectDeploy}
        />
      </CollapsibleSection>

      <DeployLogsModal
        deployment={selectedDeploy}
        open={selectedDeploy !== undefined}
        onOpenChange={(open) => {
          if (!open) onSelectDeploy(null);
        }}
      />

      <Dialog open={commitsOpen} onOpenChange={setCommitsOpen}>
        <DialogContent className="max-w-lg max-h-[85vh] flex flex-col gap-0 overflow-hidden p-0">
          <DialogHeader className="shrink-0 border-b border-border px-4 py-3">
            <DialogTitle className="flex items-center gap-2 text-base">
              <GitCommit className="h-4 w-4" />
              Redeploy from commit
            </DialogTitle>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            <CommitSelectorInline
              appId={app.id}
              onSelect={handleRedeployFromCommit}
              disabled={actions.redeploy.isPending}
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
