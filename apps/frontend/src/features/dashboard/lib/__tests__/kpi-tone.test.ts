import { describe, expect, it } from "vitest";
import { deriveKpiTones, toneSeverity } from "../kpi-tone";

describe("deriveKpiTones", () => {
  const baseStats = {
    totalApps: 0,
    totalServers: 0,
    onlineServers: 0,
    totalContainers: 0,
    runningContainers: 0,
    successfulDeploys: 0,
    failedDeploys: 0,
  } as const;

  it("returns default tones when there is no infrastructure yet", () => {
    const tones = deriveKpiTones(baseStats);
    expect(tones).toEqual({
      apps: "default",
      servers: "default",
      containers: "default",
      deploys: "default",
    });
  });

  it("marks servers success when all are online", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalServers: 3,
      onlineServers: 3,
    });
    expect(tones.servers).toBe("success");
  });

  it("marks servers warning when some are offline", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalServers: 3,
      onlineServers: 2,
    });
    expect(tones.servers).toBe("warning");
  });

  it("marks servers destructive when all are offline", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalServers: 3,
      onlineServers: 0,
    });
    expect(tones.servers).toBe("destructive");
  });

  it("marks containers success when running >= 80% of total", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalContainers: 10,
      runningContainers: 8,
    });
    expect(tones.containers).toBe("success");
  });

  it("marks containers warning when running is between 50% and 80%", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalContainers: 10,
      runningContainers: 6,
    });
    expect(tones.containers).toBe("warning");
  });

  it("marks containers destructive when running is below 50%", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      totalContainers: 11,
      runningContainers: 3,
    });
    expect(tones.containers).toBe("destructive");
  });

  it("marks deploys destructive when every deploy failed", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      successfulDeploys: 0,
      failedDeploys: 4,
    });
    expect(tones.deploys).toBe("destructive");
  });

  it("marks deploys warning when some deploys failed but some succeeded", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      successfulDeploys: 3,
      failedDeploys: 1,
    });
    expect(tones.deploys).toBe("warning");
  });

  it("marks deploys success when there are only successful deploys", () => {
    const tones = deriveKpiTones({
      ...baseStats,
      successfulDeploys: 8,
      failedDeploys: 0,
    });
    expect(tones.deploys).toBe("success");
  });

  it("keeps apps always on the default tone", () => {
    const tones = deriveKpiTones({ ...baseStats, totalApps: 42 });
    expect(tones.apps).toBe("default");
  });
});

describe("toneSeverity", () => {
  it("orders tones from default up to destructive", () => {
    expect(toneSeverity("default")).toBeLessThan(toneSeverity("success"));
    expect(toneSeverity("success")).toBeLessThan(toneSeverity("warning"));
    expect(toneSeverity("warning")).toBeLessThan(toneSeverity("destructive"));
  });
});
