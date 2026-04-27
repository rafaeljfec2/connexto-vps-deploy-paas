import { describe, expect, it } from "vitest";
import type { ContainerStats, Deployment, HealthStatus } from "@/types";
import {
  buildAppNextBestAction,
  buildAppSubtitle,
  deriveAppKpiTones,
  parseHealthState,
  pickAppNorthStar,
} from "../app-hero-insights";

function makeDeploy(overrides: Partial<Deployment> = {}): Deployment {
  return {
    id: "dep-1",
    appId: "app-1",
    commitSha: "abcdef1234567890",
    commitMessage: "chore: test",
    status: "success",
    startedAt: "2026-04-26T10:00:00Z",
    finishedAt: "2026-04-26T10:01:00Z",
    errorMessage: null,
    logs: null,
    previousImageTag: null,
    currentImageTag: "app:1",
    createdAt: "2026-04-26T10:00:00Z",
    ...overrides,
  };
}

function makeHealth(overrides: Partial<HealthStatus> = {}): HealthStatus {
  return {
    status: "running",
    health: "healthy",
    ...overrides,
  };
}

function makeStats(overrides: Partial<ContainerStats> = {}): ContainerStats {
  return {
    cpuPercent: 10,
    memoryUsage: 100 * 1024 * 1024,
    memoryLimit: 1024 * 1024 * 1024,
    memoryPercent: 10,
    networkRx: 0,
    networkTx: 0,
    pids: 1,
    ...overrides,
  };
}

describe("parseHealthState", () => {
  it("returns 'unknown' when health is missing", () => {
    expect(parseHealthState(undefined)).toBe("unknown");
    expect(parseHealthState(null)).toBe("unknown");
  });

  it("returns 'healthy' when running and reported healthy", () => {
    expect(parseHealthState(makeHealth())).toBe("healthy");
  });

  it("returns 'unhealthy' when running but unhealthy", () => {
    expect(parseHealthState(makeHealth({ health: "unhealthy" }))).toBe(
      "unhealthy",
    );
  });

  it("returns 'offline' when container is exited or paused", () => {
    expect(
      parseHealthState(makeHealth({ status: "exited", health: "none" })),
    ).toBe("offline");
    expect(
      parseHealthState(makeHealth({ status: "paused", health: "none" })),
    ).toBe("offline");
  });

  it("returns 'starting' when restarting", () => {
    expect(
      parseHealthState(makeHealth({ status: "restarting", health: "none" })),
    ).toBe("starting");
  });
});

describe("deriveAppKpiTones", () => {
  it("marks lastDeploy destructive when the last deploy failed", () => {
    const tones = deriveAppKpiTones({
      latestDeploy: makeDeploy({ status: "failed" }),
      healthState: "healthy",
      containerStats: makeStats(),
    });
    expect(tones.lastDeploy).toBe("destructive");
  });

  it("marks health destructive when container is unhealthy", () => {
    const tones = deriveAppKpiTones({
      latestDeploy: makeDeploy(),
      healthState: "unhealthy",
      containerStats: makeStats(),
    });
    expect(tones.health).toBe("destructive");
  });

  it("marks cpu destructive when usage is at or above 85 percent", () => {
    const tones = deriveAppKpiTones({
      latestDeploy: makeDeploy(),
      healthState: "healthy",
      containerStats: makeStats({ cpuPercent: 90, memoryPercent: 30 }),
    });
    expect(tones.cpu).toBe("destructive");
    expect(tones.memory).toBe("success");
  });

  it("returns default tones when stats are missing", () => {
    const tones = deriveAppKpiTones({
      latestDeploy: null,
      healthState: "unknown",
      containerStats: null,
    });
    expect(tones).toEqual({
      lastDeploy: "default",
      health: "default",
      cpu: "default",
      memory: "default",
    });
  });
});

describe("pickAppNorthStar", () => {
  it("picks health as north star when unhealthy", () => {
    const key = pickAppNorthStar({
      lastDeploy: "success",
      health: "destructive",
      cpu: "success",
      memory: "success",
    });
    expect(key).toBe("health");
  });

  it("prefers lastDeploy failed when health is success", () => {
    const key = pickAppNorthStar({
      lastDeploy: "destructive",
      health: "success",
      cpu: "warning",
      memory: "success",
    });
    expect(key).toBe("lastDeploy");
  });

  it("falls back to lastDeploy when everything is success", () => {
    const key = pickAppNorthStar({
      lastDeploy: "success",
      health: "success",
      cpu: "success",
      memory: "success",
    });
    expect(key).toBe("lastDeploy");
  });
});

describe("buildAppSubtitle", () => {
  it("builds 'deployed to URL' subtitle when openAppUrl is present", () => {
    const subtitle = buildAppSubtitle({
      latestDeploy: makeDeploy(),
      openAppUrl: "https://app.example.com",
      branch: "main",
    });
    expect(subtitle).toContain("https://app.example.com");
    expect(subtitle.toLowerCase()).toContain("deployed");
  });

  it("falls back to branch-only subtitle when no URL", () => {
    const subtitle = buildAppSubtitle({
      latestDeploy: makeDeploy(),
      openAppUrl: null,
      branch: "develop",
    });
    expect(subtitle).toContain("develop");
    expect(subtitle).not.toContain("serving at");
  });

  it("returns 'never deployed yet' when there is no latest deploy", () => {
    const subtitle = buildAppSubtitle({
      latestDeploy: null,
      openAppUrl: null,
      branch: "main",
    });
    expect(subtitle.toLowerCase()).toContain("never deployed");
  });
});

describe("buildAppNextBestAction", () => {
  const healthyTones = {
    lastDeploy: "success",
    health: "success",
    cpu: "success",
    memory: "success",
  } as const;

  it("returns null NBA when everything is healthy", () => {
    const nba = buildAppNextBestAction({
      latestDeploy: makeDeploy(),
      healthState: "healthy",
      tones: healthyTones,
    });
    expect(nba).toBeNull();
  });

  it("surfaces an NBA linking to logs when latest deploy failed", () => {
    const nba = buildAppNextBestAction({
      latestDeploy: makeDeploy({ status: "failed" }),
      healthState: "healthy",
      tones: { ...healthyTones, lastDeploy: "destructive" },
    });
    expect(nba).not.toBeNull();
    expect(nba?.action).toBe("view-logs");
    expect(nba?.severity).toBe("destructive");
  });

  it("surfaces a redeploy NBA when container is offline", () => {
    const nba = buildAppNextBestAction({
      latestDeploy: makeDeploy(),
      healthState: "offline",
      tones: { ...healthyTones, health: "destructive" },
    });
    expect(nba?.action).toBe("redeploy");
    expect(nba?.severity).toBe("warning");
  });
});
