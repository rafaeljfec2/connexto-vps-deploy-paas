import { ROUTES } from "@/constants/routes";
import { describe, expect, it } from "vitest";
import type { DeployStatus } from "@/types";
import {
  buildHeroSubtitle,
  buildNextBestAction,
  pickNorthStar,
} from "../hero-insights";
import type { KpiTones } from "../kpi-tone";

function tones(partial: Partial<KpiTones> = {}): KpiTones {
  return {
    apps: "default",
    servers: "success",
    containers: "success",
    deploys: "success",
    ...partial,
  };
}

describe("pickNorthStar", () => {
  it("promotes the worst-tone KPI to North Star", () => {
    expect(pickNorthStar(tones({ containers: "destructive" }))).toBe(
      "containers",
    );
  });

  it("prefers servers over containers when both are degraded", () => {
    expect(
      pickNorthStar(tones({ servers: "warning", containers: "warning" })),
    ).toBe("servers");
  });

  it("returns servers as a calm-state default when nothing is degraded", () => {
    expect(pickNorthStar(tones())).toBe("servers");
  });
});

describe("buildHeroSubtitle", () => {
  it("builds a last-deploy subtitle when there is at least one deploy", () => {
    const now = new Date().toISOString();
    const subtitle = buildHeroSubtitle(
      [
        {
          appId: "a1",
          appName: "billing-api",
          deployment: {
            id: "d1",
            status: "success" satisfies DeployStatus,
            commitSha: "abcdef1",
            startedAt: now,
            finishedAt: now,
          },
        },
      ],
      5,
      2,
    );

    expect(subtitle).toContain("billing-api");
    expect(subtitle.toLowerCase()).toContain("last deploy");
  });

  it("falls back to a counts subtitle when there are no recent deploys", () => {
    expect(buildHeroSubtitle([], 3, 2)).toBe(
      "3 apps · 2 servers · no deploys yet",
    );
  });

  it("falls back to an onboarding subtitle when there are no apps", () => {
    expect(buildHeroSubtitle([], 0, 0)).toBe(
      "Let's get your first app deployed.",
    );
  });
});

describe("buildNextBestAction", () => {
  const healthyInput = {
    tones: tones(),
    onlineServers: 2,
    totalServers: 2,
    runningContainers: 10,
    totalContainers: 10,
    successfulDeploys: 5,
    failedDeploys: 0,
  } as const;

  it("returns null when the whole platform is healthy", () => {
    expect(buildNextBestAction(healthyInput)).toBeNull();
  });

  it("surfaces a destructive alert and server link when all servers are offline", () => {
    const action = buildNextBestAction({
      ...healthyInput,
      tones: tones({ servers: "destructive" }),
      onlineServers: 0,
    });

    expect(action?.severity).toBe("destructive");
    expect(action?.ctaHref).toBe(ROUTES.SERVERS);
  });

  it("ranks server-offline warning above failed-deploys warning", () => {
    const action = buildNextBestAction({
      ...healthyInput,
      tones: tones({ servers: "warning", deploys: "warning" }),
      onlineServers: 1,
      totalServers: 3,
      failedDeploys: 2,
    });

    expect(action?.severity).toBe("warning");
    expect(action?.ctaHref).toBe(ROUTES.SERVERS);
  });

  it("points to the audit log when deploys have failed and servers are healthy", () => {
    const action = buildNextBestAction({
      ...healthyInput,
      tones: tones({ deploys: "warning" }),
      successfulDeploys: 3,
      failedDeploys: 1,
    });

    expect(action?.ctaHref).toBe(ROUTES.AUDIT);
  });
});
