import { useState } from "react";
import { HardDrive } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { ServerSelector } from "@/components/server-selector";
import { ImageList } from "@/features/images";

export function ImagesPage() {
  const [serverId, setServerId] = useState<string | undefined>();

  return (
    <div className="py-6 space-y-6">
      <PageHeader
        title="Docker Images"
        description="Manage Docker images on this server"
        icon={HardDrive}
        actions={<ServerSelector value={serverId} onChange={setServerId} />}
      />
      <ImageList serverId={serverId} />
    </div>
  );
}
