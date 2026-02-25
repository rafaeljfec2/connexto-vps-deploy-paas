import {
  ChevronsDownUp,
  ChevronsUpDown,
  Globe,
  RefreshCw,
  RotateCcw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { AppSettingsDialog } from "@/features/apps/components/app-settings-dialog";
import type { useAppActions } from "@/features/apps/hooks/use-app-actions";

interface HeaderActionsProps {
  readonly allExpanded: boolean;
  readonly toggleAllSections: () => void;
  readonly openAppUrl: string | null;
  readonly app: {
    id: string;
    name: string;
    repositoryUrl: string;
    branch: string;
    workdir: string;
    webhookId: number | null;
  };
  readonly actions: ReturnType<typeof useAppActions>;
  readonly hasSuccessfulDeploy: boolean;
}

export function HeaderActions({
  allExpanded,
  toggleAllSections,
  openAppUrl,
  app,
  actions,
  hasSuccessfulDeploy,
}: HeaderActionsProps) {
  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={toggleAllSections}
        className="hidden sm:inline-flex"
      >
        {allExpanded ? (
          <ChevronsDownUp className="h-4 w-4" />
        ) : (
          <ChevronsUpDown className="h-4 w-4" />
        )}
        <span className="hidden lg:inline ml-2">
          {allExpanded ? "Collapse All" : "Expand All"}
        </span>
      </Button>
      {openAppUrl && (
        <Button variant="outline" size="sm" asChild>
          <a href={openAppUrl} target="_blank" rel="noopener noreferrer">
            <Globe className="h-4 w-4" />
            <span className="hidden lg:inline ml-2">Open App</span>
          </a>
        </Button>
      )}
      <AppSettingsDialog app={app} />
      <Button
        variant="outline"
        size="sm"
        onClick={actions.handleRollback}
        disabled={actions.rollback.isPending || !hasSuccessfulDeploy}
        className="hidden md:inline-flex"
      >
        <RotateCcw className="h-4 w-4" />
        <span className="hidden lg:inline ml-2">Rollback</span>
      </Button>
      <Button
        size="sm"
        onClick={() => actions.handleRedeploy()}
        disabled={actions.redeploy.isPending}
        className="hidden md:inline-flex"
      >
        <RefreshCw
          className={`h-4 w-4 ${actions.redeploy.isPending ? "animate-spin" : ""}`}
        />
        <span className="hidden lg:inline ml-2">Redeploy</span>
      </Button>
    </>
  );
}
