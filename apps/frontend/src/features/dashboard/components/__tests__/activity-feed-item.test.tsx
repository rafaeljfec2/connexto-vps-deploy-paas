import { MemoryRouter } from "react-router-dom";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ActivityFeedItem } from "../activity-feed-item";

describe("ActivityFeedItem", () => {
  it("renders app name, commit message, sha and relative time", () => {
    render(
      <MemoryRouter>
        <ActivityFeedItem
          appId="app-1"
          appName="connexto-digital-signer-api"
          status="success"
          commitSha="4f3e2b10987654321"
          commitMessage="chore: bump api version"
          timestamp="2026-10-28T12:00:00Z"
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("connexto-digital-signer-api")).toBeInTheDocument();
    expect(screen.getByText("chore: bump api version")).toBeInTheDocument();
    expect(screen.getByText(/4f3e2b1/)).toBeInTheDocument();
  });

  it("renders fallback message when commit message is missing", () => {
    render(
      <MemoryRouter>
        <ActivityFeedItem
          appId="app-2"
          appName="oabprev-integration-suite"
          status="success"
          commitSha="2f1a9b0123"
          timestamp="2026-10-28T12:00:00Z"
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("Manual deploy triggered")).toBeInTheDocument();
  });
});
