export type AuditTab = "platform" | "webhooks";

export const EVENT_TYPES = [
  { value: "app.created", label: "App Created" },
  { value: "app.deleted", label: "App Deleted" },
  { value: "app.purged", label: "App Purged" },
  { value: "deploy.started", label: "Deploy Started" },
  { value: "deploy.success", label: "Deploy Success" },
  { value: "deploy.failed", label: "Deploy Failed" },
  { value: "env.created", label: "Env Created" },
  { value: "env.updated", label: "Env Updated" },
  { value: "env.deleted", label: "Env Deleted" },
  { value: "domain.added", label: "Domain Added" },
  { value: "domain.removed", label: "Domain Removed" },
  { value: "container.started", label: "Container Started" },
  { value: "container.stopped", label: "Container Stopped" },
  { value: "container.removed", label: "Container Removed" },
  { value: "user.logged_in", label: "User Login" },
  { value: "user.logged_out", label: "User Logout" },
  { value: "image.removed", label: "Image Removed" },
  { value: "images.pruned", label: "Images Pruned" },
] as const;

export const RESOURCE_TYPES = [
  { value: "app", label: "App" },
  { value: "deployment", label: "Deployment" },
  { value: "env_var", label: "Environment" },
  { value: "domain", label: "Domain" },
  { value: "container", label: "Container" },
  { value: "user", label: "User" },
  { value: "image", label: "Image" },
] as const;
