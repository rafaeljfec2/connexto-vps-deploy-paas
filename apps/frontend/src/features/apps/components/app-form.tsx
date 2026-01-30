import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Folder,
  GitBranch,
  Key,
  Rocket,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/form-field";
import { cn } from "@/lib/utils";
import { useCreateApp } from "../hooks/use-apps";
import { useBulkUpsertEnvVars } from "../hooks/use-env-vars";

interface EnvVarInput {
  readonly id: string;
  readonly key: string;
  readonly value: string;
  readonly isSecret: boolean;
}

const STEPS = [
  { id: 1, title: "Repository", icon: GitBranch },
  { id: 2, title: "Environment", icon: Key },
  { id: 3, title: "Deploy", icon: Rocket },
] as const;

export function AppForm() {
  const navigate = useNavigate();
  const createApp = useCreateApp();
  const [createdAppId, setCreatedAppId] = useState<string | null>(null);
  const bulkUpsertEnvVars = useBulkUpsertEnvVars(createdAppId ?? "");

  const [currentStep, setCurrentStep] = useState(1);
  const [name, setName] = useState("");
  const [repositoryUrl, setRepositoryUrl] = useState("");
  const [branch, setBranch] = useState("main");
  const [workdir, setWorkdir] = useState("");
  const [envVars, setEnvVars] = useState<EnvVarInput[]>([]);
  const [newEnvKey, setNewEnvKey] = useState("");
  const [newEnvValue, setNewEnvValue] = useState("");
  const [newEnvIsSecret, setNewEnvIsSecret] = useState(false);

  const isStep1Valid = name.length >= 2 && repositoryUrl.includes("github.com");

  const handleAddEnvVar = () => {
    if (!newEnvKey.trim()) return;

    setEnvVars((prev) => [
      ...prev,
      {
        id: crypto.randomUUID(),
        key: newEnvKey.trim().toUpperCase(),
        value: newEnvValue,
        isSecret: newEnvIsSecret,
      },
    ]);
    setNewEnvKey("");
    setNewEnvValue("");
    setNewEnvIsSecret(false);
  };

  const handleRemoveEnvVar = (id: string) => {
    setEnvVars((prev) => prev.filter((v) => v.id !== id));
  };

  const handleNext = () => {
    if (currentStep < 3) {
      setCurrentStep((prev) => prev + 1);
    }
  };

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep((prev) => prev - 1);
    }
  };

  const handleDeploy = async () => {
    try {
      const app = await createApp.mutateAsync({
        name,
        repositoryUrl,
        branch,
        workdir: workdir || undefined,
      });

      setCreatedAppId(app.id);

      if (envVars.length > 0) {
        const varsToSave = envVars.map(({ key, value, isSecret }) => ({
          key,
          value,
          isSecret,
        }));
        await bulkUpsertEnvVars.mutateAsync(varsToSave);
      }

      navigate(`/apps/${app.id}`);
    } catch (error) {
      console.error("Failed to create app:", error);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-8">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold">Deploy Your Application</h1>
        <p className="text-muted-foreground">
          Connect your repository and start deploying in minutes
        </p>
      </div>

      <div className="flex items-center justify-center gap-2">
        {STEPS.map((step, index) => (
          <div key={step.id} className="flex items-center">
            <div
              className={cn(
                "flex items-center justify-center w-10 h-10 rounded-full border-2 transition-colors",
                currentStep === step.id &&
                  "border-primary bg-primary text-primary-foreground",
                currentStep > step.id &&
                  "border-primary bg-primary/10 text-primary",
                currentStep < step.id &&
                  "border-muted-foreground/30 text-muted-foreground",
              )}
            >
              {currentStep > step.id && <Check className="h-5 w-5" />}
              {currentStep <= step.id && <step.icon className="h-5 w-5" />}
            </div>
            <span
              className={cn(
                "ml-2 text-sm font-medium hidden sm:block",
                currentStep >= step.id
                  ? "text-foreground"
                  : "text-muted-foreground",
              )}
            >
              {step.title}
            </span>
            {index < STEPS.length - 1 && (
              <div
                className={cn(
                  "w-12 h-0.5 mx-3",
                  currentStep > step.id
                    ? "bg-primary"
                    : "bg-muted-foreground/30",
                )}
              />
            )}
          </div>
        ))}
      </div>

      <Card>
        <CardContent className="pt-6">
          {currentStep === 1 && (
            <div className="space-y-6">
              <div className="space-y-2">
                <h2 className="text-xl font-semibold">Connect Repository</h2>
                <p className="text-sm text-muted-foreground">
                  Your repository must contain a{" "}
                  <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                    Dockerfile
                  </code>{" "}
                  and{" "}
                  <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                    paasdeploy.json
                  </code>
                </p>
              </div>

              <div className="space-y-4">
                <FormField
                  label="Application Name"
                  htmlFor="name"
                  helper="Used for container name and routing"
                  required
                >
                  <Input
                    id="name"
                    placeholder="my-awesome-app"
                    value={name}
                    onChange={(e) =>
                      setName(
                        e.target.value
                          .toLowerCase()
                          .replaceAll(/[^a-z0-9-]/g, "-"),
                      )
                    }
                    required
                    minLength={2}
                    maxLength={63}
                  />
                </FormField>

                <FormField
                  label="GitHub Repository"
                  htmlFor="repository"
                  required
                >
                  <Input
                    id="repository"
                    placeholder="https://github.com/owner/repo"
                    value={repositoryUrl}
                    onChange={(e) => setRepositoryUrl(e.target.value)}
                    required
                  />
                </FormField>

                <div className="grid grid-cols-2 gap-4">
                  <FormField label="Branch" htmlFor="branch">
                    <div className="relative">
                      <GitBranch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        id="branch"
                        placeholder="main"
                        value={branch}
                        onChange={(e) => setBranch(e.target.value)}
                        className="pl-9"
                      />
                    </div>
                  </FormField>

                  <FormField
                    label="Working Directory"
                    htmlFor="workdir"
                    helper="For monorepos"
                  >
                    <div className="relative">
                      <Folder className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        id="workdir"
                        placeholder="apps/api"
                        value={workdir}
                        onChange={(e) => setWorkdir(e.target.value)}
                        className="pl-9"
                      />
                    </div>
                  </FormField>
                </div>
              </div>
            </div>
          )}

          {currentStep === 2 && (
            <div className="space-y-6">
              <div className="space-y-2">
                <h2 className="text-xl font-semibold">Environment Variables</h2>
                <p className="text-sm text-muted-foreground">
                  Add secrets and configuration. You can also add them later.
                </p>
              </div>

              <div className="space-y-3">
                <div className="flex gap-2">
                  <Input
                    placeholder="KEY_NAME"
                    value={newEnvKey}
                    onChange={(e) =>
                      setNewEnvKey(
                        e.target.value
                          .toUpperCase()
                          .replaceAll(/[^A-Z0-9_]/g, "_"),
                      )
                    }
                    className="font-mono text-sm"
                  />
                  <Input
                    placeholder="value"
                    type={newEnvIsSecret ? "password" : "text"}
                    value={newEnvValue}
                    onChange={(e) => setNewEnvValue(e.target.value)}
                    className="font-mono text-sm"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleAddEnvVar}
                    disabled={!newEnvKey.trim()}
                  >
                    Add
                  </Button>
                </div>

                <label className="flex items-center gap-2 text-sm text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={newEnvIsSecret}
                    onChange={(e) => setNewEnvIsSecret(e.target.checked)}
                    className="rounded"
                  />
                  <span>Mark as secret (value hidden in UI)</span>
                </label>

                {envVars.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <Key className="h-12 w-12 mx-auto mb-3 opacity-20" />
                    <p className="text-sm">No variables added yet</p>
                    <p className="text-xs">
                      Add database URLs, API keys, and other secrets
                    </p>
                  </div>
                ) : (
                  <div className="space-y-2 mt-4">
                    {envVars.map((envVar) => (
                      <div
                        key={envVar.id}
                        className="flex items-center gap-2 p-2 bg-muted rounded-lg"
                      >
                        <span className="font-mono text-sm font-medium flex-1">
                          {envVar.key}
                        </span>
                        <span className="font-mono text-sm text-muted-foreground flex-1 truncate">
                          {envVar.isSecret ? "••••••••" : envVar.value}
                        </span>
                        {envVar.isSecret && (
                          <span className="text-xs bg-yellow-500/10 text-yellow-600 px-2 py-0.5 rounded">
                            secret
                          </span>
                        )}
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveEnvVar(envVar.id)}
                          className="text-destructive hover:text-destructive"
                        >
                          Remove
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-6">
              <div className="space-y-2">
                <h2 className="text-xl font-semibold">Ready to Deploy</h2>
                <p className="text-sm text-muted-foreground">
                  Review your configuration and start the first deployment
                </p>
              </div>

              <div className="space-y-4">
                <div className="p-4 bg-muted rounded-lg space-y-3">
                  <h3 className="font-medium">Application</h3>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <span className="text-muted-foreground">Name:</span>
                    <span className="font-mono">{name}</span>
                    <span className="text-muted-foreground">Repository:</span>
                    <span className="font-mono truncate">{repositoryUrl}</span>
                    <span className="text-muted-foreground">Branch:</span>
                    <span className="font-mono">{branch}</span>
                    {workdir && (
                      <>
                        <span className="text-muted-foreground">Workdir:</span>
                        <span className="font-mono">{workdir}</span>
                      </>
                    )}
                  </div>
                </div>

                <div className="p-4 bg-muted rounded-lg space-y-3">
                  <h3 className="font-medium">
                    Environment Variables ({envVars.length})
                  </h3>
                  {envVars.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      No variables configured
                    </p>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      {envVars.map((envVar) => (
                        <span
                          key={envVar.id}
                          className={cn(
                            "font-mono text-xs px-2 py-1 rounded",
                            envVar.isSecret
                              ? "bg-yellow-500/10 text-yellow-600"
                              : "bg-primary/10 text-primary",
                          )}
                        >
                          {envVar.key}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                <div className="p-4 border border-primary/20 bg-primary/5 rounded-lg">
                  <div className="flex items-start gap-3">
                    <Rocket className="h-5 w-5 text-primary mt-0.5" />
                    <div>
                      <h3 className="font-medium">What happens next?</h3>
                      <ul className="text-sm text-muted-foreground mt-1 space-y-1">
                        <li>1. Repository will be cloned</li>
                        <li>2. Docker image will be built</li>
                        <li>3. Container will be deployed</li>
                        <li>4. Health check will verify the deployment</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex justify-between">
        <Button
          type="button"
          variant="outline"
          onClick={currentStep === 1 ? () => navigate("/") : handleBack}
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          {currentStep === 1 ? "Cancel" : "Back"}
        </Button>

        {currentStep < 3 ? (
          <Button
            type="button"
            onClick={handleNext}
            disabled={currentStep === 1 && !isStep1Valid}
          >
            {currentStep === 2 ? "Review" : "Next"}
            <ArrowRight className="h-4 w-4 ml-2" />
          </Button>
        ) : (
          <Button
            onClick={handleDeploy}
            disabled={createApp.isPending || bulkUpsertEnvVars.isPending}
          >
            <Rocket className="h-4 w-4 mr-2" />
            {createApp.isPending ? "Creating..." : "Deploy Application"}
          </Button>
        )}
      </div>
    </div>
  );
}
