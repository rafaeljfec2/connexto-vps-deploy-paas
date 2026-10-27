import { describe, expect, it } from "vitest";
import { parseDeployLogLine } from "../log-utils";

describe("parseDeployLogLine", () => {
  it("parses known stage prefixes from formatLogStage", () => {
    const prefixes = [
      "git",
      "build",
      "deploy",
      "health",
      "rollback",
      "cleanup",
      "complete",
      "agent",
    ] as const;

    for (const prefix of prefixes) {
      const parsed = parseDeployLogLine(`[12:34:56] [${prefix}] hello`, 0);
      expect(parsed.prefix).toBe(prefix);
      expect(parsed.content).toBe("hello");
      expect(parsed.timestamp).not.toBeNull();
    }
  });

  it("strips a Docker timestamp left in the content and keeps deploy time", () => {
    const parsed = parseDeployLogLine(
      "[19:58:01] [health] 2026-04-30T22:58:01.123456789Z Error: Cannot find module 'express'",
      3,
    );

    expect(parsed.prefix).toBe("health");
    expect(parsed.content).toBe("Error: Cannot find module 'express'");
    expect(parsed.timestamp).not.toBeNull();
    expect(parsed.lineNumber).toBe(4);
    expect(parsed.type).toBe("error");
  });

  it("uses Docker timestamp when deploy stamp is missing", () => {
    const parsed = parseDeployLogLine(
      "[health] 2026-04-30T22:58:01.123456789Z container api is restarting",
      0,
    );

    expect(parsed.prefix).toBe("health");
    expect(parsed.content).toBe("container api is restarting");
    expect(parsed.timestamp).not.toBeNull();
  });

  it("strips Docker timestamps without fractional seconds", () => {
    const parsed = parseDeployLogLine(
      "[health] 2026-04-30T22:58:01Z container api is restarting",
      0,
    );

    expect(parsed.prefix).toBe("health");
    expect(parsed.content).toBe("container api is restarting");
    expect(parsed.timestamp).not.toBeNull();
  });
});
