import { Activity } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { AuditList } from "@/features/audit";

export function AuditPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Logs"
        description="Track all platform events and activities"
        icon={Activity}
      />
      <AuditList />
    </div>
  );
}
