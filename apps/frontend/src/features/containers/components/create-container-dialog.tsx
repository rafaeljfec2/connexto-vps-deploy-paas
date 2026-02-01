import { useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { CreateContainerInput, PortMappingInput } from "@/types";
import { useCreateContainer } from "../hooks/use-containers";

interface PortMappingWithId extends PortMappingInput {
  readonly id: string;
}

interface EnvVarWithId {
  readonly id: string;
  readonly key: string;
  readonly value: string;
}

interface CreateContainerDialogProps {
  readonly trigger?: React.ReactNode;
}

export function CreateContainerDialog({ trigger }: CreateContainerDialogProps) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [image, setImage] = useState("");
  const [ports, setPorts] = useState<PortMappingWithId[]>([]);
  const [envVars, setEnvVars] = useState<EnvVarWithId[]>([]);
  const [restartPolicy, setRestartPolicy] = useState<string>("unless-stopped");

  const createContainer = useCreateContainer();

  const handleAddPort = () => {
    setPorts([
      ...ports,
      {
        id: crypto.randomUUID(),
        hostPort: 0,
        containerPort: 0,
        protocol: "tcp",
      },
    ]);
  };

  const handleRemovePort = (index: number) => {
    setPorts(ports.filter((_, i) => i !== index));
  };

  const handlePortChange = (
    index: number,
    field: "hostPort" | "containerPort" | "protocol",
    value: string | number,
  ) => {
    setPorts((prev) =>
      prev.map((port, i) => {
        if (i !== index) return port;
        if (field === "hostPort") {
          return {
            id: port.id,
            hostPort: value as number,
            containerPort: port.containerPort,
            protocol: port.protocol,
          };
        }
        if (field === "containerPort") {
          return {
            id: port.id,
            hostPort: port.hostPort,
            containerPort: value as number,
            protocol: port.protocol,
          };
        }
        return {
          id: port.id,
          hostPort: port.hostPort,
          containerPort: port.containerPort,
          protocol: value as "tcp" | "udp",
        };
      }),
    );
  };

  const handleAddEnvVar = () => {
    setEnvVars([...envVars, { id: crypto.randomUUID(), key: "", value: "" }]);
  };

  const handleRemoveEnvVar = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index));
  };

  const handleEnvVarChange = (
    index: number,
    field: "key" | "value",
    value: string,
  ) => {
    setEnvVars((prev) =>
      prev.map((env, i) => {
        if (i !== index) return env;
        if (field === "key") {
          return { id: env.id, key: value, value: env.value };
        }
        return { id: env.id, key: env.key, value: value };
      }),
    );
  };

  const handleSubmit = () => {
    const input: CreateContainerInput = {
      name: name || undefined!,
      image,
      ports: ports.filter((p) => p.hostPort > 0 && p.containerPort > 0),
      env: envVars.reduce(
        (acc, { key, value }) => {
          if (key) acc[key] = value;
          return acc;
        },
        {} as Record<string, string>,
      ),
      restartPolicy: restartPolicy as CreateContainerInput["restartPolicy"],
    };

    createContainer.mutate(input, {
      onSuccess: () => {
        setOpen(false);
        resetForm();
      },
    });
  };

  const resetForm = () => {
    setName("");
    setImage("");
    setPorts([]);
    setEnvVars([]);
    setRestartPolicy("unless-stopped");
  };

  const isValid = image.trim() !== "";

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Container
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Container</DialogTitle>
          <DialogDescription>
            Create a new Docker container from an image.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          <div className="space-y-2">
            <label htmlFor="name" className="text-sm font-medium">
              Container Name
            </label>
            <Input
              placeholder="my-container"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Optional. A random name will be assigned if not provided.
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="image" className="text-sm font-medium">
              Image <span className="text-destructive">*</span>
            </label>
            <Input
              placeholder="nginx:latest"
              value={image}
              onChange={(e) => setImage(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Docker image name with optional tag (e.g., postgres:16-alpine)
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="restartPolicy" className="text-sm font-medium">
              Restart Policy
            </label>
            <Select value={restartPolicy} onValueChange={setRestartPolicy}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="no">No</SelectItem>
                <SelectItem value="always">Always</SelectItem>
                <SelectItem value="unless-stopped">Unless Stopped</SelectItem>
                <SelectItem value="on-failure">On Failure</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Port Mappings</span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddPort}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add Port
              </Button>
            </div>
            {ports.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No port mappings configured.
              </p>
            ) : (
              <div className="space-y-2">
                {ports.map((port, index) => (
                  <div key={port.id} className="flex items-center gap-2">
                    <Input
                      type="number"
                      placeholder="Host port"
                      value={port.hostPort || ""}
                      onChange={(e) =>
                        handlePortChange(
                          index,
                          "hostPort",
                          Number.parseInt(e.target.value) || 0,
                        )
                      }
                      className="w-24"
                    />
                    <span className="text-muted-foreground">:</span>
                    <Input
                      type="number"
                      placeholder="Container port"
                      value={port.containerPort || ""}
                      onChange={(e) =>
                        handlePortChange(
                          index,
                          "containerPort",
                          Number.parseInt(e.target.value) || 0,
                        )
                      }
                      className="w-24"
                    />
                    <Select
                      value={port.protocol ?? "tcp"}
                      onValueChange={(v) =>
                        handlePortChange(index, "protocol", v)
                      }
                    >
                      <SelectTrigger className="w-20">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="tcp">TCP</SelectItem>
                        <SelectItem value="udp">UDP</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemovePort(index)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Environment Variables</span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddEnvVar}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add Variable
              </Button>
            </div>
            {envVars.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No environment variables configured.
              </p>
            ) : (
              <div className="space-y-2">
                {envVars.map((env, index) => (
                  <div key={env.id} className="flex items-center gap-2">
                    <Input
                      placeholder="KEY"
                      value={env.key}
                      onChange={(e) =>
                        handleEnvVarChange(index, "key", e.target.value)
                      }
                      className="w-1/3"
                    />
                    <span className="text-muted-foreground">=</span>
                    <Input
                      placeholder="value"
                      value={env.value}
                      onChange={(e) =>
                        handleEnvVarChange(index, "value", e.target.value)
                      }
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveEnvVar(index)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!isValid || createContainer.isPending}
          >
            {createContainer.isPending && (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            )}
            Create Container
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
