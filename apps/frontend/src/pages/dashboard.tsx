import { Link } from "react-router-dom";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AppList } from "@/features/apps/components/app-list";
import { ActivityFeed } from "@/features/dashboard/components/activity-feed";
import { HeroDashboard } from "@/features/dashboard/components/hero-dashboard";
import { ServerHealthOverview } from "@/features/dashboard/components/server-health-overview";

export function DashboardPage() {
  return (
    <div className="space-y-6">
      {/* Hero Dashboard & Stats Overview */}
      <HeroDashboard />

      {/* Server Health Overview */}
      <ServerHealthOverview />

      {/* Main Grid: Applications & Activity Feed */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_340px] xl:grid-cols-[1fr_360px]">
        <div className="min-w-0 space-y-4">
          {/* Applications Header */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground">
                Applications
              </h2>
              <p className="text-sm text-muted-foreground mt-0.5">
                Manage and monitor your deployments
              </p>
            </div>
            <Button
              asChild
              className="shrink-0 bg-primary hover:bg-primary/90 text-primary-foreground font-medium shadow-sm"
            >
              <Link to="/apps/new">
                <Plus className="h-4 w-4 mr-1.5" />
                New App
              </Link>
            </Button>
          </div>

          {/* Applications Toolbar and List */}
          <AppList />
        </div>

        {/* Recent Activity Sidebar */}
        <div id="activity-feed" className="order-last scroll-mt-24">
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
}
