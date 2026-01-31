import { CheckCircle, Clock, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import type { MigrationContainer } from "@/types";

interface ContainerRowProps {
  readonly container: MigrationContainer;
  readonly selected: boolean;
  readonly onToggle: () => void;
}

export function ContainerRow({
  container,
  selected,
  onToggle,
}: ContainerRowProps) {
  const isRunning = container.state === "running";
  const ports = container.ports ?? [];

  return (
    <div className="flex items-center gap-4 p-3 border rounded-lg hover:bg-muted/50">
      <Checkbox checked={selected} onCheckedChange={onToggle} />
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium">{container.name}</span>
          <Badge variant={isRunning ? "default" : "secondary"}>
            {isRunning ? (
              <CheckCircle className="h-3 w-3 mr-1" />
            ) : (
              <XCircle className="h-3 w-3 mr-1" />
            )}
            {container.state}
          </Badge>
        </div>
        <div className="text-sm text-muted-foreground flex gap-4">
          <span>{container.image}</span>
          {ports.length > 0 && <span>Ports: {ports.join(", ")}</span>}
          {container.uptime && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {container.uptime}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
