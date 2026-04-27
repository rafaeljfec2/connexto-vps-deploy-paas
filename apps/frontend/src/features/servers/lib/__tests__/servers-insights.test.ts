import { describe, expect, it } from "vitest";
import type { Server } from "@/types";
import {
  buildServersNextBestAction,
  buildServersSubtitle,
  summarizeServers,
} from "../servers-insights";

function makeServer(overrides: Partial<Server> = {}): Server {
  return {
    id: "srv-1",
    name: "outcoders",
    host: "147.79.81.2",
    sshPort: 22,
    sshUser: "paasdeploy",
    status: "online",
    agentVersion: "0.22.2",
    agentUpdateMode: "grpc",
    latestAgentVersion: "0.22.2",
    lastHeartbeatAt: "2026-04-26T21:04:00Z",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-04-26T21:04:00Z",
    ...overrides,
  };
}

describe("summarizeServers", () => {
  it("counts servers by status", () => {
    const summary = summarizeServers([
      makeServer({ id: "a", status: "online" }),
      makeServer({ id: "b", status: "online" }),
      makeServer({ id: "c", status: "offline" }),
      makeServer({ id: "d", status: "error" }),
      makeServer({ id: "e", status: "pending" }),
      makeServer({ id: "f", status: "provisioning" }),
    ]);

    expect(summary).toMatchObject({
      total: 6,
      online: 2,
      offline: 1,
      error: 1,
      pending: 1,
      provisioning: 1,
    });
  });

  it("counts outdated agents", () => {
    const summary = summarizeServers([
      makeServer({
        id: "a",
        agentVersion: "0.22.2",
        latestAgentVersion: "0.22.2",
      }),
      makeServer({
        id: "b",
        agentVersion: "0.21.0",
        latestAgentVersion: "0.22.2",
      }),
      makeServer({
        id: "c",
        agentVersion: undefined,
        latestAgentVersion: "0.22.2",
      }),
    ]);

    expect(summary.outdatedAgents).toBe(1);
    expect(summary.latestAgentVersion).toBe("0.22.2");
  });

  it("returns zeroed summary for empty fleet", () => {
    expect(summarizeServers([])).toMatchObject({
      total: 0,
      online: 0,
      offline: 0,
      outdatedAgents: 0,
      latestAgentVersion: null,
    });
  });
});

describe("buildServersSubtitle", () => {
  it("builds empty-state subtitle when no servers", () => {
    const subtitle = buildServersSubtitle(summarizeServers([]));
    expect(subtitle).toMatch(/no servers yet/i);
  });

  it("builds fleet subtitle with online ratio and agent version", () => {
    const subtitle = buildServersSubtitle(
      summarizeServers([
        makeServer({ id: "a", status: "online" }),
        makeServer({ id: "b", status: "offline" }),
      ]),
    );
    expect(subtitle).toBe("2 servers · 1/2 online · agent v0.22.2");
  });

  it("uses singular form for a single server", () => {
    const subtitle = buildServersSubtitle(
      summarizeServers([makeServer({ id: "a", status: "online" })]),
    );
    expect(subtitle).toBe("1 server · 1/1 online · agent v0.22.2");
  });
});

describe("buildServersNextBestAction", () => {
  it("returns null when fleet is empty", () => {
    expect(buildServersNextBestAction(summarizeServers([]))).toBeNull();
  });

  it("returns destructive NBA when any server is in error", () => {
    const action = buildServersNextBestAction(
      summarizeServers([makeServer({ status: "error" })]),
    );
    expect(action?.severity).toBe("destructive");
    expect(action?.message).toMatch(/error/i);
  });

  it("returns warning NBA when servers are pending", () => {
    const action = buildServersNextBestAction(
      summarizeServers([
        makeServer({ id: "a", status: "online" }),
        makeServer({ id: "b", status: "pending" }),
      ]),
    );
    expect(action?.severity).toBe("warning");
    expect(action?.message).toMatch(/waiting to be provisioned/i);
  });

  it("returns warning NBA when servers are offline", () => {
    const action = buildServersNextBestAction(
      summarizeServers([
        makeServer({ id: "a", status: "online" }),
        makeServer({ id: "b", status: "offline" }),
      ]),
    );
    expect(action?.severity).toBe("warning");
    expect(action?.message).toMatch(/offline/i);
  });

  it("returns warning NBA when there are outdated agents", () => {
    const action = buildServersNextBestAction(
      summarizeServers([
        makeServer({
          agentVersion: "0.20.0",
          latestAgentVersion: "0.22.2",
        }),
      ]),
    );
    expect(action?.severity).toBe("warning");
    expect(action?.message).toMatch(/outdated/i);
  });

  it("returns null when fleet is healthy and up to date", () => {
    const action = buildServersNextBestAction(
      summarizeServers([
        makeServer({ id: "a", status: "online" }),
        makeServer({ id: "b", status: "online" }),
      ]),
    );
    expect(action).toBeNull();
  });

  it("prioritizes destructive NBA over other warnings", () => {
    const action = buildServersNextBestAction(
      summarizeServers([
        makeServer({ id: "a", status: "error" }),
        makeServer({ id: "b", status: "offline" }),
      ]),
    );
    expect(action?.severity).toBe("destructive");
  });
});
