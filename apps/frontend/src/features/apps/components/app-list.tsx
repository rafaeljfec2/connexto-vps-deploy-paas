import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "@/contexts/auth-context";
import { Box, Grid3X3, List, Plus, Search, Server, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import { extractRepoOwner } from "../utils/org-helpers";
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
  const { isAdmin } = useAuth();

  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    const saved = localStorage.getItem(VIEW_MODE_KEY);
    return saved === "grid" ? "grid" : "list";
  });

  const [serverFilter, setServerFilter] = useState<string>(() => {
    return localStorage.getItem(SERVER_FILTER_KEY) ?? ALL_SERVERS;
  });

  const [searchQuery, setSearchQuery] = useState("");

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

  const visibleApps = useMemo(() => {
    if (!apps) return [];
    if (isAdmin) return apps;
    return apps.filter((app) => app.serverId);
  }, [apps, isAdmin]);

  const filteredApps = useMemo(() => {
    let list = visibleApps;

    // Filter by server
    if (serverFilter === LOCAL_SERVER) {
      list = list.filter((app) => !app.serverId);
    } else if (serverFilter !== ALL_SERVERS) {
      list = list.filter((app) => app.serverId === serverFilter);
    }

    // Filter by search query
    const query = searchQuery.trim().toLowerCase();
    if (query !== "") {
      list = list.filter((app) => {
        const nameMatch = app.name.toLowerCase().includes(query);
        const branchMatch = app.branch.toLowerCase().includes(query);
        const repoMatch = app.repositoryUrl.toLowerCase().includes(query);
        const runtimeMatch =
          app.runtime?.toLowerCase().includes(query) ?? false;
        const versionMatch =
          app.appVersion?.toLowerCase().includes(query) ?? false;
        const workdirMatch =
          app.workdir?.toLowerCase().includes(query) ?? false;
        const serverNameMatch = app.serverId
          ? (serverMap.get(app.serverId)?.toLowerCase().includes(query) ??
            false)
          : false;
        const ownerMatch =
          extractRepoOwner(app.repositoryUrl)?.toLowerCase().includes(query) ??
          false;

        return (
          nameMatch ||
          branchMatch ||
          repoMatch ||
          runtimeMatch ||
          versionMatch ||
          workdirMatch ||
          serverNameMatch ||
          ownerMatch
        );
      });
    }

    return list;
  }, [visibleApps, serverFilter, searchQuery, serverMap]);

  const getServerName = (serverId?: string): string | undefined => {
    if (!serverId) return undefined;
    return serverMap.get(serverId);
  };

  if (isLoading) {
    return <LoadingGrid count={6} columns={1} />;
  }

  if (error) {
    return <ErrorMessage message="Failed to load applications" />;
  }

  if (!visibleApps || visibleApps.length === 0) {
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

  const hasFilterActive =
    serverFilter !== ALL_SERVERS || searchQuery.trim() !== "";

  return (
    <div className="space-y-4">
      {/* 2. Toolbar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* Left: Server Filter & Search */}
        <div className="flex flex-wrap items-center gap-2.5 flex-1 min-w-[280px]">
          <Select value={serverFilter} onValueChange={setServerFilter}>
            <SelectTrigger
              className="w-[170px] sm:w-[190px] h-9 text-sm rounded-lg border-border/60 bg-card/60 shadow-sm"
              aria-label="Filter applications by server"
            >
              <Server
                className="h-3.5 w-3.5 mr-2 text-muted-foreground shrink-0"
                aria-hidden="true"
              />
              <SelectValue placeholder="All servers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_SERVERS}>All servers</SelectItem>
              {isAdmin && (
                <SelectItem value={LOCAL_SERVER}>
                  Local (self-hosted)
                </SelectItem>
              )}
              {servers?.map((server) => (
                <SelectItem key={server.id} value={server.id}>
                  {server.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Search Input */}
          <div className="relative flex-1 min-w-[180px] max-w-sm">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none"
              aria-hidden="true"
            />
            <Input
              type="search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search applications..."
              className="pl-9 pr-8 h-9 text-sm rounded-lg border-border/60 bg-card/60 shadow-sm placeholder:text-muted-foreground/70 focus-visible:ring-primary/40"
              aria-label="Search applications"
            />
            {searchQuery !== "" && (
              <button
                type="button"
                onClick={() => setSearchQuery("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                aria-label="Clear search"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </div>

        {/* Right: View Mode Toggle */}
        <div className="flex items-center rounded-lg border border-border/60 bg-card/60 p-0.5 shadow-sm shrink-0">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-8 w-8 p-0 rounded-md transition-colors",
              viewMode === "list" && "bg-muted text-foreground shadow-sm",
            )}
            onClick={() => setViewMode("list")}
            aria-label="List view"
            title="List view"
          >
            <List className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              "h-8 w-8 p-0 rounded-md transition-colors",
              viewMode === "grid" && "bg-muted text-foreground shadow-sm",
            )}
            onClick={() => setViewMode("grid")}
            aria-label="Grid view"
            title="Grid view"
          >
            <Grid3X3 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Empty State when filters yield no matches */}
      {filteredApps.length === 0 && hasFilterActive && (
        <EmptyState
          icon={Search}
          title="No applications found"
          description="No applications match your filter or search criteria."
          action={
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSearchQuery("");
                setServerFilter(ALL_SERVERS);
              }}
            >
              Reset filters
            </Button>
          }
        />
      )}

      {/* Grid View */}
      {filteredApps.length > 0 && viewMode === "grid" && (
        <div className="grid gap-3 sm:grid-cols-2">
          {filteredApps.map((app) => (
            <AppCard
              key={app.id}
              app={app}
              serverName={getServerName(app.serverId)}
            />
          ))}
        </div>
      )}

      {/* List View */}
      {filteredApps.length > 0 && viewMode === "list" && (
        <div className="flex flex-col gap-2.5">
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
