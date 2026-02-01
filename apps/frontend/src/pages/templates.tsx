import { PageHeader } from "@/components/page-header";
import { TemplateList } from "@/features/templates";

export function TemplatesPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/containers"
        title="Application Templates"
        description="Deploy pre-configured applications from templates."
      />
      <TemplateList />
    </div>
  );
}
