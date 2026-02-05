import { HardDrive } from "lucide-react";
import { PageHeader } from "@/components/page-header";
import { ImageList } from "@/features/images";

export function ImagesPage() {
  return (
    <div className="container mx-auto max-w-[1920px] py-6 space-y-6">
      <PageHeader
        title="Docker Images"
        description="Manage Docker images on this server"
        icon={HardDrive}
      />
      <ImageList />
    </div>
  );
}
