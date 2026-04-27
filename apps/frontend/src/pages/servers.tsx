import { ServerList } from "@/features/servers/components/server-list";
import { ServersHero } from "@/features/servers/components/servers-hero";
import { useServers } from "@/features/servers/hooks/use-servers";

export function ServersPage() {
  const { data: servers, isLoading } = useServers();

  return (
    <div className="space-y-6">
      <ServersHero servers={servers} isLoading={isLoading} />

      <section aria-labelledby="servers-heading">
        <h2 id="servers-heading" className="sr-only">
          Servers
        </h2>
        <ServerList />
      </section>
    </div>
  );
}
