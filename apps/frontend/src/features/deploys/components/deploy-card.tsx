import { Clock, GitCommit, Tag } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ErrorMessage } from "@/components/error-message";
import { IconText } from "@/components/icon-text";
import { StatusBadge } from "@/components/status-badge";
import { formatDate, formatDuration, truncateCommitSha } from "@/lib/utils";
import type { Deployment } from "@/types";

interface DeployCardProps {
  readonly deployment: Deployment;
  readonly isCurrent?: boolean;
  readonly onClick?: () => void;
}

export function DeployCard({
  deployment,
  isCurrent = false,
  onClick,
}: Readonly<DeployCardProps>) {
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
            <div className="flex items-center gap-2 flex-wrap">
              <GitCommit className="h-4 w-4 text-muted-foreground shrink-0" />
              <code className="text-sm font-mono">
                {truncateCommitSha(deployment.commitSha)}
              </code>
              <StatusBadge status={deployment.status} />
              {deployment.appVersion && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-5 font-mono"
                >
                  <Tag className="h-3 w-3 mr-0.5" />v{deployment.appVersion}
                </Badge>
              )}
              {isCurrent && (
                <Badge variant="secondary" className="text-xs">
                  Current
                </Badge>
              )}
            </div>

            {deployment.commitMessage && (
              <p className="text-sm text-muted-foreground truncate max-w-[300px]">
                {deployment.commitMessage}
              </p>
            )}

            <IconText icon={Clock} className="text-xs">
              <span>{formatDate(deployment.createdAt)}</span>
              {deployment.startedAt && deployment.finishedAt && (
                <span>
                  {" "}
                  (
                  {formatDuration(
                    new Date(deployment.finishedAt).getTime() -
                      new Date(deployment.startedAt).getTime(),
                  )}
                  )
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
