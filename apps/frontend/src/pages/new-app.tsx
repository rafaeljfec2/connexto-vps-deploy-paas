import { AppForm } from "@/features/apps/components/app-form";
import { PageHeader } from "@/components/page-header";

export function NewAppPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title="New Application"
        description="Connect a GitHub repository for automatic deployments."
      />
      <AppForm />
    </div>
  );
}
