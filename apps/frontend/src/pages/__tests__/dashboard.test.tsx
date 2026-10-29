import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DashboardPage } from "../dashboard";

vi.mock("@/features/dashboard/components/hero-dashboard", () => ({
  HeroDashboard: () => (
    <div data-testid="hero-dashboard">Hero Dashboard Component</div>
  ),
}));

vi.mock("@/features/dashboard/components/server-health-overview", () => ({
  ServerHealthOverview: () => (
    <div data-testid="server-health-overview">Server Health Overview</div>
  ),
}));

vi.mock("@/features/apps/components/app-list", () => ({
  AppList: () => <div data-testid="app-list">App List Component</div>,
}));

vi.mock("@/features/dashboard/components/activity-feed", () => ({
  ActivityFeed: () => (
    <div data-testid="activity-feed">Activity Feed Component</div>
  ),
}));

describe("DashboardPage", () => {
  it("renders the hero dashboard, server health overview, and applications section", () => {
    const queryClient = new QueryClient();

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByTestId("hero-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("server-health-overview")).toBeInTheDocument();

    expect(
      screen.getByRole("heading", { name: "Applications" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Manage and monitor your deployments"),
    ).toBeInTheDocument();

    const newAppButton = screen.getByRole("link", { name: /new app/i });
    expect(newAppButton).toBeInTheDocument();
    expect(newAppButton).toHaveAttribute("href", "/apps/new");

    expect(screen.getByTestId("app-list")).toBeInTheDocument();
    expect(screen.getByTestId("activity-feed")).toBeInTheDocument();
  });
});
