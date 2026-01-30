import { PageHeader } from "@/components/page-header";
import { AppList } from "@/features/apps/components/app-list";

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Applications"
        description="Manage and monitor your deployed applications."
      />
      <AppList />
    </div>
  );
}
