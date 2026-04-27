import type { TokenScope } from "@/services/api/tokens";

export const SCOPE_DESCRIPTIONS: Record<TokenScope, string> = {
  read: "Read-only access to apps, containers, servers, images, resources and audit logs.",
  deploy: "Trigger deploys, rollbacks and template installations.",
  "containers:write":
    "Start, stop, restart containers and run manual healthchecks.",
  "config:write":
    "Create and update environment variables, domains and SSL configuration.",
  "resources:write":
    "Create Docker networks and volumes, connect containers to networks.",
  "servers:write": "Provision, update and manage remote servers and agents.",
  destructive:
    "Required (in addition to other scopes) to delete apps, containers, images, volumes, networks or servers.",
  admin:
    "Full access. Overrides all other scope checks. Use with extreme care.",
};
