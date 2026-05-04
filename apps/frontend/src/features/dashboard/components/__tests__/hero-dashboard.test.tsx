import { renderWithProviders } from "@/test/test-utils";
import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CommandPaletteProvider } from "@/hooks/use-command-palette";
import type { DeployStatus } from "@/types";
import { HeroDashboard } from "../hero-dashboard";

vi.mock("@/contexts/auth-context", () => ({
  useAuth: () => ({
    user: {
      id: "u1",
      name: "Ada Lovelace",
      email: "ada@example.com",
      githubLogin: "ada",
    },
  }),
}));

vi.mock("@/hooks/use-sse", () => ({
  useSSEConnectionStatus: () => true,
}));

const statsMock = vi.hoisted(() => ({
  value: {
    totalApps: 0,
    totalServers: 0,
    onlineServers: 0,
    totalContainers: 0,
    runningContainers: 0,
    recentDeploys: [] as readonly {
      appId: string;
      appName: string;
      deployment: {
        id: string;
        status: DeployStatus;
        commitSha: string;
        startedAt: string | null;
        finishedAt: string | null;
      };
    }[],
    successfulDeploys: 0,
    failedDeploys: 0,
    isLoading: false,
  },
}));

vi.mock("../../hooks/use-dashboard-stats", () => ({
  useDashboardStats: () => statsMock.value,
}));

type Stats = typeof statsMock.value;

function setStats(partial: Partial<Stats>) {
  statsMock.value = { ...statsMock.value, ...partial };
}

function renderHero() {
  return renderWithProviders(
    <CommandPaletteProvider>
      <HeroDashboard />
    </CommandPaletteProvider>,
  );
}

describe("HeroDashboard", () => {
  it("renders greeting with the user's first name", () => {
    setStats({
      totalApps: 3,
      totalServers: 2,
      onlineServers: 2,
      totalContainers: 5,
      runningContainers: 5,
      isLoading: false,
    });

    renderHero();
    expect(
      screen.getByRole("heading", { level: 1, name: /Ada/ }),
    ).toBeInTheDocument();
  });

  it("does not render an NBA banner when everything is healthy", () => {
    setStats({
      totalApps: 3,
      totalServers: 2,
      onlineServers: 2,
      totalContainers: 5,
      runningContainers: 5,
      successfulDeploys: 2,
      failedDeploys: 0,
      isLoading: false,
    });

    renderHero();
    expect(screen.queryByText(/Action required/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Needs your attention/i)).not.toBeInTheDocument();
  });

  it("surfaces the NBA banner with a destructive tone when all servers are offline", () => {
    setStats({
      totalApps: 3,
      totalServers: 2,
      onlineServers: 0,
      totalContainers: 5,
      runningContainers: 0,
      successfulDeploys: 0,
      failedDeploys: 0,
      isLoading: false,
    });

    renderHero();
    expect(screen.getByText(/Action required/i)).toBeInTheDocument();
    const viewServers = screen.getAllByRole("link", { name: /View servers/i });
    expect(viewServers.length).toBeGreaterThan(0);
  });

  it("renders up to three recent deploys inside the live activity panel", () => {
    const now = new Date().toISOString();
    setStats({
      totalApps: 4,
      totalServers: 1,
      onlineServers: 1,
      totalContainers: 4,
      runningContainers: 4,
      successfulDeploys: 4,
      failedDeploys: 0,
      isLoading: false,
      recentDeploys: [
        {
          appId: "a1",
          appName: "billing-api",
          deployment: {
            id: "d1",
            status: "success",
            commitSha: "aaaaaaa1",
            startedAt: now,
            finishedAt: now,
          },
        },
        {
          appId: "a2",
          appName: "frontend",
          deployment: {
            id: "d2",
            status: "success",
            commitSha: "bbbbbbb2",
            startedAt: now,
            finishedAt: now,
          },
        },
        {
          appId: "a3",
          appName: "worker",
          deployment: {
            id: "d3",
            status: "success",
            commitSha: "ccccccc3",
            startedAt: now,
            finishedAt: now,
          },
        },
        {
          appId: "a4",
          appName: "cron",
          deployment: {
            id: "d4",
            status: "success",
            commitSha: "ddddddd4",
            startedAt: now,
            finishedAt: now,
          },
        },
      ],
    });

    renderHero();
    const liveSection = screen.getByText(/Live activity/i).closest("div");
    expect(liveSection).not.toBeNull();
    const parent = liveSection?.parentElement as HTMLElement;
    expect(within(parent).getByText("billing-api")).toBeInTheDocument();
    expect(within(parent).getByText("frontend")).toBeInTheDocument();
    expect(within(parent).getByText("worker")).toBeInTheDocument();
    expect(within(parent).queryByText("cron")).not.toBeInTheDocument();
  });

  it("renders four KPI skeletons while stats are loading", () => {
    setStats({ isLoading: true });
    const { container } = renderHero();
    expect(
      container.querySelectorAll('[aria-busy="true"]').length,
    ).toBeGreaterThanOrEqual(4);
    expect(
      screen.getByText(/Loading infrastructure pulse/),
    ).toBeInTheDocument();
  });

  it("renders CTA links pointing to /apps/new and /servers", () => {
    setStats({
      totalApps: 1,
      totalServers: 1,
      onlineServers: 1,
      totalContainers: 1,
      runningContainers: 1,
      isLoading: false,
    });

    renderHero();
    expect(screen.getByRole("link", { name: /^New App$/i })).toHaveAttribute(
      "href",
      "/apps/new",
    );
    expect(
      screen.getAllByRole("link", { name: /^Servers$/i })[0],
    ).toHaveAttribute("href", "/servers");
  });

  it("exposes the command palette trigger button", () => {
    setStats({ isLoading: false });
    renderHero();
    expect(
      screen.getByRole("button", { name: /Open command palette/i }),
    ).toBeInTheDocument();
  });
});
