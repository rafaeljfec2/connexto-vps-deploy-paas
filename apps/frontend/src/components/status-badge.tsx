import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { DeployStatus } from "@/types";

interface StatusBadgeProps {
  readonly status: DeployStatus;
  readonly size?: "sm" | "default";
}

const statusConfig: Record<
  DeployStatus,
  { label: string; variant: "success" | "running" | "failed" | "pending" }
> = {
  success: { label: "Deployed", variant: "success" },
  running: { label: "Building", variant: "running" },
  failed: { label: "Failed", variant: "failed" },
  pending: { label: "Pending", variant: "pending" },
  cancelled: { label: "Cancelled", variant: "pending" },
};

export function StatusBadge({ status, size = "default" }: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <Badge
      variant={config.variant}
      className={cn(size === "sm" && "text-xs px-1.5 py-0")}
    >
      {config.label}
    </Badge>
  );
}
