import { useMemo, useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Eye,
  EyeOff,
  Plus,
  Save,
  Search,
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
  const [filter, setFilter] = useState("");

  const filteredVars = useMemo(() => {
    if (!envVars || !filter.trim()) return envVars;
    const term = filter.toLowerCase();
    return envVars.filter(
      (v) =>
        v.key.toLowerCase().includes(term) ||
        (!v.isSecret && v.value.toLowerCase().includes(term)),
    );
  }, [envVars, filter]);

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

  const showFilter = (envVars?.length ?? 0) >= 8;

  const content = (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        {showFilter && (
          <div className="relative flex-1">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder="Filter variables..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="h-8 pl-7 text-xs"
            />
            {filter && (
              <button
                type="button"
                onClick={() => setFilter("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        )}
        {!isAdding && (
          <Button
            size="sm"
            className="h-8 px-2 text-xs ml-auto shrink-0"
            onClick={handleAdd}
          >
            <Plus className="h-3.5 w-3.5 mr-1" />
            Add Variable
          </Button>
        )}
      </div>

      {isLoading && <p className="text-sm text-muted-foreground">Loading...</p>}

      {isAdding && (
        <div className="flex flex-col gap-2 p-2 border rounded-md bg-muted/50">
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
              className="font-mono text-xs flex-1 h-8"
            />
            <Input
              placeholder="value"
              type={editing.isSecret ? "password" : "text"}
              value={editing.value}
              onChange={(e) =>
                setEditing((prev) => ({ ...prev, value: e.target.value }))
              }
              className="font-mono text-xs flex-[2] h-8"
            />
          </div>
          <div className="flex items-center justify-between">
            <label className="flex items-center gap-2 text-xs">
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
              />
              <span>Secret (hidden in UI)</span>
            </label>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 p-0"
                onClick={handleCancelAdd}
                disabled={createEnvVar.isPending}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
              <Button
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={handleSaveNew}
                disabled={!editing.key.trim() || createEnvVar.isPending}
              >
                <Save className="h-3.5 w-3.5 mr-1" />
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

      {filter && filteredVars?.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-4">
          No variables matching &quot;{filter}&quot;
        </p>
      )}

      <div className="max-h-[400px] overflow-y-auto space-y-2 pr-1">
        {filteredVars?.map((envVar) => (
          <div
            key={envVar.id}
            className="flex items-center gap-2 p-2 border rounded-md"
          >
            {editingId === envVar.id ? (
              <>
                <span className="font-mono text-xs font-medium min-w-[110px]">
                  {envVar.key}
                </span>
                <Input
                  placeholder="new value"
                  type={editing.isSecret ? "password" : "text"}
                  value={editing.value}
                  onChange={(e) =>
                    setEditing((prev) => ({ ...prev, value: e.target.value }))
                  }
                  className="font-mono text-xs flex-1 h-8"
                />
                <label className="flex items-center gap-1 text-xs whitespace-nowrap">
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
                  />
                  <span>Secret</span>
                </label>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={handleCancelEdit}
                  disabled={updateEnvVar.isPending}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
                <Button
                  size="icon"
                  className="h-7 w-7"
                  onClick={handleSaveEdit}
                  disabled={updateEnvVar.isPending}
                >
                  <Save className="h-3.5 w-3.5" />
                </Button>
              </>
            ) : (
              <>
                <span className="font-mono text-xs font-medium min-w-[110px]">
                  {envVar.key}
                </span>
                <span className="font-mono text-xs text-muted-foreground flex-1 truncate">
                  {envVar.isSecret && showSecrets[envVar.id] && envVar.value}
                  {envVar.isSecret && !showSecrets[envVar.id] && "••••••••"}
                  {!envVar.isSecret && envVar.value}
                </span>
                {envVar.isSecret && (
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7"
                    onClick={() => toggleShowSecret(envVar.id)}
                  >
                    {showSecrets[envVar.id] ? (
                      <EyeOff className="h-3.5 w-3.5" />
                    ) : (
                      <Eye className="h-3.5 w-3.5" />
                    )}
                  </Button>
                )}
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={() =>
                    handleStartEdit(envVar.id, envVar.value, envVar.isSecret)
                  }
                >
                  <Save className="h-3.5 w-3.5" />
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7 text-destructive hover:text-destructive"
                  onClick={() => handleDelete(envVar.id)}
                  disabled={deleteEnvVar.isPending}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </>
            )}
          </div>
        ))}
      </div>
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
            &mdash; {varsCount} variables configured
          </span>
        </div>
      </CardHeader>
      <CardContent className={cn(!isExpanded && "hidden")}>
        {content}
      </CardContent>
    </Card>
  );
}
