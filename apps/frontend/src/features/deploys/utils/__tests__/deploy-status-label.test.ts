import { describe, expect, it } from "vitest";
import {
  getDeployDurationMs,
  getDeployStatusPresentation,
} from "../deploy-status-label";

describe("getDeployStatusPresentation", () => {
  it("maps deploy statuses to Vercel-style labels", () => {
    expect(getDeployStatusPresentation("success")).toEqual({
      label: "Ready",
      tone: "success",
    });
    expect(getDeployStatusPresentation("failed")).toEqual({
      label: "Error",
      tone: "error",
    });
    expect(getDeployStatusPresentation("running")).toEqual({
      label: "Building",
      tone: "running",
    });
    expect(getDeployStatusPresentation("pending")).toEqual({
      label: "Building",
      tone: "running",
    });
    expect(getDeployStatusPresentation("cancelled")).toEqual({
      label: "Cancelled",
      tone: "muted",
    });
  });
});

describe("getDeployDurationMs", () => {
  it("returns finished-start duration when both timestamps exist", () => {
    expect(
      getDeployDurationMs(
        "2026-10-28T00:00:00.000Z",
        "2026-10-28T00:01:24.000Z",
      ),
    ).toBe(84_000);
  });

  it("returns elapsed time against now when still running", () => {
    expect(
      getDeployDurationMs(
        "2026-10-28T00:00:00.000Z",
        null,
        Date.parse("2026-10-28T00:00:32.000Z"),
      ),
    ).toBe(32_000);
  });

  it("returns null without a start timestamp", () => {
    expect(getDeployDurationMs(null, "2026-10-28T00:00:10.000Z")).toBeNull();
  });
});
