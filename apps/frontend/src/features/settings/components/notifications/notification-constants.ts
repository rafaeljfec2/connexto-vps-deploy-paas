import type { NotificationChannelType, NotificationEventType } from "@/types";

export const CHANNEL_TYPES: {
  readonly value: NotificationChannelType;
  readonly label: string;
}[] = [
  { value: "slack", label: "Slack" },
  { value: "discord", label: "Discord" },
  { value: "email", label: "Email" },
];

export const EVENT_TYPES: {
  readonly value: NotificationEventType;
  readonly label: string;
}[] = [
  { value: "deploy_running", label: "Deploy started" },
  { value: "deploy_success", label: "Deploy success" },
  { value: "deploy_failed", label: "Deploy failed" },
  { value: "container_down", label: "Container down" },
  { value: "health_unhealthy", label: "Health unhealthy" },
];

export function getEventTypeLabel(eventType: string): string {
  const found = EVENT_TYPES.find((e) => e.value === eventType);
  return found?.label ?? eventType;
}
