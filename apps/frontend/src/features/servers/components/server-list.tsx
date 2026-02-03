import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  Check,
  Circle,
  Loader2,
  Play,
  Server as ServerIcon,
  Trash2,
} from "lucide-react";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorMessage } from "@/components/error-message";
import { formatDate } from "@/lib/utils";
import type { Server, ServerStatus } from "@/types";
import { useProvisionProgress } from "../hooks/use-provision-progress";
import {
  useDeleteServer,
  useProvisionServer,
  useServers,
} from "../hooks/use-servers";
import { clearProvisionProgress } from "../provision-progress-store";

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

const STEP_LABELS: Record<string, string> = {
  ssh_connect: "Conectando via SSH",
  remote_env: "Verificando ambiente",
  sftp_client: "Conectando SFTP",
  install_dir: "Criando diretórios",
  agent_certs: "Instalando certificados",
  agent_binary: "Copiando agent",
  systemd_unit: "Configurando serviço",
  start_agent: "Iniciando agent",
};

function ServerCard({ server }: { readonly server: Server }) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [provisionDialogOpen, setProvisionDialogOpen] = useState(false);
  const provisionMutation = useProvisionServer();
  const deleteMutation = useDeleteServer();
  const provisionProgress = useProvisionProgress(server.id);
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (logsEndRef.current && provisionProgress?.logs.length) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [provisionProgress?.logs.length]);

  const handleProvision = () => {
    setProvisionDialogOpen(true);
    void provisionMutation.mutateAsync(server.id);
  };

  const handleProvisionDialogClose = (open: boolean) => {
    if (
      !open &&
      (provisionProgress?.status === "completed" ||
        provisionProgress?.status === "failed")
    ) {
      clearProvisionProgress(server.id);
    }
    setProvisionDialogOpen(open);
  };

  const isProvisioning =
    provisionMutation.isPending ||
    server.status === "provisioning" ||
    provisionProgress?.status === "running";

  const handleDelete = () => {
    void deleteMutation
      .mutateAsync(server.id)
      .then(() => setDeleteDialogOpen(false));
  };

  return (
    <>
      <Link to={`/servers/${server.id}`} className="block">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div className="flex items-center gap-2">
              <ServerIcon
                className="h-5 w-5 text-muted-foreground"
                aria-hidden
              />
              <span className="font-medium">{server.name}</span>
              <StatusBadge status={server.status} />
            </div>
            <div
              className="flex gap-2"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
              }}
            >
              <Button
                variant="outline"
                size="sm"
                onClick={handleProvision}
                disabled={isProvisioning}
              >
                {isProvisioning ? (
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
      </Link>

      <Dialog
        open={provisionDialogOpen}
        onOpenChange={handleProvisionDialogClose}
      >
        <DialogContent className="max-w-lg max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>Provisionamento: {server.name}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-4 flex-1 min-h-0">
            <div className="space-y-2">
              {[
                "ssh_connect",
                "remote_env",
                "sftp_client",
                "install_dir",
                "agent_certs",
                "agent_binary",
                "systemd_unit",
                "start_agent",
              ].map((step) => {
                const state = provisionProgress?.steps.find(
                  (s) => s.step === step,
                );
                const label = STEP_LABELS[step] ?? step;
                const isDone = state?.status === "ok";
                const isRunning = state?.status === "running";
                let StepIcon = Circle;
                let stepIconClass = "h-4 w-4 text-muted-foreground shrink-0";
                let labelClass = "text-muted-foreground/70";
                if (isDone) {
                  StepIcon = Check;
                  stepIconClass = "h-4 w-4 text-green-600 shrink-0";
                  labelClass = "text-muted-foreground";
                } else if (isRunning) {
                  StepIcon = Loader2;
                  stepIconClass = "h-4 w-4 animate-spin text-primary shrink-0";
                  labelClass = "font-medium";
                }
                return (
                  <div key={step} className="flex items-center gap-2 text-sm">
                    <StepIcon className={stepIconClass} aria-hidden />
                    <span className={labelClass}>{label}</span>
                  </div>
                );
              })}
            </div>
            <div className="flex-1 min-h-[120px] rounded border bg-muted/30 p-2">
              <ScrollArea className="h-[140px] w-full">
                <pre className="text-xs font-mono whitespace-pre-wrap break-words p-1">
                  {provisionProgress?.logs.length
                    ? provisionProgress.logs.join("\n")
                    : "Aguardando logs..."}
                </pre>
                <div ref={logsEndRef} />
              </ScrollArea>
            </div>
          </div>
        </DialogContent>
      </Dialog>

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
