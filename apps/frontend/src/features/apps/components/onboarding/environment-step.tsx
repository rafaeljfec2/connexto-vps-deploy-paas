import { useState } from "react";
import { Eye, EyeOff, FileText, Plus, Trash2, Variable } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { LocalEnvVar, StepProps } from "./types";

interface EditingEnvVar {
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
}

const INITIAL_VAR: EditingEnvVar = {
  key: "",
  value: "",
  isSecret: false,
};

export function EnvironmentStep({ data, onUpdate, onNext, onBack }: StepProps) {
  const [newVar, setNewVar] = useState<EditingEnvVar>(INITIAL_VAR);
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
  const [showPasteMode, setShowPasteMode] = useState(false);
  const [pasteContent, setPasteContent] = useState("");

  const handleAddVar = () => {
    if (!newVar.key.trim()) return;

    const localId = crypto.randomUUID();
    const envVar: LocalEnvVar = {
      localId,
      key: newVar.key.trim().toUpperCase(),
      value: newVar.value,
      isSecret: newVar.isSecret,
    };

    onUpdate({ envVars: [...data.envVars, envVar] });
    setNewVar(INITIAL_VAR);
  };

  const handleRemoveVar = (localId: string) => {
    onUpdate({ envVars: data.envVars.filter((v) => v.localId !== localId) });
  };

  const toggleShowSecret = (localId: string) => {
    setShowSecrets((prev) => ({ ...prev, [localId]: !prev[localId] }));
  };

  const handlePasteEnvFile = () => {
    const lines = pasteContent.split("\n");
    const newVars: LocalEnvVar[] = [];

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;

      const eqIndex = trimmed.indexOf("=");
      if (eqIndex === -1) continue;

      const key = trimmed.slice(0, eqIndex).trim().toUpperCase();
      let value = trimmed.slice(eqIndex + 1).trim();

      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }

      if (key && !data.envVars.some((v) => v.key === key)) {
        newVars.push({
          localId: crypto.randomUUID(),
          key,
          value,
          isSecret:
            key.toLowerCase().includes("secret") ||
            key.toLowerCase().includes("password") ||
            key.toLowerCase().includes("token") ||
            key.toLowerCase().includes("key"),
        });
      }
    }

    if (newVars.length > 0) {
      onUpdate({ envVars: [...data.envVars, ...newVars] });
    }

    setPasteContent("");
    setShowPasteMode(false);
  };

  return (
    <Card className="border-0 shadow-none md:border md:shadow-sm">
      <CardContent className="p-0 md:p-6 space-y-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <Variable className="h-5 w-5" />
            <h3 className="font-semibold">Environment Variables</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Add environment variables that will be injected during deployment.
            You can skip this step and add them later.
          </p>
        </div>

        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setShowPasteMode(!showPasteMode)}
          >
            <FileText className="h-4 w-4 mr-2" />
            Paste .env file
          </Button>
        </div>

        {showPasteMode && (
          <div className="space-y-2 p-4 border rounded-lg bg-muted/50">
            <textarea
              placeholder="Paste your .env file content here..."
              value={pasteContent}
              onChange={(e) => setPasteContent(e.target.value)}
              className="w-full h-32 p-3 text-sm font-mono bg-background border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <div className="flex gap-2 justify-end">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowPasteMode(false);
                  setPasteContent("");
                }}
              >
                Cancel
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={handlePasteEnvFile}
                disabled={!pasteContent.trim()}
              >
                Import Variables
              </Button>
            </div>
          </div>
        )}

        <div className="space-y-3">
          <div className="flex flex-col md:flex-row gap-2 p-3 border rounded-lg bg-muted/30">
            <Input
              placeholder="KEY_NAME"
              value={newVar.key}
              onChange={(e) =>
                setNewVar((prev) => ({
                  ...prev,
                  key: e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, "_"),
                }))
              }
              className="font-mono text-sm md:w-1/3"
            />
            <div className="relative flex-1">
              <Input
                placeholder="value"
                type={newVar.isSecret ? "password" : "text"}
                value={newVar.value}
                onChange={(e) =>
                  setNewVar((prev) => ({ ...prev, value: e.target.value }))
                }
                className="font-mono text-sm pr-10"
              />
              <button
                type="button"
                onClick={() =>
                  setNewVar((prev) => ({ ...prev, isSecret: !prev.isSecret }))
                }
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {newVar.isSecret ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
            <Button
              type="button"
              size="icon"
              onClick={handleAddVar}
              disabled={!newVar.key.trim()}
              className="shrink-0"
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          {data.envVars.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">
              No environment variables added yet.
            </p>
          ) : (
            <div className="space-y-2">
              {data.envVars.map((envVar) => (
                <div
                  key={envVar.localId}
                  className="flex items-center gap-2 p-3 border rounded-lg"
                >
                  <span className="font-mono text-sm font-medium min-w-[100px] md:min-w-[140px] truncate">
                    {envVar.key}
                  </span>
                  <span className="font-mono text-sm text-muted-foreground flex-1 truncate">
                    {envVar.isSecret
                      ? showSecrets[envVar.localId]
                        ? envVar.value
                        : "••••••••"
                      : envVar.value}
                  </span>
                  {envVar.isSecret && (
                    <Button
                      type="button"
                      size="icon"
                      variant="ghost"
                      onClick={() => toggleShowSecret(envVar.localId)}
                    >
                      {showSecrets[envVar.localId] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  )}
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="text-destructive hover:text-destructive shrink-0"
                    onClick={() => handleRemoveVar(envVar.localId)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>

      <CardFooter className="p-0 pt-6 md:p-6 md:pt-0 flex flex-col md:flex-row gap-3">
        <Button
          type="button"
          variant="outline"
          className="w-full md:w-auto"
          onClick={onBack}
        >
          Back
        </Button>
        <Button className="w-full md:w-auto md:ml-auto" onClick={onNext}>
          Continue to Review
        </Button>
      </CardFooter>
    </Card>
  );
}
