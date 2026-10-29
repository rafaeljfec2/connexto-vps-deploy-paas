import { describe, expect, it } from "vitest";
import { extractRepoOwner } from "../org-helpers";

describe("extractRepoOwner", () => {
  it("extracts owner from standard HTTPS GitHub URL", () => {
    expect(
      extractRepoOwner("https://github.com/Outcoders/connexto-nexo-assist-api"),
    ).toBe("Outcoders");
  });

  it("extracts owner from SSH GitHub URL", () => {
    expect(
      extractRepoOwner("git@github.com:OABPrev/oabprev-integration-suite.git"),
    ).toBe("OABPrev");
  });

  it("extracts owner from GitHub URL with trailing slash or .git", () => {
    expect(extractRepoOwner("https://github.com/facebook/react.git/")).toBe(
      "facebook",
    );
  });

  it("returns null for empty or invalid repository URLs", () => {
    expect(extractRepoOwner("")).toBeNull();
    expect(extractRepoOwner("https://github.com/")).toBeNull();
    expect(extractRepoOwner("just-a-name")).toBeNull();
  });
});
