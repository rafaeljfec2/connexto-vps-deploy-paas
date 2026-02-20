import { useState } from "react";
import {
  ChevronDown,
  ChevronUp,
  ExternalLink,
  HardDrive,
  MoreVertical,
  Network,
  Play,
  RefreshCw,
  ScrollText,
  Square,
  Terminal,
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { Container } from "@/types";
import {
  useRemoveContainer,
  useRestartContainer,
  useStartContainer,
  useStopContainer,
} from "../hooks/use-containers";
import { ContainerActions } from "./container-actions";
import { ContainerConsoleDialog } from "./container-console-dialog";
import { ContainerLogsDialog } from "./container-logs-dialog";
import {
  ContainerHealthBadge,
  ContainerStateBadge,
} from "./container-state-badge";

interface ContainerCardProps {
  readonly container: Container;
  readonly serverId?: string;
}

function formatPorts(ports: Container["ports"]): string {
  if (!ports || ports.length === 0) return "-";
  return ports
    .map((p) =>
      p.publicPort ? `${p.publicPort}:${p.privatePort}` : `${p.privatePort}`,
    )
    .join(", ");
}

function formatContainerCreated(created: string): string {
  const date = new Date(created);
  if (Number.isNaN(date.getTime())) return created;
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function ContainerCard({ container, serverId }: ContainerCardProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showLogsDialog, setShowLogsDialog] = useState(false);
  const [showConsoleDialog, setShowConsoleDialog] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const startContainer = useStartContainer();
  const stopContainer = useStopContainer();
  const restartContainer = useRestartContainer();
  const removeContainer = useRemoveContainer();

  const isRunning = container.state === "running";
  const isLoading =
    startContainer.isPending ||
    stopContainer.isPending ||
    restartContainer.isPending ||
    removeContainer.isPending;

  const handleDelete = () => {
    removeContainer.mutate(
      { id: container.id, force: true, serverId },
      { onSuccess: () => setShowDeleteDialog(false) },
    );
  };

  const dockerHubUrl = `https://hub.docker.com/r/${container.image.split(":")[0]}`;

  const hasNetworks = container.networks && container.networks.length > 0;
  const hasMounts = container.mounts && container.mounts.length > 0;
  const hasDetails = hasNetworks || hasMounts;
  const portsStr = formatPorts(container.ports);

  return (
    <>
      <tr className="border-b border-border hover:bg-muted/50 transition-colors align-middle">
        <td className="py-3 px-4 min-w-0">
          <div className="flex items-center gap-2 min-w-0">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="font-medium truncate block min-w-0">
                    {container.name}
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-xs">
                  <p className="font-mono break-all">{container.name}</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            {container.isFlowDeployManaged && (
              <Badge
                variant="outline"
                className="shrink-0 text-[10px] px-1.5 py-0"
              >
                FD
              </Badge>
            )}
          </div>
        </td>
        <td className="py-3 px-4 whitespace-nowrap">
          <div className="flex items-center gap-2">
            <ContainerStateBadge state={container.state} />
            <ContainerHealthBadge health={container.health} />
          </div>
        </td>
        <td className="py-3 px-4 hidden md:table-cell whitespace-nowrap">
          <ContainerActions
            containerId={container.id}
            serverId={serverId}
            isRunning={isRunning}
            onShowLogs={() => setShowLogsDialog(true)}
            onShowConsole={() => setShowConsoleDialog(true)}
          />
        </td>
        <td className="py-3 px-4 hidden lg:table-cell min-w-0 max-w-[200px]">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-sm text-muted-foreground truncate block">
                  {container.image}
                </span>
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-sm">
                <p className="font-mono break-all">{container.image}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </td>
        <td className="py-3 px-4 hidden xl:table-cell whitespace-nowrap">
          <span className="text-xs text-muted-foreground font-mono">
            {container.ipAddress ?? "-"}
          </span>
        </td>
        <td className="py-3 px-4 hidden xl:table-cell min-w-0 max-w-[140px]">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-xs text-muted-foreground font-mono truncate block">
                  {portsStr}
                </span>
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs">
                <p className="font-mono break-all">{portsStr}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </td>
        <td className="py-3 px-4 hidden 2xl:table-cell whitespace-nowrap">
          <div className="flex items-center gap-2">
            {hasNetworks && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge variant="outline" className="gap-1 cursor-help">
                      <Network className="h-3 w-3" />
                      {container.networks.length}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className="font-medium mb-1">Networks</p>
                    {container.networks.map((network) => (
                      <p key={network} className="text-xs">
                        {network}
                      </p>
                    ))}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
            {hasMounts && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge variant="outline" className="gap-1 cursor-help">
                      <HardDrive className="h-3 w-3" />
                      {container.mounts.length}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className="font-medium mb-1">Volumes/Mounts</p>
                    {container.mounts.map((mount) => (
                      <p
                        key={`${mount.type}:${mount.source}:${mount.destination}:${mount.readOnly}`}
                        className="text-xs font-mono"
                      >
                        {mount.source.length > 30
                          ? `...${mount.source.slice(-30)}`
                          : mount.source}{" "}
                        → {mount.destination}
                        {mount.readOnly && " (ro)"}
                      </p>
                    ))}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
            {hasDetails && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                onClick={() => setExpanded(!expanded)}
              >
                {expanded ? (
                  <ChevronUp className="h-4 w-4" />
                ) : (
                  <ChevronDown className="h-4 w-4" />
                )}
              </Button>
            )}
          </div>
        </td>
        <td className="py-3 px-4 hidden lg:table-cell whitespace-nowrap">
          <span className="text-sm text-muted-foreground">
            {formatContainerCreated(container.created)}
          </span>
        </td>
        <td className="py-3 px-4 whitespace-nowrap">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {isRunning ? (
                <>
                  <DropdownMenuItem
                    onClick={() =>
                      stopContainer.mutate({ id: container.id, serverId })
                    }
                    disabled={isLoading}
                  >
                    <Square className="mr-2 h-4 w-4" />
                    Stop
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() =>
                      restartContainer.mutate({ id: container.id, serverId })
                    }
                    disabled={isLoading}
                  >
                    <RefreshCw className="mr-2 h-4 w-4" />
                    Restart
                  </DropdownMenuItem>
                </>
              ) : (
                <DropdownMenuItem
                  onClick={() =>
                    startContainer.mutate({ id: container.id, serverId })
                  }
                  disabled={isLoading}
                >
                  <Play className="mr-2 h-4 w-4" />
                  Start
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={() => setShowLogsDialog(true)}>
                <ScrollText className="mr-2 h-4 w-4" />
                View Logs
              </DropdownMenuItem>
              {isRunning && (
                <DropdownMenuItem onClick={() => setShowConsoleDialog(true)}>
                  <Terminal className="mr-2 h-4 w-4" />
                  Open Console
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => window.open(dockerHubUrl, "_blank")}
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                View on Docker Hub
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => setShowDeleteDialog(true)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Remove
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </td>
      </tr>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {container.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the container. If the container is running, it
              will be force stopped first.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removeContainer.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={removeContainer.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeContainer.isPending ? "Removing..." : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <ContainerLogsDialog
        containerId={container.id}
        serverId={serverId}
        containerName={container.name}
        open={showLogsDialog}
        onOpenChange={setShowLogsDialog}
      />

      <ContainerConsoleDialog
        containerId={container.id}
        containerName={container.name}
        open={showConsoleDialog}
        onOpenChange={setShowConsoleDialog}
      />

      {expanded && hasDetails && (
        <tr className="border-b border-border bg-muted/30">
          <td colSpan={9} className="py-3 px-4">
            <div className="grid gap-4 md:grid-cols-2">
              {hasNetworks && (
                <div>
                  <h4 className="text-sm font-medium flex items-center gap-2 mb-2">
                    <Network className="h-4 w-4" />
                    Networks ({container.networks.length})
                  </h4>
                  <div className="flex flex-wrap gap-1">
                    {container.networks.map((network) => (
                      <Badge
                        key={network}
                        variant="secondary"
                        className="text-xs"
                      >
                        {network}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
              {hasMounts && (
                <div>
                  <h4 className="text-sm font-medium flex items-center gap-2 mb-2">
                    <HardDrive className="h-4 w-4" />
                    Volumes/Mounts ({container.mounts.length})
                  </h4>
                  <div className="space-y-1">
                    {container.mounts.map((mount) => (
                      <div
                        key={`${mount.type}:${mount.source}:${mount.destination}:${mount.readOnly}`}
                        className="text-xs font-mono bg-muted rounded px-2 py-1"
                      >
                        <span className="text-muted-foreground">
                          {mount.type}:
                        </span>{" "}
                        <span className="break-all">{mount.source}</span>
                        <span className="text-muted-foreground mx-1">→</span>
                        <span className="break-all">{mount.destination}</span>
                        {mount.readOnly && (
                          <Badge variant="outline" className="ml-2 text-[10px]">
                            RO
                          </Badge>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}
