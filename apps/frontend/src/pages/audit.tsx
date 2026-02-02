import { Activity } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { AuditList } from "@/features/audit";

export function AuditPage() {
  return (
    <div className="container mx-auto py-6 space-y-6">
      <PageHeader
        title="Audit Logs"
        description="Track all platform events and activities"
        icon={Activity}
      />
      <AuditList />
    </div>
  );
}
