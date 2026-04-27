import { renderWithProviders, screen } from "@/test/test-utils";
import { describe, expect, it, vi } from "vitest";
import type { useAppActions } from "@/features/apps/hooks/use-app-actions";
import type { App, Deployment, HealthStatus } from "@/types";
import { AppHero } from "../app-hero";

type UseAppActions = ReturnType<typeof useAppActions>;

function makeApp(overrides: Partial<App> = {}): App {
  return {
    id: "app-1",
    name: "checkout-api",
    repositoryUrl: "https://github.com/acme/checkout",
    branch: "main",
    workdir: ".",
    runtime: "node",
    config: {},
    status: "active",
    webhookId: null,
    appVersion: "1.2.3",
    serverId: "srv-1",
    lastDeployedAt: "2026-04-26T10:00:00Z",
    createdAt: "2026-04-01T00:00:00Z",
    updatedAt: "2026-04-26T10:00:00Z",
    ...overrides,
  };
}

function makeDeploy(overrides: Partial<Deployment> = {}): Deployment {
  return {
    id: "dep-1",
    appId: "app-1",
    commitSha: "abcdef1234567890",
    commitMessage: "feat: initial deploy",
    status: "success",
    startedAt: "2026-04-26T10:00:00Z",
    finishedAt: "2026-04-26T10:01:00Z",
    errorMessage: null,
    logs: null,
    previousImageTag: null,
    currentImageTag: "checkout-api:1",
    createdAt: "2026-04-26T10:00:00Z",
    ...overrides,
  };
}

function makeActions(overrides: Partial<UseAppActions> = {}): UseAppActions {
  const noopMutation = {
    isPending: false,
    mutate: vi.fn(),
  };
  return {
    redeploy: noopMutation,
    rollback: noopMutation,
    setupWebhook: noopMutation,
    removeWebhook: noopMutation,
    restartContainer: noopMutation,
    stopContainer: noopMutation,
    startContainer: noopMutation,
    handleRedeploy: vi.fn(),
    handleRollback: vi.fn(),
    handleSetupWebhook: vi.fn(),
    handleRemoveWebhook: vi.fn(),
    ...overrides,
  } as unknown as UseAppActions;
}

const healthyHealth: HealthStatus = {
  status: "running",
  health: "healthy",
};

describe("AppHero", () => {
  it("renders app name and version in the heading", () => {
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={[makeDeploy()]}
        containerStats={null}
        openAppUrl={null}
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy
      />,
    );
    expect(
      screen.getByRole("heading", { level: 1, name: /checkout-api/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/v1\.2\.3/)).toBeInTheDocument();
  });

  it("shows the health badge with the correct state", () => {
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={[makeDeploy()]}
        containerStats={null}
        openAppUrl={null}
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy
      />,
    );
    expect(screen.getAllByText(/healthy/i).length).toBeGreaterThan(0);
  });

  it("renders an NBA banner when the latest deploy failed", () => {
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={[makeDeploy({ status: "failed" })]}
        containerStats={null}
        openAppUrl={null}
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy={false}
      />,
    );
    expect(screen.getByText(/action required/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view logs/i })).toHaveAttribute(
      "href",
      "#container-logs-section",
    );
  });

  it("renders Open App CTA when openAppUrl is provided", () => {
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={[makeDeploy()]}
        containerStats={null}
        openAppUrl="https://app.example.com"
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy
      />,
    );
    const cta = screen.getByRole("link", { name: /open app/i });
    expect(cta).toHaveAttribute("href", "https://app.example.com");
    expect(cta).toHaveAttribute("target", "_blank");
  });

  it("disables the Rollback button when there is no successful deploy", () => {
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={[makeDeploy({ status: "failed" })]}
        containerStats={null}
        openAppUrl={null}
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy={false}
      />,
    );
    const rollback = screen.getByRole("button", { name: /rollback/i });
    expect(rollback).toBeDisabled();
  });

  it("renders up to three deploys inside the recent deploys panel", () => {
    const deployments = [
      makeDeploy({ id: "d1", commitMessage: "first change" }),
      makeDeploy({ id: "d2", commitMessage: "second change" }),
      makeDeploy({ id: "d3", commitMessage: "third change" }),
      makeDeploy({ id: "d4", commitMessage: "fourth change" }),
    ];
    renderWithProviders(
      <AppHero
        app={makeApp()}
        health={healthyHealth}
        deployments={deployments}
        containerStats={null}
        openAppUrl={null}
        actions={makeActions()}
        allExpanded={false}
        toggleAllSections={() => {}}
        hasSuccessfulDeploy
      />,
    );
    expect(screen.getByText(/first change/)).toBeInTheDocument();
    expect(screen.getByText(/second change/)).toBeInTheDocument();
    expect(screen.getByText(/third change/)).toBeInTheDocument();
    expect(screen.queryByText(/fourth change/)).not.toBeInTheDocument();
  });
});
