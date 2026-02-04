import { Clock } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import { useDeploys } from "../hooks/use-deploys";
import { DeployCard } from "./deploy-card";

interface DeployTimelineProps {
  readonly appId: string;
  readonly onSelectDeploy?: (deployId: string) => void;
}

export function DeployTimeline({ appId, onSelectDeploy }: DeployTimelineProps) {
  const { data: deployments, isLoading, error } = useDeploys(appId);

  if (isLoading) {
    return <LoadingGrid count={3} columns={1} itemHeight="h-24" />;
  }

  if (error) {
    return <ErrorMessage message="Failed to load deployments" />;
  }

  if (!deployments || deployments.length === 0) {
    return (
      <EmptyState
        icon={Clock}
        title="No deployments yet"
        description="Trigger a deploy to see it here."
      />
    );
  }

  const currentDeployId =
    deployments.find((d) => d.status === "success")?.id ?? null;

  return (
    <ScrollArea className="h-[450px] w-full">
      <div className="space-y-3 pr-3 max-w-full">
        {deployments.map((deployment) => (
          <DeployCard
            key={deployment.id}
            deployment={deployment}
            isCurrent={deployment.id === currentDeployId}
            onClick={
              onSelectDeploy ? () => onSelectDeploy(deployment.id) : undefined
            }
          />
        ))}
      </div>
    </ScrollArea>
  );
}
