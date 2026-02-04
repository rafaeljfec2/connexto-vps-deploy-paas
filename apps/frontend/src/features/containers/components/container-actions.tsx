import { Play, RefreshCw, ScrollText, Square, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  useRestartContainer,
  useStartContainer,
  useStopContainer,
} from "../hooks/use-containers";

interface ContainerActionsProps {
  readonly containerId: string;
  readonly isRunning: boolean;
  readonly onShowLogs: () => void;
  readonly onShowConsole: () => void;
}

export function ContainerActions({
  containerId,
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

  return (
    <div className="flex gap-1">
      {isRunning ? (
        <>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => stopContainer.mutate(containerId)}
            disabled={isLoading}
            title="Stop"
          >
            <Square className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => restartContainer.mutate(containerId)}
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
          onClick={() => startContainer.mutate(containerId)}
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
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={onShowConsole}
          title="Console (shell)"
        >
          <Terminal className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}
