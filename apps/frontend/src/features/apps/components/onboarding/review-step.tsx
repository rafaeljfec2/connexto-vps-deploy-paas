import { useState } from "react";
import {
  Check,
  Eye,
  EyeOff,
  FolderGit2,
  GitBranch,
  Link,
  Monitor,
  Package,
  Server,
  Variable,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { useServers } from "@/features/servers/hooks/use-servers";
import type { StepProps } from "./types";

interface ReviewItemProps {
  readonly icon: React.ReactNode;
  readonly label: string;
  readonly value: string;
  readonly empty?: string;
}

function ReviewItem({ icon, label, value, empty }: Readonly<ReviewItemProps>) {
  const displayValue = value || (
    <span className="text-muted-foreground italic">{empty}</span>
  );

  return (
    <div className="flex items-start gap-3 p-3 border rounded-lg">
      <div className="text-primary mt-0.5">{icon}</div>
      <div className="min-w-0 flex-1">
        <p className="text-sm text-muted-foreground">{label}</p>
        <p className="font-medium truncate">{displayValue}</p>
      </div>
      <Check className="h-4 w-4 text-green-500 mt-1" />
    </div>
  );
}

export function ReviewStep({ data, onNext, onBack }: Readonly<StepProps>) {
  const { data: servers } = useServers();
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});

  const toggleShowSecret = (localId: string) => {
    setShowSecrets((prev) => ({ ...prev, [localId]: !prev[localId] }));
  };

  const getEnvVarDisplayValue = (
    isSecret: boolean,
    localId: string,
    value: string,
  ): string => {
    if (!isSecret) return value;
    return showSecrets[localId] ? value : "••••••••";
  };

  return (
    <Card className="border-0 shadow-none md:border md:shadow-sm">
      <CardContent className="p-0 md:p-6 space-y-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-primary">
            <Check className="h-5 w-5" />
            <h3 className="font-semibold">Review your configuration</h3>
          </div>
          <p className="text-sm text-muted-foreground">
            Please review your settings before creating the application.
          </p>
        </div>

        <div className="space-y-4">
          <div>
            <h4 className="text-sm font-medium mb-3 flex items-center gap-2">
              <FolderGit2 className="h-4 w-4" />
              Repository Settings
            </h4>
            <div className="space-y-2">
              <ReviewItem
                icon={<Package className="h-4 w-4" />}
                label="Application Name"
                value={data.name}
              />
              <ReviewItem
                icon={<Link className="h-4 w-4" />}
                label="Repository"
                value={data.repositoryUrl}
              />
              <ReviewItem
                icon={<GitBranch className="h-4 w-4" />}
                label="Branch"
                value={data.branch}
                empty="main"
              />
              <ReviewItem
                icon={<FolderGit2 className="h-4 w-4" />}
                label="Working Directory"
                value={data.workdir}
                empty="root"
              />
            </div>
          </div>

          <div>
            <h4 className="text-sm font-medium mb-3 flex items-center gap-2">
              <Server className="h-4 w-4" />
              Deploy Target
            </h4>
            <div className="space-y-2">
              {data.serverId ? (
                <ReviewItem
                  icon={<Server className="h-4 w-4" />}
                  label="Remote Server"
                  value={
                    servers?.find((s) => s.id === data.serverId)?.name ??
                    data.serverId
                  }
                />
              ) : (
                <ReviewItem
                  icon={<Monitor className="h-4 w-4" />}
                  label="Local Server"
                  value="This machine (default)"
                />
              )}
            </div>
          </div>

          <div>
            <h4 className="text-sm font-medium mb-3 flex items-center gap-2">
              <Variable className="h-4 w-4" />
              Environment Variables ({data.envVars.length})
            </h4>

            {data.envVars.length === 0 ? (
              <p className="text-sm text-muted-foreground p-3 border rounded-lg bg-muted/30">
                No environment variables configured. You can add them after
                deployment.
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
                      {getEnvVarDisplayValue(
                        envVar.isSecret,
                        envVar.localId,
                        envVar.value,
                      )}
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
                    <Check className="h-4 w-4 text-green-500" />
                  </div>
                ))}
              </div>
            )}
          </div>
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
          Create & Deploy
        </Button>
      </CardFooter>
    </Card>
  );
}
