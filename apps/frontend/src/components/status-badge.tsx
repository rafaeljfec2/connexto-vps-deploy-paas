import { Badge } from "@/components/ui/badge";
import type { DeployStatus } from "@/types";

interface StatusBadgeProps {
  readonly status: DeployStatus;
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

export function StatusBadge({ status }: StatusBadgeProps) {
  const config = statusConfig[status];

  return <Badge variant={config.variant}>{config.label}</Badge>;
}
