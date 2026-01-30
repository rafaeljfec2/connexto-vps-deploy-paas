import { Link } from "react-router-dom";
import { GitBranch, Clock, ExternalLink } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
import { IconText } from "@/components/icon-text";
import { formatRelativeTime, formatRepositoryUrl } from "@/lib/utils";
import type { App, Deployment } from "@/types";

interface AppCardProps {
  readonly app: App;
  readonly latestDeploy?: Deployment;
}

export function AppCard({ app, latestDeploy }: AppCardProps) {
  return (
    <Link to={`/apps/${app.id}`}>
      <Card className="hover:bg-accent/50 transition-colors cursor-pointer">
        <CardHeader className="pb-2">
          <div className="flex items-start justify-between">
            <CardTitle className="text-lg">{app.name}</CardTitle>
            {latestDeploy && <StatusBadge status={latestDeploy.status} />}
          </div>
        </CardHeader>
        <CardContent className="space-y-2">
          <IconText icon={GitBranch}>
            <span>{app.branch}</span>
          </IconText>

          <IconText icon={ExternalLink}>
            <span className="truncate">
              {formatRepositoryUrl(app.repositoryUrl)}
            </span>
          </IconText>

          {app.lastDeployedAt && (
            <IconText icon={Clock}>
              <span>Deployed {formatRelativeTime(app.lastDeployedAt)}</span>
            </IconText>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}
