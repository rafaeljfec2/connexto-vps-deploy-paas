import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  Activity,
  ArrowUpRight,
  Check,
  CheckCircle2,
  Circle,
  Copy,
  Cpu,
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
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useRelativeTick } from "@/hooks/use-relative-tick";
import { formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Server, ServerStatus } from "@/types";
import { useProvisionProgress } from "../hooks/use-provision-progress";
import { useDeleteServer, useProvisionServer } from "../hooks/use-servers";
import { clearProvisionProgress } from "../provision-progress-store";

interface StatusDescriptor {
  readonly label: string;
  readonly dotClass: string;
  readonly ringClass: string;
  readonly badgeClass: string;
  readonly pulse: boolean;
}

const STATUS: Record<ServerStatus, StatusDescriptor> = {
  online: {
    label: "Online",
    dotClass: "bg-emerald-500",
    ringClass: "ring-emerald-500/20",
    badgeClass:
      "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    pulse: true,
  },
  provisioning: {
    label: "Provisioning",
    dotClass: "bg-blue-500",
    ringClass: "ring-blue-500/20",
    badgeClass:
      "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300",
    pulse: true,
  },
  pending: {
    label: "Pending",
    dotClass: "bg-yellow-500",
    ringClass: "ring-yellow-500/20",
    badgeClass:
      "border-yellow-500/30 bg-yellow-500/10 text-yellow-700 dark:text-yellow-300",
    pulse: false,
  },
  offline: {
    label: "Offline",
    dotClass: "bg-muted-foreground/50",
    ringClass: "ring-muted-foreground/10",
    badgeClass: "border-muted-foreground/30 bg-muted/40 text-muted-foreground",
    pulse: false,
  },
  error: {
    label: "Error",
    dotClass: "bg-red-500",
    ringClass: "ring-red-500/20",
    badgeClass:
      "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300",
    pulse: false,
  },
};

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

const PROVISION_STEPS = [
  "ssh_connect",
  "remote_env",
  "sftp_client",
  "install_dir",
  "agent_certs",
  "agent_binary",
  "systemd_unit",
  "start_agent",
] as const;

function StatusDot({ status }: { readonly status: ServerStatus }) {
  const s = STATUS[status];
  return (
    <span className="relative inline-flex h-2.5 w-2.5 shrink-0">
      {s.pulse && (
        <span
          className={cn(
            "absolute inline-flex h-full w-full animate-ping rounded-full opacity-75",
            s.dotClass,
          )}
          aria-hidden="true"
        />
      )}
      <span
        className={cn(
          "relative inline-flex h-2.5 w-2.5 rounded-full ring-2",
          s.dotClass,
          s.ringClass,
        )}
        aria-hidden="true"
      />
    </span>
  );
}

function StatusBadge({ status }: { readonly status: ServerStatus }) {
  const s = STATUS[status];
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide",
        s.badgeClass,
      )}
    >
      {s.label}
    </span>
  );
}

function CopyAddressButton({ value }: { readonly value: string }) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const t = window.setTimeout(() => setCopied(false), 1500);
    return () => window.clearTimeout(t);
  }, [copied]);

  const handleCopy = async (e: React.MouseEvent | React.KeyboardEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!navigator.clipboard) return;
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
    } catch {
      // swallow — browser may block clipboard in insecure contexts
    }
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className="ml-1 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      aria-label={copied ? "Address copied" : "Copy address"}
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-emerald-500" aria-hidden="true" />
      ) : (
        <Copy className="h-3.5 w-3.5" aria-hidden="true" />
      )}
    </button>
  );
}

interface ServerCardProps {
  readonly server: Server;
}

export function ServerCard({ server }: ServerCardProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [provisionDialogOpen, setProvisionDialogOpen] = useState(false);
  const provisionMutation = useProvisionServer();
  const deleteMutation = useDeleteServer();
  const provisionProgress = useProvisionProgress(server.id);
  const logsEndRef = useRef<HTMLDivElement>(null);

  useRelativeTick();

  useEffect(() => {
    if (logsEndRef.current && provisionProgress?.logs.length) {
      logsEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [provisionProgress?.logs.length]);

  const address = `${server.sshUser}@${server.host}:${server.sshPort}`;

  const isProvisioning =
    provisionMutation.isPending ||
    server.status === "provisioning" ||
    provisionProgress?.status === "running";

  const needsAttention =
    server.status === "error" || server.status === "pending";

  const isAgentOutdated = useMemo(() => {
    if (server.agentVersion == null) return false;
    return server.agentVersion !== server.latestAgentVersion;
  }, [server.agentVersion, server.latestAgentVersion]);

  const heartbeat = server.lastHeartbeatAt
    ? formatRelativeTime(server.lastHeartbeatAt)
    : null;

  const handleProvision = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
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

  const handleOpenDelete = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDeleteDialogOpen(true);
  };

  const handleDelete = () => {
    void deleteMutation
      .mutateAsync(server.id)
      .then(() => setDeleteDialogOpen(false));
  };

  return (
    <>
      <Link
        to={`/servers/${server.id}`}
        className="group block focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-xl"
      >
        <Card
          className={cn(
            "relative flex h-full flex-col gap-3 overflow-hidden p-4 transition-all",
            "hover:border-border hover:shadow-sm",
            needsAttention && "border-yellow-500/40",
            server.status === "error" && "border-red-500/40",
          )}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-muted/40 text-muted-foreground">
                <ServerIcon className="h-4 w-4" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                  <StatusDot status={server.status} />
                  <span className="truncate text-sm font-semibold text-foreground">
                    {server.name}
                  </span>
                </div>
                <div className="mt-0.5 flex items-center gap-1 text-[11px] text-muted-foreground">
                  <span className="font-mono truncate">{address}</span>
                  <CopyAddressButton value={address} />
                </div>
              </div>
            </div>
            <StatusBadge status={server.status} />
          </div>

          <div className="grid grid-cols-2 gap-2 rounded-lg border border-border/40 bg-muted/20 px-3 py-2 text-[11px]">
            <div className="flex min-w-0 items-center gap-1.5 text-muted-foreground">
              <Cpu className="h-3 w-3 shrink-0" aria-hidden="true" />
              <span className="font-medium uppercase tracking-wide text-[9px]">
                Agent
              </span>
              <span className="ml-auto truncate font-mono text-foreground">
                {server.agentVersion != null ? `v${server.agentVersion}` : "—"}
              </span>
              {isAgentOutdated && (
                <span
                  className="inline-flex items-center rounded-sm border border-yellow-500/30 bg-yellow-500/10 px-1 text-[9px] font-medium uppercase tracking-wide text-yellow-700 dark:text-yellow-300"
                  title={`Latest: v${server.latestAgentVersion}`}
                >
                  Old
                </span>
              )}
            </div>
            <div className="flex min-w-0 items-center gap-1.5 text-muted-foreground">
              <Activity className="h-3 w-3 shrink-0" aria-hidden="true" />
              <span className="font-medium uppercase tracking-wide text-[9px]">
                Heartbeat
              </span>
              <span
                className="ml-auto truncate text-foreground"
                title={server.lastHeartbeatAt ?? undefined}
              >
                {heartbeat ?? "never"}
              </span>
            </div>
          </div>

          <div
            className="mt-auto flex items-center justify-between gap-2"
            role="toolbar"
            aria-label={`Actions for ${server.name}`}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onKeyDown={(e) => e.stopPropagation()}
          >
            <Button
              variant={needsAttention ? "default" : "outline"}
              size="sm"
              className="h-8 flex-1"
              onClick={handleProvision}
              disabled={isProvisioning}
            >
              {isProvisioning ? (
                <Loader2
                  className="mr-1.5 h-3.5 w-3.5 animate-spin"
                  aria-hidden="true"
                />
              ) : (
                <Play className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              )}
              {isProvisioning ? "Provisioning" : "Provision"}
            </Button>
            <span
              className="inline-flex items-center gap-1 text-[11px] font-medium text-muted-foreground transition-colors group-hover:text-foreground"
              aria-hidden="true"
            >
              Details
              <ArrowUpRight className="h-3.5 w-3.5" />
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
              onClick={handleOpenDelete}
              disabled={deleteMutation.isPending}
              aria-label={`Delete ${server.name}`}
            >
              <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
            </Button>
          </div>
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
              {PROVISION_STEPS.map((step) => {
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
                  StepIcon = CheckCircle2;
                  stepIconClass = "h-4 w-4 text-emerald-600 shrink-0";
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
