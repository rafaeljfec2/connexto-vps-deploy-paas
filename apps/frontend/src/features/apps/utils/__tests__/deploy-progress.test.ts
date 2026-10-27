import { describe, expect, it } from "vitest";
import { deriveDeployProgressState } from "../deploy-progress";

describe("deriveDeployProgressState", () => {
  it("returns null for terminal statuses", () => {
    expect(deriveDeployProgressState("[health] ok", "success")).toBeNull();
    expect(deriveDeployProgressState("[health] fail", "failed")).toBeNull();
  });

  it("shows queued when pending", () => {
    expect(deriveDeployProgressState("", "pending")).toEqual({
      phase: "Queued",
      progress: 5,
      icon: "loader",
    });
  });

  it("detects health from [health] or Running health check", () => {
    expect(
      deriveDeployProgressState(
        "[12:00:00] [deploy] Container deployed\n[12:00:01] [health] Running health check on port 3000",
        "running",
      ),
    ).toMatchObject({ phase: "Health check", progress: 88, icon: "rocket" });

    expect(
      deriveDeployProgressState(
        "[deploy] done\nRunning health check on port 8080 path /health ...",
        "running",
      ),
    ).toMatchObject({ phase: "Health check", progress: 88 });
  });

  it("prefers rollback over health when both appear", () => {
    const state = deriveDeployProgressState(
      "[health] Health check failed\n[rollback] Rolling back to: app:prev",
      "running",
    );
    expect(state).toEqual({
      phase: "Rolling back",
      progress: 95,
      icon: "rocket",
    });
  });

  it("does not match the removed Waiting for health check string alone", () => {
    const state = deriveDeployProgressState(
      "[deploy] Deploying container...\nWaiting for health check",
      "running",
    );
    expect(state?.phase).toBe("Deploying");
    expect(state?.progress).toBe(75);
  });
});
