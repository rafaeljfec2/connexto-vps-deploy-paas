import { Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { DeployStatus } from "@/types";

interface StatusBadgeProps {
  readonly status: DeployStatus;
  readonly size?: "sm" | "default";
  readonly className?: string;
  readonly showDot?: boolean;
}

const statusConfig: Record<
  DeployStatus,
  {
    label: string;
    variant: "success" | "running" | "failed" | "pending";
    dotClass: string;
    isAnimated?: boolean;
  }
> = {
  success: {
    label: "Deployed",
    variant: "success",
    dotClass: "bg-status-success",
  },
  running: {
    label: "Building",
    variant: "running",
    dotClass: "bg-status-running",
    isAnimated: true,
  },
  failed: {
    label: "Failed",
    variant: "failed",
    dotClass: "bg-status-failed",
  },
  pending: {
    label: "Pending",
    variant: "pending",
    dotClass: "bg-status-pending",
  },
  cancelled: {
    label: "Cancelled",
    variant: "pending",
    dotClass: "bg-muted-foreground",
  },
};

export function StatusBadge({
  status,
  size = "default",
  className,
  showDot = true,
}: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <Badge
      variant={config.variant}
      className={cn(
        "inline-flex items-center gap-1.5 font-medium border border-current/20",
        size === "sm" ? "text-xs px-2 py-0.5" : "text-xs px-2.5 py-0.5",
        className,
      )}
    >
      {showDot &&
        (config.isAnimated ? (
          <Loader2 className="h-2.5 w-2.5 animate-spin" aria-hidden="true" />
        ) : (
          <span
            className={cn("h-1.5 w-1.5 rounded-full shrink-0", config.dotClass)}
            aria-hidden="true"
          />
        ))}
      <span>{config.label}</span>
    </Badge>
  );
}
