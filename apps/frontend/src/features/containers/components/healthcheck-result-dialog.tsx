import { AlertCircle, CheckCircle2, HeartPulse, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ApiError } from "@/types";
import { useRunHealthcheck } from "../hooks/use-containers";

interface HealthcheckResultDialogProps {
  readonly containerId: string;
  readonly containerName: string;
  readonly serverId?: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

interface NormalizedError {
  readonly notConfigured: boolean;
  readonly message: string;
}

function normalizeError(error: unknown): NormalizedError {
  if (error instanceof ApiError) {
    return { notConfigured: error.status === 422, message: error.message };
  }
  if (error instanceof Error) {
    return { notConfigured: false, message: error.message };
  }
  return { notConfigured: false, message: "Unknown error" };
}

export function HealthcheckResultDialog({
  containerId,
  containerName,
  serverId,
  open,
  onOpenChange,
}: HealthcheckResultDialogProps) {
  const mutation = useRunHealthcheck();

  const handleRun = () => {
    mutation.mutate({ id: containerId, serverId });
  };

  const result = mutation.data;
  const error = mutation.error ? normalizeError(mutation.error) : null;

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        onOpenChange(value);
        if (!value) {
          mutation.reset();
        }
      }}
    >
      <DialogContent className="max-w-2xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HeartPulse className="h-5 w-5" />
            Healthcheck — {containerName}
          </DialogTitle>
          <DialogDescription>
            Run the container&apos;s healthcheck command on demand and inspect
            the result.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-auto space-y-3">
          {!mutation.isPending && !result && !error && (
            <div className="text-center py-6 text-sm text-muted-foreground">
              Click &quot;Run healthcheck&quot; to execute the configured
              healthcheck command inside the container.
            </div>
          )}

          {mutation.isPending && (
            <div className="flex items-center justify-center py-8 gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
              Running healthcheck...
            </div>
          )}

          {error && (
            <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm">
              <div className="flex items-center gap-2 font-medium text-destructive mb-1">
                <AlertCircle className="h-4 w-4" />
                {error.notConfigured
                  ? "Healthcheck not configured"
                  : "Healthcheck failed"}
              </div>
              <p className="text-destructive/90 text-xs break-words">
                {error.message}
              </p>
              {error.notConfigured && (
                <p className="text-xs text-muted-foreground mt-2">
                  Add a HEALTHCHECK directive to the image or define a
                  healthcheck in your compose configuration to enable this
                  feature.
                </p>
              )}
            </div>
          )}

          {result && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 flex-wrap">
                {result.success ? (
                  <Badge
                    variant="outline"
                    className="border-emerald-500 text-emerald-600 dark:text-emerald-400 gap-1"
                  >
                    <CheckCircle2 className="h-3 w-3" />
                    Exit code {result.exitCode}
                  </Badge>
                ) : (
                  <Badge variant="destructive" className="gap-1">
                    <AlertCircle className="h-3 w-3" />
                    Exit code {result.exitCode}
                  </Badge>
                )}
                <Badge variant="secondary">{result.durationMs} ms</Badge>
              </div>

              {result.command && result.command.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1">
                    Command
                  </p>
                  <code className="text-xs bg-muted px-2 py-1 rounded block break-all font-mono">
                    {result.command.join(" ")}
                  </code>
                </div>
              )}

              <Tabs defaultValue="stdout" className="w-full">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="stdout">stdout</TabsTrigger>
                  <TabsTrigger value="stderr">stderr</TabsTrigger>
                </TabsList>
                <TabsContent value="stdout">
                  <pre className="text-xs bg-muted p-3 rounded font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-64 overflow-y-auto min-h-16">
                    {result.stdout.length > 0 ? result.stdout : "(empty)"}
                  </pre>
                </TabsContent>
                <TabsContent value="stderr">
                  <pre className="text-xs bg-muted p-3 rounded font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-64 overflow-y-auto min-h-16">
                    {result.stderr.length > 0 ? result.stderr : "(empty)"}
                  </pre>
                </TabsContent>
              </Tabs>
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button onClick={handleRun} disabled={mutation.isPending}>
            {mutation.isPending && (
              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
            )}
            {result || error ? "Run again" : "Run healthcheck"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
