import { PageHeader } from "@/components/page-header";
import { CloudflareConnection } from "@/features/settings/components/cloudflare-connection";

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title="Settings"
        description="Manage your FlowDeploy integrations and preferences"
      />

      <div className="space-y-6">
        <section>
          <h2 className="text-lg font-semibold mb-4">Integrations</h2>
          <CloudflareConnection />
        </section>
      </div>
    </div>
  );
}
