export type HealthStatus =
  | "healthy"
  | "unhealthy"
  | "starting"
  | "stopped"
  | "unknown";

export function getHealthTextColor(status: string): string {
  switch (status) {
    case "healthy":
      return "text-status-success";
    case "unhealthy":
      return "text-status-failed";
    case "starting":
      return "text-status-running";
    case "stopped":
      return "text-status-pending";
    default:
      return "text-muted-foreground";
  }
}

export type SslStatus = "active" | "pending" | "no_tls" | "error";

export function getSslStatusConfig(status: SslStatus): {
  color: string;
  label: string;
} {
  switch (status) {
    case "active":
      return { color: "text-status-success", label: "Active" };
    case "pending":
      return { color: "text-status-running", label: "Pending" };
    case "no_tls":
      return { color: "text-status-pending", label: "No TLS" };
    case "error":
      return { color: "text-status-failed", label: "Error" };
  }
}
