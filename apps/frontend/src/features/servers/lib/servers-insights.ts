import type { Server, ServerStatus } from "@/types";

export interface ServersSummary {
  readonly total: number;
  readonly online: number;
  readonly offline: number;
  readonly pending: number;
  readonly provisioning: number;
  readonly error: number;
  readonly outdatedAgents: number;
  readonly latestAgentVersion: string | null;
}

export interface ServersNextBestAction {
  readonly severity: "destructive" | "warning";
  readonly message: string;
  readonly ctaLabel: string;
  readonly ctaHref: string;
}

export function summarizeServers(servers: readonly Server[]): ServersSummary {
  const counts: Record<ServerStatus, number> = {
    pending: 0,
    provisioning: 0,
    online: 0,
    offline: 0,
    error: 0,
  };

  let outdatedAgents = 0;
  let latestAgentVersion: string | null = null;

  for (const server of servers) {
    counts[server.status] += 1;

    if (
      server.agentVersion != null &&
      server.agentVersion !== server.latestAgentVersion
    ) {
      outdatedAgents += 1;
    }

    if (latestAgentVersion == null && server.latestAgentVersion) {
      latestAgentVersion = server.latestAgentVersion;
    }
  }

  return {
    total: servers.length,
    online: counts.online,
    offline: counts.offline,
    pending: counts.pending,
    provisioning: counts.provisioning,
    error: counts.error,
    outdatedAgents,
    latestAgentVersion,
  };
}

export function buildServersSubtitle(summary: ServersSummary): string {
  if (summary.total === 0) {
    return "No servers yet — add your first machine to enable remote deploys.";
  }

  const parts: string[] = [];
  parts.push(`${summary.total} ${summary.total === 1 ? "server" : "servers"}`);
  parts.push(`${summary.online}/${summary.total} online`);

  if (summary.latestAgentVersion) {
    parts.push(`agent v${summary.latestAgentVersion}`);
  }

  return parts.join(" · ");
}

export function buildServersNextBestAction(
  summary: ServersSummary,
): ServersNextBestAction | null {
  if (summary.total === 0) return null;

  if (summary.error > 0) {
    return {
      severity: "destructive",
      message:
        summary.error === 1
          ? "1 server is reporting errors. Review the setup guide and re-provision it."
          : `${summary.error} servers are reporting errors. Review the setup guide and re-provision them.`,
      ctaLabel: "Setup guide",
      ctaHref: "/helper/server-setup",
    };
  }

  if (summary.pending > 0) {
    return {
      severity: "warning",
      message:
        summary.pending === 1
          ? "1 server is waiting to be provisioned."
          : `${summary.pending} servers are waiting to be provisioned.`,
      ctaLabel: "Setup guide",
      ctaHref: "/helper/server-setup",
    };
  }

  if (summary.offline > 0) {
    return {
      severity: "warning",
      message:
        summary.offline === 1
          ? "1 server is offline. Check network connectivity."
          : `${summary.offline} servers are offline. Check network connectivity.`,
      ctaLabel: "Setup guide",
      ctaHref: "/helper/server-setup",
    };
  }

  if (summary.outdatedAgents > 0) {
    return {
      severity: "warning",
      message:
        summary.outdatedAgents === 1
          ? "1 server is running an outdated agent."
          : `${summary.outdatedAgents} servers are running outdated agents.`,
      ctaLabel: "Setup guide",
      ctaHref: "/helper/server-setup",
    };
  }

  return null;
}
