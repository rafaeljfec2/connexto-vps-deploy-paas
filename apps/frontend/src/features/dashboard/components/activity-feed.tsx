import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
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

  const items = recentDeploys.slice(0, MAX_ITEMS);

  return (
    <Card className="h-fit rounded-xl border border-border/60 bg-card/40 shadow-sm backdrop-blur">
      <CardHeader className="pb-3 pt-4 px-4">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-semibold text-foreground">
            Recent activity
          </CardTitle>
          <Link
            to={ROUTES.AUDIT}
            className="text-xs font-medium text-primary hover:text-primary/80 transition-colors"
          >
            View all
          </Link>
        </div>
      </CardHeader>
      <CardContent className="px-2 pb-3">
        {items.length === 0 ? (
          <p className="px-2 py-6 text-center text-xs text-muted-foreground">
            No recent deployments
          </p>
        ) : (
          <div className="space-y-0.5">
            {items.map((deploy, index) => (
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
                isLast={index === items.length - 1}
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
    <Card className="h-fit rounded-xl border border-border/60 bg-card/40">
      <CardHeader className="pb-3 pt-4 px-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3 w-12" />
        </div>
      </CardHeader>
      <CardContent className="space-y-3 px-3 pb-3">
        {["sk-feed-1", "sk-feed-2", "sk-feed-3", "sk-feed-4", "sk-feed-5"].map(
          (key) => (
            <div key={key} className="flex items-start gap-3 px-2 py-1.5">
              <Skeleton className="mt-0.5 h-4 w-4 rounded-full" />
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-3.5 w-28" />
                <Skeleton className="h-3 w-full" />
                <Skeleton className="h-2.5 w-20" />
              </div>
            </div>
          ),
        )}
      </CardContent>
    </Card>
  );
}
