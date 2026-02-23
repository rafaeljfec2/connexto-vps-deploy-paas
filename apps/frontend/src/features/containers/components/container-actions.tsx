import { Play, RefreshCw, ScrollText, Square, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  useRestartContainer,
  useStartContainer,
  useStopContainer,
} from "../hooks/use-containers";

interface ContainerActionsProps {
  readonly containerId: string;
  readonly serverId?: string;
  readonly isRunning: boolean;
  readonly onShowLogs: () => void;
  readonly onShowConsole: () => void;
}

export function ContainerActions({
  containerId,
  serverId,
  isRunning,
  onShowLogs,
  onShowConsole,
}: ContainerActionsProps) {
  const startContainer = useStartContainer();
  const stopContainer = useStopContainer();
  const restartContainer = useRestartContainer();

  const isLoading =
    startContainer.isPending ||
    stopContainer.isPending ||
    restartContainer.isPending;

  const isRemote = !!serverId;

  return (
    <div className="flex gap-1">
      {isRunning ? (
        <>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => stopContainer.mutate({ id: containerId, serverId })}
            disabled={isLoading}
            title="Stop"
          >
            <Square className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() =>
              restartContainer.mutate({ id: containerId, serverId })
            }
            disabled={isLoading}
            title="Restart"
          >
            <RefreshCw
              className={`h-4 w-4 ${restartContainer.isPending ? "animate-spin" : ""}`}
            />
          </Button>
        </>
      ) : (
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={() => startContainer.mutate({ id: containerId, serverId })}
          disabled={isLoading}
          title="Start"
        >
          <Play className="h-4 w-4" />
        </Button>
      )}
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8"
        onClick={onShowLogs}
        title="Logs"
      >
        <ScrollText className="h-4 w-4" />
      </Button>
      {isRunning && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={onShowConsole}
                  disabled={isRemote}
                  title={isRemote ? undefined : "Console (shell)"}
                >
                  <Terminal className="h-4 w-4" />
                </Button>
              </span>
            </TooltipTrigger>
            {isRemote && (
              <TooltipContent>
                <p>Console is not available for remote containers</p>
              </TooltipContent>
            )}
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}
