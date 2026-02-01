import { Badge } from "@/components/ui/badge";

interface ContainerStateBadgeProps {
  readonly state: string;
}

const stateStyles: Record<string, string> = {
  running: "bg-green-500/20 text-green-400 border-green-500/30",
  exited: "bg-red-500/20 text-red-400 border-red-500/30",
  paused: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  restarting: "bg-blue-500/20 text-blue-400 border-blue-500/30",
  dead: "bg-gray-500/20 text-gray-400 border-gray-500/30",
  created: "bg-gray-500/20 text-gray-400 border-gray-500/30",
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
  healthy: "bg-green-500/20 text-green-400 border-green-500/30",
  unhealthy: "bg-red-500/20 text-red-400 border-red-500/30",
  starting: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
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
