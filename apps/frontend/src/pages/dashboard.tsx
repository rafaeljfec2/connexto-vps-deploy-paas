import { AppList } from "@/features/apps/components/app-list";
import { ActivityFeed } from "@/features/dashboard/components/activity-feed";
import { HeroDashboard } from "@/features/dashboard/components/hero-dashboard";
import { ServerHealthOverview } from "@/features/dashboard/components/server-health-overview";

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <HeroDashboard />
      <ServerHealthOverview />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_340px]">
        <div className="min-w-0 space-y-4">
          <h2 className="text-sm font-medium text-muted-foreground">
            Applications
          </h2>
          <AppList />
        </div>
        <div
          id="activity-feed"
          className="order-first scroll-mt-24 lg:order-last"
        >
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
}
