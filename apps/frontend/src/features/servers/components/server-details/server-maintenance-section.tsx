import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, Box, HardDrive, Loader2, Trash2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBytes, formatDateCompact } from "@/lib/format";
import { api } from "@/services/api";
import type { CleanupLog } from "@/services/api/docker";

interface ServerMaintenanceSectionProps {
  readonly serverId: string;
}

function CleanupTypeBadge({ type }: { readonly type: string }) {
  const variants: Record<string, "default" | "secondary" | "destructive"> = {
    containers: "default",
    volumes: "secondary",
    images: "destructive",
  };
  return <Badge variant={variants[type] ?? "default"}>{type}</Badge>;
}

function StatusBadge({ status }: { readonly status: string }) {
  return (
    <Badge variant={status === "success" ? "default" : "destructive"}>
      {status}
    </Badge>
  );
}

export function ServerMaintenanceSection({
  serverId,
}: ServerMaintenanceSectionProps) {
  const queryClient = useQueryClient();
  const [pruneType, setPruneType] = useState<"containers" | "volumes" | null>(
    null,
  );
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const cleanupLogsQuery = useQuery({
    queryKey: ["cleanup-logs", serverId],
    queryFn: () => api.cleanup.getLogs(serverId),
  });

  const invalidateCleanupLogs = async () => {
    await queryClient.invalidateQueries({
      queryKey: ["cleanup-logs", serverId],
    });
  };

  const pruneContainersMutation = useMutation({
    mutationFn: () => api.cleanup.pruneContainers(serverId),
    onSuccess: async () => {
      setErrorMessage(null);
      await invalidateCleanupLogs();
    },
    onError: () => {
      setErrorMessage(
        "Failed to prune containers. The server may be unreachable.",
      );
      void invalidateCleanupLogs();
    },
  });

  const pruneVolumesMutation = useMutation({
    mutationFn: () => api.cleanup.pruneVolumes(serverId),
    onSuccess: async () => {
      setErrorMessage(null);
      await invalidateCleanupLogs();
    },
    onError: () => {
      setErrorMessage(
        "Failed to prune volumes. The server may be unreachable.",
      );
      void invalidateCleanupLogs();
    },
  });

  const isPruning =
    pruneContainersMutation.isPending || pruneVolumesMutation.isPending;

  const handlePrune = async () => {
    try {
      if (pruneType === "containers") {
        await pruneContainersMutation.mutateAsync();
      } else if (pruneType === "volumes") {
        await pruneVolumesMutation.mutateAsync();
      }
    } catch {
      /* handled by onError */
    }
    setPruneType(null);
  };

  const lastResult =
    pruneContainersMutation.data ?? pruneVolumesMutation.data ?? null;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Docker Cleanup</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-col gap-2 sm:flex-row">
            <AlertDialog
              open={pruneType === "containers"}
              onOpenChange={(open) => {
                if (!open) setPruneType(null);
              }}
            >
              <AlertDialogTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={isPruning}
                  onClick={() => setPruneType("containers")}
                >
                  {pruneContainersMutation.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Box className="h-4 w-4 mr-2" />
                  )}
                  Prune Containers
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Prune stopped containers</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will remove all stopped containers from this server.
                    Running containers will not be affected.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handlePrune}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Prune Containers
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>

            <AlertDialog
              open={pruneType === "volumes"}
              onOpenChange={(open) => {
                if (!open) setPruneType(null);
              }}
            >
              <AlertDialogTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={isPruning}
                  onClick={() => setPruneType("volumes")}
                >
                  {pruneVolumesMutation.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <HardDrive className="h-4 w-4 mr-2" />
                  )}
                  Prune Volumes
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Prune unused volumes</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will remove all unused Docker volumes from this server.
                    Volumes attached to running containers will not be affected.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handlePrune}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Prune Volumes
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>

          {errorMessage != null && (
            <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3 text-sm text-destructive flex items-center gap-2">
              <AlertCircle className="h-4 w-4 flex-shrink-0" />
              <p>{errorMessage}</p>
            </div>
          )}

          {lastResult != null && (
            <div className="rounded-md bg-muted p-3 text-sm">
              <p>
                <span className="font-medium">
                  {lastResult.itemsRemoved} {lastResult.cleanupType}
                </span>{" "}
                removed &middot; {formatBytes(lastResult.spaceReclaimedBytes)}{" "}
                reclaimed
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base flex items-center gap-2">
            <Trash2 className="h-4 w-4" />
            Cleanup History
          </CardTitle>
        </CardHeader>
        <CardContent>
          <CleanupLogList
            logs={cleanupLogsQuery.data ?? []}
            isLoading={cleanupLogsQuery.isLoading}
          />
        </CardContent>
      </Card>
    </div>
  );
}

function CleanupLogList({
  logs,
  isLoading,
}: {
  readonly logs: readonly CleanupLog[];
  readonly isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        Loading...
      </p>
    );
  }

  if (logs.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        No cleanup history yet.
      </p>
    );
  }

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <div
          key={log.id}
          className="flex flex-col gap-1 rounded-md border p-3 text-sm sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="flex items-center gap-2 flex-wrap">
            <CleanupTypeBadge type={log.cleanupType} />
            <StatusBadge status={log.status} />
            <Badge variant="outline">{log.trigger}</Badge>
          </div>
          <div className="flex items-center gap-3 text-muted-foreground text-xs sm:text-sm">
            <span>{log.itemsRemoved} removed</span>
            <span>&middot;</span>
            <span>{formatBytes(log.spaceReclaimedBytes)}</span>
            <span>&middot;</span>
            <span>{formatDateCompact(log.createdAt)}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
