import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { PortMappingInput, Template } from "@/types";
import { useDeployTemplate } from "../hooks/use-templates";

interface TemplateDeployDialogProps {
  readonly template: Template;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

interface PortState {
  readonly id: string;
  readonly hostPort: number;
  readonly containerPort: number;
  readonly protocol: "tcp" | "udp";
}

export function TemplateDeployDialog({
  template,
  open,
  onOpenChange,
}: TemplateDeployDialogProps) {
  const [containerName, setContainerName] = useState(template.id);
  const [envValues, setEnvValues] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    template.env?.forEach((e) => {
      initial[e.name] = e.default ?? "";
    });
    return initial;
  });
  const [portMappings, setPortMappings] = useState<PortState[]>(() =>
    (template.ports ?? []).map((port) => ({
      id: crypto.randomUUID(),
      hostPort: port,
      containerPort: port,
      protocol: "tcp" as const,
    })),
  );

  const deployTemplate = useDeployTemplate();

  const handleDeploy = () => {
    const ports: PortMappingInput[] = portMappings
      .filter((p) => p.hostPort > 0)
      .map((p) => ({
        hostPort: p.hostPort,
        containerPort: p.containerPort,
        protocol: p.protocol,
      }));

    deployTemplate.mutate(
      {
        id: template.id,
        input: {
          name: containerName,
          env: envValues,
          ports,
        },
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  const hasRequiredEnvVars = !template.env?.some(
    (e) => e.required && !envValues[e.name],
  );

  const handlePortChange = (index: number, value: number) => {
    setPortMappings((prev) =>
      prev.map((port, i) =>
        i === index
          ? {
              id: port.id,
              hostPort: value,
              containerPort: port.containerPort,
              protocol: port.protocol,
            }
          : port,
      ),
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Deploy {template.name}</DialogTitle>
          <DialogDescription>
            Configure and deploy {template.name} from the template.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label htmlFor="containerName" className="text-sm font-medium">
              Container Name
            </label>
            <Input
              value={containerName}
              onChange={(e) => setContainerName(e.target.value)}
              placeholder={template.id}
            />
          </div>

          {template.env && template.env.length > 0 && (
            <div className="space-y-3">
              <label htmlFor="envValues" className="text-sm font-medium">
                Environment Variables
              </label>
              {template.env.map((envVar) => (
                <div key={envVar.name} className="space-y-1">
                  <label className="text-sm text-muted-foreground">
                    {envVar.label}
                    {envVar.required && (
                      <span className="text-destructive ml-1">*</span>
                    )}
                  </label>
                  <Input
                    type={
                      envVar.name.toLowerCase().includes("password")
                        ? "password"
                        : "text"
                    }
                    value={envValues[envVar.name] ?? ""}
                    onChange={(e) =>
                      setEnvValues({
                        ...envValues,
                        [envVar.name]: e.target.value,
                      })
                    }
                    placeholder={envVar.default ?? envVar.description}
                  />
                  {envVar.description && (
                    <p className="text-xs text-muted-foreground">
                      {envVar.description}
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}

          {portMappings.length > 0 && (
            <div className="space-y-2">
              <label htmlFor="portMappings" className="text-sm font-medium">
                Port Mappings
              </label>
              {portMappings.map((port, index) => (
                <div key={port.id} className="flex items-center gap-2">
                  <Input
                    type="number"
                    value={port.hostPort}
                    onChange={(e) =>
                      handlePortChange(
                        index,
                        Number.parseInt(e.target.value) || 0,
                      )
                    }
                    className="w-24"
                  />
                  <span className="text-muted-foreground">:</span>
                  <span className="text-sm font-mono">
                    {port.containerPort}
                  </span>
                </div>
              ))}
              <p className="text-xs text-muted-foreground">
                Host port : Container port
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleDeploy}
            disabled={!hasRequiredEnvVars || deployTemplate.isPending}
          >
            {deployTemplate.isPending && (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            )}
            Deploy
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
