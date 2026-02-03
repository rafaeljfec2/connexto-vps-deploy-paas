import { useState } from "react";
import { Loader2, Play, Server as ServerIcon, Trash2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { formatDate } from "@/lib/utils";
import type { Server, ServerStatus } from "@/types";
import {
  useDeleteServer,
  useProvisionServer,
  useServers,
} from "../hooks/use-servers";

function StatusBadge({ status }: { readonly status: ServerStatus }) {
  const variants: Record<
    ServerStatus,
    {
      variant: "default" | "secondary" | "destructive" | "outline";
      label: string;
    }
  > = {
    pending: { variant: "secondary", label: "Pending" },
    provisioning: { variant: "default", label: "Provisioning" },
    online: { variant: "default", label: "Online" },
    offline: { variant: "outline", label: "Offline" },
    error: { variant: "destructive", label: "Error" },
  };
  const { variant, label } = variants[status] ?? {
    variant: "secondary" as const,
    label: status,
  };
  return <Badge variant={variant}>{label}</Badge>;
}

function ServerCard({ server }: { readonly server: Server }) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const provisionMutation = useProvisionServer();
  const deleteMutation = useDeleteServer();

  const handleProvision = () => {
    void provisionMutation.mutateAsync(server.id);
  };

  const handleDelete = () => {
    void deleteMutation
      .mutateAsync(server.id)
      .then(() => setDeleteDialogOpen(false));
  };

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="flex items-center gap-2">
            <ServerIcon className="h-5 w-5 text-muted-foreground" aria-hidden />
            <span className="font-medium">{server.name}</span>
            <StatusBadge status={server.status} />
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleProvision}
              disabled={
                provisionMutation.isPending || server.status === "provisioning"
              }
            >
              {provisionMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              ) : (
                <Play className="h-4 w-4" aria-hidden />
              )}
              <span className="ml-2">Provision</span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setDeleteDialogOpen(true)}
              disabled={deleteMutation.isPending}
              aria-label={`Delete ${server.name}`}
            >
              <Trash2 className="h-4 w-4 text-destructive" aria-hidden />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground space-y-1">
          <p>
            {server.sshUser}@{server.host}:{server.sshPort}
          </p>
          {server.agentVersion != null && <p>Agent: {server.agentVersion}</p>}
          {server.lastHeartbeatAt != null && (
            <p>Last heartbeat: {formatDate(server.lastHeartbeatAt)}</p>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete server?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove &quot;{server.name}&quot; from the list. Apps
              using this server will fall back to local deploy.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

export function ServerList() {
  const { data: servers, isLoading, error } = useServers();

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (error) {
    return <ErrorMessage message="Failed to load servers" />;
  }

  if (!servers?.length) {
    return (
      <EmptyState
        icon={ServerIcon}
        title="No servers"
        description="Add a remote server to enable deploy to other machines."
      />
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {servers.map((server) => (
        <ServerCard key={server.id} server={server} />
      ))}
    </div>
  );
}
