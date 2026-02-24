import { useCallback } from "react";
import { useParams } from "react-router-dom";
import {
  Box,
  HardDrive,
  Monitor,
  RefreshCw,
  Server as ServerIcon,
  Settings,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ErrorMessage } from "@/components/error-message";
import { PageHeader } from "@/components/page-header";
import { ContainerList } from "@/features/containers";
import { ImageList } from "@/features/images";
import {
  AgentVersionCard,
  ResourceUsageSection,
  ServerAppsSection,
  ServerSettingsSection,
  SystemInfoBar,
} from "@/features/servers/components/server-details";
import { useServerStats } from "@/features/servers/hooks/use-server-stats";
import { useServer } from "@/features/servers/hooks/use-servers";
import { cn } from "@/lib/utils";
import type { Server } from "@/types";

function getStatusBadgeVariant(
  status: string,
): "default" | "destructive" | "secondary" {
  if (status === "online" || status === "provisioning") return "default";
  if (status === "error" || status === "offline") return "destructive";
  return "secondary";
}

interface ServerInfoCardProps {
  readonly server: Server;
  readonly stats: import("@/types").ServerStats | null;
  readonly onUpdated: () => void;
}

function ServerInfoCard({ server, stats, onUpdated }: ServerInfoCardProps) {
  const showAgent = server.status === "online";
  const showStats = stats != null;

  if (!showStats && !showAgent) return null;

  return (
    <Card>
      <CardContent className="py-3 space-y-0 divide-y divide-border">
        {showStats && (
          <div className={cn(showAgent && "pb-3")}>
            <SystemInfoBar stats={stats} />
          </div>
        )}
        {showAgent && (
          <div className={cn(showStats && "pt-3")}>
            <AgentVersionCard server={server} onUpdated={onUpdated} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export function ServerDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
    refetch: refetchServer,
  } = useServer(id);
  const {
    data: stats,
    isLoading: statsLoading,
    error: statsError,
    refetch: refetchStats,
    isFetching,
  } = useServerStats(id);

  const refetchAll = useCallback(() => {
    refetchServer();
    refetchStats();
  }, [refetchServer, refetchStats]);

  if (serverLoading || !id) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-14 w-full" />
        <Skeleton className="h-[52px] w-full" />
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-5">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-[80px]" />
          ))}
        </div>
      </div>
    );
  }

  if (serverError || !server) {
    return <ErrorMessage message="Server not found" />;
  }

  const hasStats = stats != null;
  const statsUnavailable = statsError != null;

  return (
    <div className="space-y-3">
      <PageHeader
        backTo="/servers"
        title={server.name}
        description={`${server.sshUser}@${server.host}:${server.sshPort}`}
        icon={ServerIcon}
        titleSuffix={
          <Badge variant={getStatusBadgeVariant(server.status)}>
            {server.status.charAt(0).toUpperCase() + server.status.slice(1)}
          </Badge>
        }
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={refetchAll}
            disabled={isFetching || statsUnavailable}
          >
            <RefreshCw
              className={cn("h-3.5 w-3.5 mr-1.5", isFetching && "animate-spin")}
            />
            Refresh
          </Button>
        }
      />

      <ServerInfoCard
        server={server}
        stats={stats ?? null}
        onUpdated={refetchAll}
      />

      <Tabs defaultValue="overview">
        <TabsList className="overflow-x-auto">
          <TabsTrigger value="overview">
            <Monitor className="h-3.5 w-3.5 mr-1.5" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="containers">
            <Box className="h-3.5 w-3.5 mr-1.5" />
            Containers
          </TabsTrigger>
          <TabsTrigger value="images">
            <HardDrive className="h-3.5 w-3.5 mr-1.5" />
            Images
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="h-3.5 w-3.5 mr-1.5" />
            Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-3">
          <ResourceUsageSection
            statsLoading={statsLoading}
            hasStats={hasStats}
            statsUnavailable={statsUnavailable}
            stats={stats ?? null}
            refetch={refetchStats}
            isFetching={isFetching}
          />
          <ServerAppsSection serverId={server.id} />
        </TabsContent>

        <TabsContent value="containers">
          <ContainerList serverId={server.id} />
        </TabsContent>

        <TabsContent value="images">
          <ImageList serverId={server.id} />
        </TabsContent>

        <TabsContent value="settings">
          <ServerSettingsSection server={server} onSaved={refetchAll} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
