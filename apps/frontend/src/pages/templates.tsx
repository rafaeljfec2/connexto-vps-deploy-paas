import { useState } from "react";
import { FileCode } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { ServerSelector } from "@/components/server-selector";
import { TemplateList } from "@/features/templates";

export function TemplatesPage() {
  const [serverId, setServerId] = useState<string | undefined>();

  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/containers"
        title="Application Templates"
        description="Deploy pre-configured applications from templates."
        icon={FileCode}
        actions={<ServerSelector value={serverId} onChange={setServerId} />}
      />
      <TemplateList serverId={serverId} />
    </div>
  );
}
