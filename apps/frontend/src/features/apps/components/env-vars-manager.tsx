import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Eye,
  EyeOff,
  Plus,
  Save,
  Trash2,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  useCreateEnvVar,
  useDeleteEnvVar,
  useEnvVars,
  useUpdateEnvVar,
} from "../hooks/use-env-vars";

interface EnvVarsManagerProps {
  readonly appId: string;
  readonly embedded?: boolean;
}

interface EditingVar {
  readonly id: string | null;
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
}

const INITIAL_EDITING: EditingVar = {
  id: null,
  key: "",
  value: "",
  isSecret: false,
};

export function EnvVarsManager({
  appId,
  embedded = false,
}: EnvVarsManagerProps) {
  const { data: envVars, isLoading } = useEnvVars(appId);
  const createEnvVar = useCreateEnvVar(appId);
  const updateEnvVar = useUpdateEnvVar(appId);
  const deleteEnvVar = useDeleteEnvVar(appId);

  const [isExpanded, setIsExpanded] = useState(embedded);
  const [isAdding, setIsAdding] = useState(false);
  const [editing, setEditing] = useState<EditingVar>(INITIAL_EDITING);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});

  const handleAdd = () => {
    setIsAdding(true);
    setEditing(INITIAL_EDITING);
  };

  const handleCancelAdd = () => {
    setIsAdding(false);
    setEditing(INITIAL_EDITING);
  };

  const handleSaveNew = async () => {
    if (!editing.key.trim()) return;

    await createEnvVar.mutateAsync({
      key: editing.key.trim().toUpperCase(),
      value: editing.value,
      isSecret: editing.isSecret,
    });

    setIsAdding(false);
    setEditing(INITIAL_EDITING);
  };

  const handleStartEdit = (id: string, value: string, isSecret: boolean) => {
    setEditingId(id);
    setEditing({ id, key: "", value: isSecret ? "" : value, isSecret });
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditing(INITIAL_EDITING);
  };

  const handleSaveEdit = async () => {
    if (!editingId) return;

    await updateEnvVar.mutateAsync({
      varId: editingId,
      input: {
        value: editing.value,
        isSecret: editing.isSecret,
      },
    });

    setEditingId(null);
    setEditing(INITIAL_EDITING);
  };

  const handleDelete = async (id: string) => {
    await deleteEnvVar.mutateAsync(id);
  };

  const toggleShowSecret = (id: string) => {
    setShowSecrets((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const varsCount = envVars?.length ?? 0;

  const content = (
    <div className="space-y-3">
      {!isAdding && (
        <div className="flex justify-end">
          <Button size="sm" onClick={handleAdd}>
            <Plus className="h-4 w-4 mr-1" />
            Add Variable
          </Button>
        </div>
      )}
      {isLoading && <p className="text-sm text-muted-foreground">Loading...</p>}
      {isAdding && (
        <div className="flex flex-col gap-2 p-3 border rounded-lg bg-muted/50">
          <div className="flex gap-2">
            <Input
              placeholder="KEY_NAME"
              value={editing.key}
              onChange={(e) =>
                setEditing((prev) => ({
                  ...prev,
                  key: e.target.value
                    .toUpperCase()
                    .replaceAll(/[^A-Z0-9_]/g, "_"),
                }))
              }
              className="font-mono text-sm flex-1"
            />
            <Input
              placeholder="value"
              type={editing.isSecret ? "password" : "text"}
              value={editing.value}
              onChange={(e) =>
                setEditing((prev) => ({ ...prev, value: e.target.value }))
              }
              className="font-mono text-sm flex-[2]"
            />
          </div>
          <div className="flex items-center justify-between">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={editing.isSecret}
                onChange={(e) =>
                  setEditing((prev) => ({
                    ...prev,
                    isSecret: e.target.checked,
                  }))
                }
                className="rounded"
              />{" "}
              Secret (hidden in UI)
            </label>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="ghost"
                onClick={handleCancelAdd}
                disabled={createEnvVar.isPending}
              >
                <X className="h-4 w-4" />
              </Button>
              <Button
                size="sm"
                onClick={handleSaveNew}
                disabled={!editing.key.trim() || createEnvVar.isPending}
              >
                <Save className="h-4 w-4 mr-1" />
                Save
              </Button>
            </div>
          </div>
        </div>
      )}

      {envVars?.length === 0 && !isAdding && (
        <p className="text-sm text-muted-foreground text-center py-4">
          No environment variables configured. Add variables to inject them
          during deployment.
        </p>
      )}

      {envVars?.map((envVar) => (
        <div
          key={envVar.id}
          className="flex items-center gap-2 p-3 border rounded-lg"
        >
          {editingId === envVar.id ? (
            <>
              <span className="font-mono text-sm font-medium min-w-[120px]">
                {envVar.key}
              </span>
              <Input
                placeholder="new value"
                type={editing.isSecret ? "password" : "text"}
                value={editing.value}
                onChange={(e) =>
                  setEditing((prev) => ({ ...prev, value: e.target.value }))
                }
                className="font-mono text-sm flex-1"
              />
              <label className="flex items-center gap-1 text-sm whitespace-nowrap">
                <input
                  type="checkbox"
                  checked={editing.isSecret}
                  onChange={(e) =>
                    setEditing((prev) => ({
                      ...prev,
                      isSecret: e.target.checked,
                    }))
                  }
                  className="rounded"
                />{" "}
                Secret
              </label>
              <Button
                size="icon"
                variant="ghost"
                onClick={handleCancelEdit}
                disabled={updateEnvVar.isPending}
              >
                <X className="h-4 w-4" />
              </Button>
              <Button
                size="icon"
                onClick={handleSaveEdit}
                disabled={updateEnvVar.isPending}
              >
                <Save className="h-4 w-4" />
              </Button>
            </>
          ) : (
            <>
              <span className="font-mono text-sm font-medium min-w-[120px]">
                {envVar.key}
              </span>
              <span className="font-mono text-sm text-muted-foreground flex-1 truncate">
                {envVar.isSecret && showSecrets[envVar.id] && envVar.value}
                {envVar.isSecret && !showSecrets[envVar.id] && "••••••••"}
                {!envVar.isSecret && envVar.value}
              </span>
              {envVar.isSecret && (
                <Button
                  size="icon"
                  variant="ghost"
                  onClick={() => toggleShowSecret(envVar.id)}
                >
                  {showSecrets[envVar.id] ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              )}
              <Button
                size="icon"
                variant="ghost"
                onClick={() =>
                  handleStartEdit(envVar.id, envVar.value, envVar.isSecret)
                }
              >
                <Save className="h-4 w-4" />
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => handleDelete(envVar.id)}
                disabled={deleteEnvVar.isPending}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          )}
        </div>
      ))}
    </div>
  );

  if (embedded) {
    return content;
  }

  return (
    <Card>
      <CardHeader
        className="flex flex-row items-center justify-between space-y-0 cursor-pointer select-none"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
          <CardTitle>Environment Variables</CardTitle>
          <span className="text-sm text-muted-foreground font-normal">
            ({varsCount})
          </span>
        </div>
      </CardHeader>
      <CardContent className={cn(!isExpanded && "hidden")}>
        {content}
      </CardContent>
    </Card>
  );
}
