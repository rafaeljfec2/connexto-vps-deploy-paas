export function getEventBadgeColor(eventType: string): string {
  if (
    eventType.includes("webhook.deployment_queued") ||
    eventType.includes("deployment_queued") ||
    eventType.includes("pong")
  ) {
    return "bg-green-500/20 text-green-400 border-green-500/30";
  }
  if (
    eventType.includes("webhook.invalid") ||
    eventType.includes("invalid_signature") ||
    eventType.includes("parse_error") ||
    eventType.includes("webhook.error")
  ) {
    return "bg-red-500/20 text-red-400 border-red-500/30";
  }
  if (
    eventType.includes("created") ||
    eventType.includes("added") ||
    eventType.includes("received")
  ) {
    return "bg-green-500/20 text-green-400 border-green-500/30";
  }
  if (
    eventType.includes("deleted") ||
    eventType.includes("removed") ||
    eventType.includes("purged")
  ) {
    return "bg-red-500/20 text-red-400 border-red-500/30";
  }
  if (eventType.includes("success") || eventType.includes("logged_in")) {
    return "bg-blue-500/20 text-blue-400 border-blue-500/30";
  }
  if (eventType.includes("failed") || eventType.includes("error")) {
    return "bg-orange-500/20 text-orange-400 border-orange-500/30";
  }
  if (eventType.includes("ignored")) {
    return "bg-gray-500/20 text-gray-400 border-gray-500/30";
  }
  return "bg-gray-500/20 text-gray-400 border-gray-500/30";
}
