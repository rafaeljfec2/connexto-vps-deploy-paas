import { GitCommit, Clock } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
import { IconText } from "@/components/icon-text";
import { ErrorMessage } from "@/components/error-message";
import { formatDate, truncateCommitSha } from "@/lib/utils";
import type { Deployment } from "@/types";

interface DeployCardProps {
  readonly deployment: Deployment;
  readonly onClick?: () => void;
}

export function DeployCard({ deployment, onClick }: DeployCardProps) {
  return (
    <Card
      className={
        onClick ? "cursor-pointer hover:bg-accent/50 transition-colors" : ""
      }
      onClick={onClick}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1 min-w-0">
            <div className="flex items-center gap-2">
              <GitCommit className="h-4 w-4 text-muted-foreground shrink-0" />
              <code className="text-sm font-mono">
                {truncateCommitSha(deployment.commitSha)}
              </code>
              <StatusBadge status={deployment.status} />
            </div>

            {deployment.commitMessage && (
              <p className="text-sm text-muted-foreground truncate">
                {deployment.commitMessage}
              </p>
            )}

            <IconText icon={Clock} className="text-xs">
              <span>{formatDate(deployment.createdAt)}</span>
              {deployment.startedAt && deployment.finishedAt && (
                <span>
                  (
                  {Math.round(
                    (new Date(deployment.finishedAt).getTime() -
                      new Date(deployment.startedAt).getTime()) /
                      1000,
                  )}
                  s)
                </span>
              )}
            </IconText>
          </div>
        </div>

        {deployment.errorMessage && (
          <ErrorMessage
            message={deployment.errorMessage}
            variant="inline"
            className="mt-2"
          />
        )}
      </CardContent>
    </Card>
  );
}
