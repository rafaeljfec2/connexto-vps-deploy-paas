import { useMemo } from "react";
import { Link } from "react-router-dom";
import { ROUTES } from "@/constants/routes";
import { useApps } from "@/features/apps/hooks/use-apps";
import { useServers } from "@/features/servers/hooks/use-servers";
import {
  ServerHealthCard,
  ServerHealthCardSkeleton,
} from "./server-health-card";

export function ServerHealthOverview() {
  const { data: servers, isLoading: serversLoading } = useServers();
  const { data: apps } = useApps();

  const appCountByServer = useMemo(() => {
    const map = new Map<string, number>();
    for (const app of apps ?? []) {
      if (app.serverId) {
        map.set(app.serverId, (map.get(app.serverId) ?? 0) + 1);
      }
    }
    return map;
  }, [apps]);

  if (serversLoading) {
    return (
      <section className="space-y-3">
        <h2 className="text-sm font-medium text-muted-foreground">Servers</h2>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {["sk-server-1", "sk-server-2", "sk-server-3"].map((key) => (
            <ServerHealthCardSkeleton key={key} />
          ))}
        </div>
      </section>
    );
  }

  if (!servers || servers.length === 0) return null;

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-muted-foreground">Servers</h2>
        <Link
          to={ROUTES.SERVERS}
          className="text-xs text-muted-foreground/80 transition-colors hover:text-foreground"
        >
          View all
        </Link>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {servers.map((server) => (
          <ServerHealthCard
            key={server.id}
            server={server}
            appCount={appCountByServer.get(server.id) ?? 0}
          />
        ))}
      </div>
    </section>
  );
}
