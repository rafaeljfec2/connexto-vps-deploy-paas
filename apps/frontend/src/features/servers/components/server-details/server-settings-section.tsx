import { useCallback, useState } from "react";
import { CheckCircle2 } from "lucide-react";
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
    </div>
  );
}
