import { PageHeader } from "@/components/page-header";
import { AddServerDialog } from "@/features/servers/components/add-server-dialog";
import { ServerList } from "@/features/servers/components/server-list";

export function ServersPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        backTo="/"
        title="Remote Servers"
        description="Manage servers for remote deploy"
        actions={<AddServerDialog />}
      />

      <section aria-labelledby="servers-heading">
        <h2 id="servers-heading" className="sr-only">
          Servers
        </h2>
        <ServerList />
      </section>
    </div>
  );
}
