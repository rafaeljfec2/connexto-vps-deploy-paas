import { useEffect, useRef } from "react";
import { Clock } from "lucide-react";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import type { Deployment } from "@/types";
import { useVisibleWindow } from "../hooks/use-visible-window";
import { DeployListRow } from "./deploy-list-row";

/** ~5 rows (py-3 + text-sm + border) — calibrated for DeployListRow height. */
const VIEWPORT_MAX_HEIGHT_CLASS = "max-h-[17.5rem]";

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
  const totalCount = deployments?.length ?? 0;
  const resetKey = `${deployments?.[0]?.id ?? ""}:${totalCount}`;

  const { visibleCount, hasMore, loadMore } = useVisibleWindow({
    totalCount,
    resetKey,
  });

  const scrollRootRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const root = scrollRootRef.current;
    const sentinel = sentinelRef.current;
    if (!root || !sentinel || !hasMore) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry?.isIntersecting) {
          loadMore();
        }
      },
      { root, rootMargin: "0px", threshold: 0 },
    );

    observer.observe(sentinel);
    return () => {
      observer.disconnect();
    };
  }, [hasMore, loadMore, visibleCount]);

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

  const visibleDeployments = deployments.slice(0, visibleCount);

  return (
    <div className="w-full min-w-0 space-y-2">
      {error && (
        <ErrorMessage
          message="Failed to refresh deployments. Showing last known list."
          variant="inline"
        />
      )}
      <div
        ref={scrollRootRef}
        className={`overflow-y-auto ${VIEWPORT_MAX_HEIGHT_CLASS}`}
        style={{ scrollbarGutter: "stable" }}
      >
        {visibleDeployments.map((deployment) => (
          <DeployListRow
            key={deployment.id}
            deployment={deployment}
            branch={branch}
            isCurrent={deployment.id === currentDeployId}
            onOpenLogs={() => onSelectDeploy(deployment.id)}
          />
        ))}
        {hasMore && (
          <div
            ref={sentinelRef}
            className="h-1 w-full shrink-0"
            aria-hidden
            data-testid="deploy-list-sentinel"
          />
        )}
      </div>
    </div>
  );
}
