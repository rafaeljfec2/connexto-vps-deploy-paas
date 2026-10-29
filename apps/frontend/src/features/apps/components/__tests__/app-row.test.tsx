import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { App, Deployment } from "@/types";
import { AppRow } from "../app-row";

const mockApp: App = {
  id: "app-123",
  name: "connexto-nexo-assist-api",
  repositoryUrl: "https://github.com/Outcoders/connexto-nexo-assist-api",
  branch: "main",
  workdir: "apps/api",
  runtime: "node",
  config: {},
  status: "active",
  webhookId: null,
  appVersion: "0.8.1",
  serverId: "server-1",
  lastDeployedAt: "2026-10-28T10:00:00Z",
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-10-28T10:00:00Z",
  lastDeployment: {
    id: "dep-1",
    status: "success",
    commitSha: "a9b2c3d4",
    commitMessage: "feat: new assist features",
    durationMs: 65000,
    startedAt: "2026-10-28T09:58:55Z",
    finishedAt: "2026-10-28T10:00:00Z",
  },
};

function renderAppRow(app: App = mockApp, latestDeploy?: Deployment) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AppRow app={app} latestDeploy={latestDeploy} serverName="Outcoders" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AppRow", () => {
  it("renders app name, tech badge, version tag, and git branch", () => {
    renderAppRow();

    expect(screen.getByText("connexto-nexo-assist-api")).toBeInTheDocument();
    expect(screen.getByText("Node.js")).toBeInTheDocument();
    expect(screen.getByText("v0.8.1")).toBeInTheDocument();
    expect(screen.getByText("main")).toBeInTheDocument();
    expect(screen.getByText("Deployed")).toBeInTheDocument();
  });

  it("renders failed indicator and badge when deployment has failed", () => {
    const failedApp: App = {
      ...mockApp,
      name: "connexto-minc-hub",
      lastDeployment: {
        id: "dep-failed",
        status: "failed",
        commitSha: "12345678",
        commitMessage: "error build",
        durationMs: 30000,
        startedAt: "2026-10-28T09:58:55Z",
        finishedAt: "2026-10-28T09:59:25Z",
      },
    };

    renderAppRow(failedApp);

    expect(screen.getByText("connexto-minc-hub")).toBeInTheDocument();
    expect(screen.getByText("Failed")).toBeInTheDocument();
    expect(screen.getByLabelText(/failed|alert/i)).toBeInTheDocument();
  });

  it("renders repository external link", () => {
    renderAppRow();

    const link = screen.getByTitle("Open repository");
    expect(link).toHaveAttribute(
      "href",
      "https://github.com/Outcoders/connexto-nexo-assist-api",
    );
  });
});
