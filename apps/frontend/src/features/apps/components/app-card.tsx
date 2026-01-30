import { Link } from "react-router-dom";
import { Clock, ExternalLink, Folder, GitBranch } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { IconText } from "@/components/icon-text";
import { StatusBadge } from "@/components/status-badge";
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

          {app.workdir && app.workdir !== "." && (
            <IconText icon={Folder}>
              <span className="truncate font-mono text-xs">{app.workdir}</span>
            </IconText>
          )}

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
