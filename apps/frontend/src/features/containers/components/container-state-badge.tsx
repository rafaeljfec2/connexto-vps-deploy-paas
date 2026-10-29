import { Badge } from "@/components/ui/badge";

interface ContainerStateBadgeProps {
  readonly state: string;
}

const stateStyles: Record<string, string> = {
  running:
    "bg-status-success/10 text-status-success-text border-status-success/30",
  exited: "bg-status-failed/10 text-status-failed-text border-status-failed/30",
  paused:
    "bg-status-running/10 text-status-running-text border-status-running/30",
  restarting: "bg-primary/10 text-primary border-primary/30",
  dead: "bg-muted text-muted-foreground border-border",
  created: "bg-muted text-muted-foreground border-border",
};

export function ContainerStateBadge({ state }: ContainerStateBadgeProps) {
  const style = stateStyles[state] ?? stateStyles.dead;

  return (
    <Badge variant="outline" className={`${style} text-xs`}>
      {state}
    </Badge>
  );
}

interface ContainerHealthBadgeProps {
  readonly health: string;
}

const healthStyles: Record<string, string> = {
  healthy:
    "bg-status-success/10 text-status-success-text border-status-success/30",
  unhealthy:
    "bg-status-failed/10 text-status-failed-text border-status-failed/30",
  starting:
    "bg-status-running/10 text-status-running-text border-status-running/30",
};

export function ContainerHealthBadge({ health }: ContainerHealthBadgeProps) {
  if (!health || health === "none") return null;

  const style = healthStyles[health] ?? "";
  if (!style) return null;

  return (
    <Badge variant="outline" className={`${style} text-xs`}>
      {health}
    </Badge>
  );
}
