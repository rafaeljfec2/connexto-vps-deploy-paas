import { useCallback, useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  Loader2,
  RefreshCw,
  ScrollText,
  Shield,
  Terminal,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/services/api";
import type { ManageServerResponse } from "@/services/api/servers";
import type { AgentUpdateMode, Server } from "@/types";

interface ServerSettingsSectionProps {
  readonly server: Server;
  readonly onSaved: () => void;
}

export function ServerSettingsSection({
  server,
  onSaved,
}: ServerSettingsSectionProps) {
  const [updateMode, setUpdateMode] = useState<AgentUpdateMode>(
    server.agentUpdateMode ?? "grpc",
  );
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const isDirty = updateMode !== (server.agentUpdateMode ?? "grpc");

  const handleSave = useCallback(async () => {
    setIsSaving(true);
    setSaveError(null);
    setSaved(false);
    try {
      await api.servers.update(server.id, { agentUpdateMode: updateMode });
      setSaved(true);
      onSaved();
      globalThis.setTimeout(() => setSaved(false), 3000);
    } catch {
      setSaveError("Failed to save settings");
    } finally {
      setIsSaving(false);
    }
  }, [server.id, updateMode, onSaved]);

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="py-4 space-y-4">
          <h3 className="text-sm font-semibold">Connection</h3>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 max-w-xl">
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Host</Label>
              <Input
                value={server.host}
                readOnly
                className="h-8 text-xs bg-muted/50"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">SSH Port</Label>
              <Input
                value={String(server.sshPort)}
                readOnly
                className="h-8 text-xs bg-muted/50"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">SSH User</Label>
              <Input
                value={server.sshUser}
                readOnly
                className="h-8 text-xs bg-muted/50"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="py-4 space-y-4">
          <h3 className="text-sm font-semibold">Agent Settings</h3>

          <div className="space-y-2 max-w-sm">
            <Label htmlFor="agent-update-mode">Agent Update Mode</Label>
            <Select
              value={updateMode}
              onValueChange={(v) => setUpdateMode(v as AgentUpdateMode)}
            >
              <SelectTrigger id="agent-update-mode">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="grpc">gRPC (Push direto)</SelectItem>
                <SelectItem value="https">
                  HTTPS (Pull via heartbeat)
                </SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              gRPC pushes the binary directly through the existing connection.
              HTTPS makes the agent download the binary via HTTP.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <Button
              size="sm"
              onClick={handleSave}
              disabled={!isDirty || isSaving}
            >
              {isSaving ? "Saving..." : "Save"}
            </Button>
            {saved && (
              <span className="flex items-center gap-1 text-xs text-emerald-500">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Saved
              </span>
            )}
            {saveError != null && (
              <span className="text-xs text-red-500">{saveError}</span>
            )}
          </div>
        </CardContent>
      </Card>

      <ServerManagementCard serverId={server.id} />
    </div>
  );
}

type ManageActionId =
  | "restart_agent"
  | "restart_user_manager"
  | "agent_logs"
  | "fix_docker_permissions";

interface ActionConfig {
  readonly id: ManageActionId;
  readonly label: string;
  readonly description: string;
  readonly icon: React.ReactNode;
  readonly variant: "default" | "outline" | "destructive";
  readonly confirm?: string;
}

const MANAGE_ACTIONS: readonly ActionConfig[] = [
  {
    id: "restart_agent",
    label: "Restart Agent",
    description: "Restart the deploy agent service",
    icon: <RefreshCw className="h-4 w-4" />,
    variant: "outline",
  },
  {
    id: "agent_logs",
    label: "View Logs",
    description: "Fetch last 100 lines of agent logs",
    icon: <ScrollText className="h-4 w-4" />,
    variant: "outline",
  },
  {
    id: "restart_user_manager",
    label: "Restart User Manager",
    description: "Apply group membership changes (agent will restart)",
    icon: <Terminal className="h-4 w-4" />,
    variant: "outline",
    confirm:
      "This will restart the systemd user manager. The agent will be restarted automatically. Continue?",
  },
  {
    id: "fix_docker_permissions",
    label: "Fix Docker Permissions",
    description: "Add user to docker group and restart services",
    icon: <Shield className="h-4 w-4" />,
    variant: "outline",
    confirm:
      "This will add the SSH user to the docker group and restart the user manager + agent. Continue?",
  },
] as const;

interface ServerManagementCardProps {
  readonly serverId: string;
}

function ServerManagementCard({ serverId }: ServerManagementCardProps) {
  const [loadingAction, setLoadingAction] = useState<ManageActionId | null>(
    null,
  );
  const [result, setResult] = useState<ManageServerResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showLogs, setShowLogs] = useState(false);

  const executeAction = useCallback(
    async (action: ActionConfig) => {
      if (action.confirm && !globalThis.confirm(action.confirm)) {
        return;
      }

      setLoadingAction(action.id);
      setResult(null);
      setError(null);
      setShowLogs(false);

      try {
        const response = await api.servers.manage(serverId, action.id);
        setResult(response);
        if (action.id === "agent_logs") {
          setShowLogs(true);
        }
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to execute action";
        setError(message);
      } finally {
        setLoadingAction(null);
      }
    },
    [serverId],
  );

  return (
    <Card>
      <CardContent className="py-4 space-y-4">
        <div>
          <h3 className="text-sm font-semibold">Server Management</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Manage agent and server services via SSH
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {MANAGE_ACTIONS.map((action) => {
            const isLoading = loadingAction === action.id;
            return (
              <Button
                key={action.id}
                variant={action.variant}
                size="sm"
                className="justify-start gap-2 h-auto py-2 px-3"
                disabled={loadingAction !== null}
                onClick={() => executeAction(action)}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  action.icon
                )}
                <div className="text-left">
                  <div className="text-xs font-medium">{action.label}</div>
                  <div className="text-[10px] text-muted-foreground font-normal">
                    {action.description}
                  </div>
                </div>
              </Button>
            );
          })}
        </div>

        <ManageResultFeedback
          result={result}
          error={error}
          showLogs={showLogs}
          onToggleLogs={() => setShowLogs((prev) => !prev)}
        />
      </CardContent>
    </Card>
  );
}

interface ManageResultFeedbackProps {
  readonly result: ManageServerResponse | null;
  readonly error: string | null;
  readonly showLogs: boolean;
  readonly onToggleLogs: () => void;
}

function ManageResultFeedback({
  result,
  error,
  showLogs,
  onToggleLogs,
}: ManageResultFeedbackProps) {
  if (error) {
    return (
      <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/5 p-3">
        <XCircle className="h-4 w-4 text-destructive mt-0.5 shrink-0" />
        <p className="text-xs text-destructive">{error}</p>
      </div>
    );
  }

  if (!result) {
    return null;
  }

  const isLogOutput =
    result.output.includes("\n") && result.output.length > 200;

  return (
    <div className="space-y-2">
      <div
        className={`flex items-start gap-2 rounded-md border p-3 ${
          result.success
            ? "border-emerald-500/50 bg-emerald-500/5"
            : "border-amber-500/50 bg-amber-500/5"
        }`}
      >
        {result.success ? (
          <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
        ) : (
          <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
        )}
        {isLogOutput ? (
          <div className="flex-1 min-w-0">
            <button
              type="button"
              onClick={onToggleLogs}
              className="text-xs font-medium hover:underline cursor-pointer"
            >
              {showLogs ? "Hide log output" : "Show log output"}
            </button>
          </div>
        ) : (
          <pre className="text-xs whitespace-pre-wrap break-words flex-1 min-w-0">
            {result.output}
          </pre>
        )}
      </div>

      {isLogOutput && showLogs && (
        <div className="rounded-md border bg-muted/50 overflow-hidden">
          <pre className="text-[11px] leading-relaxed p-3 overflow-x-auto max-h-96 font-mono">
            {result.output}
          </pre>
        </div>
      )}
    </div>
  );
}
