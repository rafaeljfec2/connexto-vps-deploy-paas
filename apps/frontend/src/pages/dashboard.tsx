import { AppList } from "@/features/apps/components/app-list";
import { ActivityFeed } from "@/features/dashboard/components/activity-feed";
import { GreetingSection } from "@/features/dashboard/components/greeting-section";
import { ServerHealthOverview } from "@/features/dashboard/components/server-health-overview";
import { StatsOverview } from "@/features/dashboard/components/stats-overview";

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <GreetingSection />
      <StatsOverview />
      <ServerHealthOverview />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_340px]">
        <div className="min-w-0 space-y-4">
          <h2 className="text-sm font-medium text-muted-foreground">
            Applications
          </h2>
          <AppList />
        </div>
        <div className="order-first lg:order-last">
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
}
