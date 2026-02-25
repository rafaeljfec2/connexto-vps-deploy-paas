import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { Activity } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useDashboardStats } from "../hooks/use-dashboard-stats";
import { ActivityFeedItem } from "./activity-feed-item";

const MAX_ITEMS = 8;

export function ActivityFeed() {
  const { recentDeploys, isLoading } = useDashboardStats();

  if (isLoading) {
    return <ActivityFeedSkeleton />;
  }

  return (
    <Card className="h-fit">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-sm font-medium">
            <Activity className="h-4 w-4 text-muted-foreground" />
            Recent Activity
          </CardTitle>
          <Link
            to={ROUTES.AUDIT}
            className="text-xs text-muted-foreground/80 transition-colors hover:text-foreground"
          >
            View all
          </Link>
        </div>
      </CardHeader>
      <CardContent className="px-2 pb-3">
        {recentDeploys.length === 0 ? (
          <p className="px-2 py-6 text-center text-xs text-muted-foreground">
            No recent deployments
          </p>
        ) : (
          <div className="space-y-0.5">
            {recentDeploys.slice(0, MAX_ITEMS).map((deploy) => (
              <ActivityFeedItem
                key={`${deploy.appId}-${deploy.deployment.id}`}
                appId={deploy.appId}
                appName={deploy.appName}
                status={deploy.deployment.status}
                commitSha={deploy.deployment.commitSha}
                commitMessage={deploy.deployment.commitMessage}
                timestamp={
                  deploy.deployment.startedAt ??
                  deploy.deployment.finishedAt ??
                  null
                }
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ActivityFeedSkeleton() {
  return (
    <Card className="h-fit">
      <CardHeader className="pb-3">
        <Skeleton className="h-4 w-32" />
      </CardHeader>
      <CardContent className="space-y-3 px-4 pb-3">
        {["sk-feed-1", "sk-feed-2", "sk-feed-3", "sk-feed-4", "sk-feed-5"].map(
          (key) => (
            <div key={key} className="flex items-start gap-3">
              <Skeleton className="mt-0.5 h-4 w-4 rounded-full" />
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-3.5 w-24" />
                <Skeleton className="h-3 w-full" />
                <Skeleton className="h-2.5 w-16" />
              </div>
            </div>
          ),
        )}
      </CardContent>
    </Card>
  );
}
