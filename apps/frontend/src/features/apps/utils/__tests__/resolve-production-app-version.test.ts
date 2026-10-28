import { describe, expect, it } from "vitest";
import type { App, Deployment } from "@/types";
import { resolveProductionAppVersion } from "../resolve-production-app-version";

function makeApp(overrides: Partial<App> = {}): Pick<App, "appVersion"> {
  return { appVersion: "1.0.0", ...overrides };
}

function makeDeploy(overrides: Partial<Deployment> = {}): Deployment {
  return {
    id: "dep-1",
    appId: "app-1",
    commitSha: "abcdef1234567890",
    commitMessage: "feat: deploy",
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

describe("resolveProductionAppVersion", () => {
  it("prefers the first successful deployment appVersion", () => {
    const version = resolveProductionAppVersion(
      makeApp({ appVersion: "9.9.9" }),
      [
        makeDeploy({ id: "running", status: "running", appVersion: "2.0.0" }),
        makeDeploy({ id: "current", status: "success", appVersion: "1.4.2" }),
        makeDeploy({ id: "older", status: "success", appVersion: "1.4.1" }),
      ],
    );
    expect(version).toBe("1.4.2");
  });

  it("falls back to app.appVersion when no success deploy has a version", () => {
    const version = resolveProductionAppVersion(
      makeApp({ appVersion: "0.3.1" }),
      [
        makeDeploy({ status: "failed" }),
        makeDeploy({ status: "success", appVersion: "" }),
        makeDeploy({ status: "success", appVersion: "   " }),
      ],
    );
    expect(version).toBe("0.3.1");
  });

  it("falls back to app.appVersion when deployments are empty or undefined", () => {
    expect(resolveProductionAppVersion(makeApp(), [])).toBe("1.0.0");
    expect(resolveProductionAppVersion(makeApp(), undefined)).toBe("1.0.0");
    expect(resolveProductionAppVersion(makeApp(), null)).toBe("1.0.0");
  });

  it("returns null when neither deploy nor app has a non-empty version", () => {
    expect(
      resolveProductionAppVersion(makeApp({ appVersion: undefined }), [
        makeDeploy({ status: "failed" }),
      ]),
    ).toBeNull();
    expect(
      resolveProductionAppVersion(makeApp({ appVersion: "  " }), []),
    ).toBeNull();
  });
});
