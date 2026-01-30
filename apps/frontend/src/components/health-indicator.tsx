import { Activity, AlertCircle, Circle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { HealthStatus } from "@/types";

interface HealthIndicatorProps {
  readonly health: HealthStatus | null | undefined;
  readonly showLabel?: boolean;
  readonly size?: "sm" | "md" | "lg";
}

type HealthState = "healthy" | "unhealthy" | "starting" | "offline" | "unknown";

interface HealthConfig {
  readonly label: string;
  readonly color: string;
  readonly bgColor: string;
  readonly icon: typeof Circle;
  readonly animate?: boolean;
}

const healthConfig: Record<HealthState, HealthConfig> = {
  healthy: {
    label: "Healthy",
    color: "text-status-success",
    bgColor: "bg-status-success",
    icon: Activity,
  },
  unhealthy: {
    label: "Unhealthy",
    color: "text-status-failed",
    bgColor: "bg-status-failed",
    icon: AlertCircle,
  },
  starting: {
    label: "Starting",
    color: "text-status-running",
    bgColor: "bg-status-running",
    icon: Loader2,
    animate: true,
  },
  offline: {
    label: "Offline",
    color: "text-status-pending",
    bgColor: "bg-status-pending",
    icon: Circle,
  },
  unknown: {
    label: "Unknown",
    color: "text-muted-foreground",
    bgColor: "bg-muted-foreground",
    icon: Circle,
  },
};

const sizeConfig = {
  sm: {
    container: "gap-1",
    icon: "h-3 w-3",
    dot: "h-2 w-2",
    text: "text-xs",
  },
  md: {
    container: "gap-1.5",
    icon: "h-4 w-4",
    dot: "h-2.5 w-2.5",
    text: "text-sm",
  },
  lg: {
    container: "gap-2",
    icon: "h-5 w-5",
    dot: "h-3 w-3",
    text: "text-base",
  },
};

function getHealthState(health: HealthStatus | null | undefined): HealthState {
  if (!health) return "unknown";

  if (health.status === "not_found" || health.status === "exited") {
    return "offline";
  }

  if (health.status === "restarting") {
    return "starting";
  }

  if (health.status === "running") {
    if (health.health === "healthy") return "healthy";
    if (health.health === "unhealthy") return "unhealthy";
    if (health.health === "starting") return "starting";
    return "healthy";
  }

  if (health.status === "paused") {
    return "offline";
  }

  return "unknown";
}

export function HealthIndicator({
  health,
  showLabel = false,
  size = "md",
}: HealthIndicatorProps) {
  const state = getHealthState(health);
  const config = healthConfig[state];
  const sizes = sizeConfig[size];
  const Icon = config.icon;

  return (
    <div className={cn("inline-flex items-center", sizes.container)}>
      <div className="relative flex items-center justify-center">
        {state === "healthy" && (
          <span
            className={cn(
              "absolute inline-flex rounded-full opacity-75 animate-ping",
              sizes.dot,
              config.bgColor,
            )}
          />
        )}
        <Icon
          className={cn(
            sizes.icon,
            config.color,
            config.animate && "animate-spin",
          )}
        />
      </div>
      {showLabel && (
        <span className={cn(sizes.text, config.color, "font-medium")}>
          {config.label}
        </span>
      )}
    </div>
  );
}

interface HealthDetailProps {
  readonly health: HealthStatus | null | undefined;
}

export function HealthDetail({ health }: HealthDetailProps) {
  if (!health) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground">
        <Circle className="h-4 w-4" />
        <span className="text-sm">No health data available</span>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <HealthIndicator health={health} showLabel size="md" />
      </div>
      {health.uptime && (
        <div className="text-sm text-muted-foreground">
          Uptime: <span className="font-mono">{health.uptime}</span>
        </div>
      )}
      {health.startedAt && (
        <div className="text-sm text-muted-foreground">
          Started:{" "}
          <span className="font-mono">
            {new Date(health.startedAt).toLocaleString()}
          </span>
        </div>
      )}
    </div>
  );
}
