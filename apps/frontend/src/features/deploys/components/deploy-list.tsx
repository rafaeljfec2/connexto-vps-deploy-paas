import { Clock } from "lucide-react";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import type { Deployment } from "@/types";
import { DeployListRow } from "./deploy-list-row";

interface DeployListProps {
  readonly deployments: readonly Deployment[] | undefined;
  readonly branch: string;
  readonly isLoading?: boolean;
  readonly error?: Error | null;
  readonly onSelectDeploy: (deployId: string) => void;
}

export function DeployList({
  deployments,
  branch,
  isLoading = false,
  error = null,
  onSelectDeploy,
}: DeployListProps) {
  const hasDeployments = deployments !== undefined && deployments.length > 0;

  if (isLoading && !hasDeployments) {
    return <LoadingGrid count={4} columns={1} itemHeight="h-12" />;
  }

  if (error && !hasDeployments) {
    return <ErrorMessage message="Failed to load deployments" />;
  }

  if (!hasDeployments) {
    return (
      <EmptyState
        icon={Clock}
        title="No deployments yet"
        description="Trigger a deploy to see it here."
      />
    );
  }

  const currentDeployId =
    deployments.find((deployment) => deployment.status === "success")?.id ??
    null;

  return (
    <div className="w-full min-w-0 space-y-2">
      {error && (
        <ErrorMessage
          message="Failed to refresh deployments. Showing last known list."
          variant="inline"
        />
      )}
      <div>
        {deployments.map((deployment) => (
          <DeployListRow
            key={deployment.id}
            deployment={deployment}
            branch={branch}
            isCurrent={deployment.id === currentDeployId}
            onOpenLogs={() => onSelectDeploy(deployment.id)}
          />
        ))}
      </div>
    </div>
  );
}
