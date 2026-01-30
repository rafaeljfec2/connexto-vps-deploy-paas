import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  CheckCircle2,
  Circle,
  ExternalLink,
  Loader2,
  Rocket,
  RotateCcw,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { api } from "@/services/api";
import { useCreateApp } from "../../hooks/use-apps";
import type { OnboardingData } from "./types";

type DeployPhase =
  | "creating_app"
  | "setting_env"
  | "setting_webhook"
  | "deploying"
  | "completed"
  | "error";

interface PhaseStatus {
  readonly phase: DeployPhase;
  readonly label: string;
  readonly status: "pending" | "loading" | "success" | "error";
  readonly errorMessage?: string;
}

interface DeployStepProps {
  readonly data: OnboardingData;
  readonly onBack: () => void;
}

const INITIAL_PHASES: readonly PhaseStatus[] = [
  { phase: "creating_app", label: "Creating application", status: "loading" },
  {
    phase: "setting_env",
    label: "Setting environment variables",
    status: "pending",
  },
  { phase: "setting_webhook", label: "Configuring webhook", status: "pending" },
  { phase: "deploying", label: "Starting first deployment", status: "pending" },
];

function getHeaderTitle(isCompleted: boolean, hasError: boolean): string {
  if (isCompleted) return "Deployment started!";
  if (hasError) return "Deployment failed";
  return "Deploying your application...";
}

function getHeaderDescription(isCompleted: boolean, hasError: boolean): string {
  if (isCompleted) {
    return "Your application is being deployed. You can track the progress on the app details page.";
  }
  if (hasError) {
    return "Something went wrong during the deployment process.";
  }
  return "Please wait while we set up your application.";
}

export function DeployStep({ data, onBack }: Readonly<DeployStepProps>) {
  const navigate = useNavigate();
  const createApp = useCreateApp();
  const [appId, setAppId] = useState<string | null>(null);
  const hasStarted = useRef(false);

  const [currentPhase, setCurrentPhase] = useState<DeployPhase>("creating_app");
  const [phases, setPhases] = useState<readonly PhaseStatus[]>(INITIAL_PHASES);
  const [error, setError] = useState<string | null>(null);

  const updatePhase = (
    phase: DeployPhase,
    status: PhaseStatus["status"],
    errorMessage?: string,
  ) => {
    setPhases((prev) =>
      prev.map((p) => (p.phase === phase ? { ...p, status, errorMessage } : p)),
    );
  };

  useEffect(() => {
    if (hasStarted.current) return;
    hasStarted.current = true;

    const runDeployment = async () => {
      let activePhase: DeployPhase = "creating_app";

      try {
        updatePhase("creating_app", "loading");
        const app = await createApp.mutateAsync({
          name: data.name,
          repositoryUrl: data.repositoryUrl,
          branch: data.branch || "main",
          workdir: data.workdir || undefined,
        });
        setAppId(app.id);
        updatePhase("creating_app", "success");

        if (data.envVars.length > 0) {
          activePhase = "setting_env";
          setCurrentPhase(activePhase);
          updatePhase("setting_env", "loading");
          await api.envVars.bulkUpsert(app.id, {
            vars: data.envVars.map(({ key, value, isSecret }) => ({
              key,
              value,
              isSecret,
            })),
          });
          updatePhase("setting_env", "success");
        } else {
          updatePhase("setting_env", "success");
        }

        activePhase = "setting_webhook";
        setCurrentPhase(activePhase);
        updatePhase("setting_webhook", "loading");
        try {
          await api.webhooks.setup(app.id);
          updatePhase("setting_webhook", "success");
        } catch {
          updatePhase("setting_webhook", "success");
        }

        activePhase = "deploying";
        setCurrentPhase(activePhase);
        updatePhase("deploying", "loading");
        await api.deployments.redeploy(app.id);
        updatePhase("deploying", "success");

        setCurrentPhase("completed");
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : "Unknown error occurred";
        setError(errorMessage);
        updatePhase(activePhase, "error", errorMessage);
        setCurrentPhase("error");
      }
    };

    runDeployment();
  }, [createApp, data]);

  const getPhaseIcon = (status: PhaseStatus["status"]) => {
    switch (status) {
      case "pending":
        return <Circle className="h-5 w-5 text-muted-foreground" />;
      case "loading":
        return <Loader2 className="h-5 w-5 text-primary animate-spin" />;
      case "success":
        return <CheckCircle2 className="h-5 w-5 text-green-500" />;
      case "error":
        return <XCircle className="h-5 w-5 text-destructive" />;
    }
  };

  const isCompleted = currentPhase === "completed";
  const hasError = currentPhase === "error";

  return (
    <Card className="border-0 shadow-none md:border md:shadow-sm">
      <CardContent className="p-0 md:p-6 space-y-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <Rocket className="h-5 w-5" />
            <h3 className="font-semibold">
              {getHeaderTitle(isCompleted, hasError)}
            </h3>
          </div>
          <p className="text-sm text-muted-foreground">
            {getHeaderDescription(isCompleted, hasError)}
          </p>
        </div>

        <div className="space-y-3">
          {phases.map((phase) => (
            <div
              key={phase.phase}
              className="flex items-center gap-3 p-3 border rounded-lg"
            >
              {getPhaseIcon(phase.status)}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">{phase.label}</p>
                {phase.errorMessage && (
                  <p className="text-xs text-destructive mt-1 truncate">
                    {phase.errorMessage}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>

        {isCompleted && (
          <div className="p-4 border rounded-lg bg-green-500/10 border-green-500/20">
            <p className="text-sm text-green-700 dark:text-green-400">
              Your application <strong>{data.name}</strong> has been created and
              the first deployment has started. It may take a few minutes to
              complete.
            </p>
          </div>
        )}

        {hasError && error && (
          <div className="p-4 border rounded-lg bg-destructive/10 border-destructive/20">
            <p className="text-sm text-destructive">{error}</p>
          </div>
        )}
      </CardContent>

      <CardFooter className="p-0 pt-6 md:p-6 md:pt-0 flex flex-col md:flex-row gap-3">
        {hasError && (
          <Button
            type="button"
            variant="outline"
            className="w-full md:w-auto"
            onClick={onBack}
          >
            <RotateCcw className="h-4 w-4 mr-2" />
            Back to Review
          </Button>
        )}

        {isCompleted && (
          <Button
            className="w-full md:w-auto md:ml-auto"
            onClick={() => navigate(`/apps/${appId}`)}
          >
            <ExternalLink className="h-4 w-4 mr-2" />
            View Application
          </Button>
        )}
      </CardFooter>
    </Card>
  );
}
