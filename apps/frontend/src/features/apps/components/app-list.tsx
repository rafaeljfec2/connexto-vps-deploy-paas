import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Box, Grid3X3, List, Plus, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { LoadingGrid } from "@/components/loading-grid";
import { useServers } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";
import { useApps } from "../hooks/use-apps";
import { AppCard } from "./app-card";
import { AppRow } from "./app-row";

type ViewMode = "grid" | "list";

const VIEW_MODE_KEY = "flowdeploy:app-list-view";
const SERVER_FILTER_KEY = "flowdeploy:app-list-server";
const ALL_SERVERS = "all";
const LOCAL_SERVER = "local";

export function AppList() {
  const { data: apps, isLoading, error } = useApps();
  const { data: servers } = useServers();

  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    const saved = localStorage.getItem(VIEW_MODE_KEY);
    return saved === "list" ? "list" : "grid";
  });

  const [serverFilter, setServerFilter] = useState<string>(() => {
    return localStorage.getItem(SERVER_FILTER_KEY) ?? ALL_SERVERS;
  });

  useEffect(() => {
    localStorage.setItem(VIEW_MODE_KEY, viewMode);
  }, [viewMode]);

  useEffect(() => {
    localStorage.setItem(SERVER_FILTER_KEY, serverFilter);
  }, [serverFilter]);

  const serverMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const server of servers ?? []) {
      map.set(server.id, server.name);
    }
    return map;
  }, [servers]);

  const filteredApps = useMemo(() => {
    if (!apps) return [];
    if (serverFilter === ALL_SERVERS) return apps;
    if (serverFilter === LOCAL_SERVER) {
      return apps.filter((app) => !app.serverId);
    }
    return apps.filter((app) => app.serverId === serverFilter);
  }, [apps, serverFilter]);

  const getServerName = (serverId?: string): string | undefined => {
    if (!serverId) return undefined;
    return serverMap.get(serverId);
  };

  if (isLoading) {
    return <LoadingGrid count={6} columns={3} />;
  }

  if (error) {
    return <ErrorMessage message="Failed to load applications" />;
  }

  if (!apps || apps.length === 0) {
    return (
      <EmptyState
        icon={Box}
        title="No applications yet"
        description="Connect your first GitHub repository to start deploying."
        action={
          <Button asChild>
            <Link to="/apps/new">
              <Plus className="h-4 w-4 mr-2" />
              Connect Repository
            </Link>
          </Button>
        }
      />
    );
  }

  const hasRemoteApps = apps.some((app) => app.serverId);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        {hasRemoteApps && servers && servers.length > 0 ? (
          <Select value={serverFilter} onValueChange={setServerFilter}>
            <SelectTrigger className="w-[200px] h-8 text-sm">
              <Server className="h-3.5 w-3.5 mr-2 text-muted-foreground" />
              <SelectValue placeholder="All servers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_SERVERS}>All servers</SelectItem>
              <SelectItem value={LOCAL_SERVER}>Local (self-hosted)</SelectItem>
              {servers.map((server) => (
                <SelectItem key={server.id} value={server.id}>
                  {server.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <div />
        )}

        <div className="flex items-center rounded-md border bg-muted p-1">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 w-7 p-0",
              viewMode === "grid" && "bg-background shadow-sm",
            )}
            onClick={() => setViewMode("grid")}
          >
            <Grid3X3 className="h-4 w-4" />
            <span className="sr-only">Grid view</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 w-7 p-0",
              viewMode === "list" && "bg-background shadow-sm",
            )}
            onClick={() => setViewMode("list")}
          >
            <List className="h-4 w-4" />
            <span className="sr-only">List view</span>
          </Button>
        </div>
      </div>

      {filteredApps.length === 0 ? (
        <EmptyState
          icon={Server}
          title="No applications on this server"
          description="There are no applications deployed to the selected server."
        />
      ) : viewMode === "grid" ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredApps.map((app) => (
            <AppCard
              key={app.id}
              app={app}
              serverName={getServerName(app.serverId)}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {filteredApps.map((app) => (
            <AppRow
              key={app.id}
              app={app}
              serverName={getServerName(app.serverId)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
