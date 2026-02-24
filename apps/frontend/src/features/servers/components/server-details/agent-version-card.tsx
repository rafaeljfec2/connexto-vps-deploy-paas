import { useCallback, useEffect, useState } from "react";
import { ArrowUpCircle, CheckCircle2, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { clearAgentUpdateState } from "@/features/servers/agent-update-store";
import { useAgentUpdate } from "@/features/servers/hooks/use-agent-update";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import type { Server } from "@/types";

function compareSemver(a: string, b: string): number {
  const pa = a.split(".").map(Number);
  const pb = b.split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    const diff = (pa[i] ?? 0) - (pb[i] ?? 0);
    if (diff !== 0) return diff;
  }
  return 0;
}

function getAgentVersionStatus(server: Server) {
  const currentVersion = server.agentVersion ?? null;
  const latestVersion = server.latestAgentVersion;
  const isUnknown = currentVersion == null;
  const isUpToDate =
    !isUnknown && compareSemver(currentVersion, latestVersion) >= 0;
  const isOutdated = !isUnknown && !isUpToDate;
  const needsUpdate = isUnknown || isOutdated;

  return {
    currentVersion,
    latestVersion,
    isUnknown,
    isUpToDate,
    isOutdated,
    needsUpdate,
  };
}

const AGENT_UPDATE_STEP_LABELS: Record<string, string> = {
  enqueued: "Waiting for agent heartbeat...",
  delivered: "Agent downloading update...",
  updated: "Update completed!",
};

interface AgentVersionCardProps {
  readonly server: Server;
  readonly onUpdated: () => void;
}

export function AgentVersionCard({ server, onUpdated }: AgentVersionCardProps) {
  const [isSending, setIsSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const agentUpdate = useAgentUpdate(server.id);

  const {
    currentVersion,
    latestVersion,
    isUnknown,
    isUpToDate,
    isOutdated,
    needsUpdate,
  } = getAgentVersionStatus(server);

  const isUpdateInProgress = agentUpdate?.status === "running";
  const isUpdateCompleted = agentUpdate?.status === "completed";
  const isUpdateError = agentUpdate?.status === "error";

  useEffect(() => {
    if (!isUpdateCompleted) return;
    onUpdated();
    const timeout = globalThis.setTimeout(() => {
      clearAgentUpdateState(server.id);
    }, 5000);
    return () => globalThis.clearTimeout(timeout);
  }, [isUpdateCompleted, onUpdated, server.id]);

  useEffect(() => {
    if (!isUpdateError) return;
    const timeout = globalThis.setTimeout(() => {
      clearAgentUpdateState(server.id);
    }, 10000);
    return () => globalThis.clearTimeout(timeout);
  }, [isUpdateError, server.id]);

  const handleUpdate = useCallback(async () => {
    setIsSending(true);
    setSendError(null);
    try {
      await api.servers.updateAgent(server.id);
    } catch {
      setSendError("Failed to enqueue agent update");
    } finally {
      setIsSending(false);
    }
  }, [server.id]);

  if (server.status !== "online") return null;

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-wrap items-center gap-2 sm:gap-3">
        <span className="text-xs font-medium text-muted-foreground">
          Agent:
        </span>
        {isUnknown && <Badge variant="secondary">Unknown version</Badge>}
        {!isUnknown && (
          <Badge variant={isOutdated ? "destructive" : "default"}>
            v{currentVersion}
          </Badge>
        )}
        {(isOutdated || isUnknown) &&
          latestVersion &&
          !isUpdateInProgress &&
          !isUpdateCompleted && (
            <span className="text-xs text-muted-foreground">
              v{latestVersion} available
            </span>
          )}
        {isUpToDate && !isUpdateCompleted && (
          <span className="flex items-center gap-1 text-xs text-emerald-500">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Up to date
          </span>
        )}
      </div>

      {isUpdateInProgress && agentUpdate != null && (
        <span className="flex items-center gap-1.5 text-xs text-blue-500">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {AGENT_UPDATE_STEP_LABELS[agentUpdate.step] ?? agentUpdate.step}
        </span>
      )}

      {isUpdateCompleted && agentUpdate != null && (
        <span className="flex items-center gap-1.5 text-xs text-emerald-500">
          <CheckCircle2 className="h-3.5 w-3.5" />
          Updated to v{agentUpdate.version}
        </span>
      )}

      {isUpdateError && agentUpdate != null && (
        <span className="text-xs text-red-500">
          Update failed: {agentUpdate.errorMessage ?? "unknown error"}
        </span>
      )}

      {!isUpdateInProgress && !isUpdateCompleted && needsUpdate && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleUpdate}
          disabled={isSending}
        >
          <ArrowUpCircle
            className={cn("h-3.5 w-3.5 mr-1.5", isSending && "animate-spin")}
          />
          {isSending ? "Sending..." : "Update Agent"}
        </Button>
      )}

      {sendError != null && (
        <span className="text-xs text-red-500">{sendError}</span>
      )}
    </div>
  );
}
