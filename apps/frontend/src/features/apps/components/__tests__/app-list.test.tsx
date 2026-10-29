import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { App } from "@/types";
import { AppList } from "../app-list";

const mockApps: App[] = [
  {
    id: "app-1",
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
      durationMs: 65000,
    },
  },
  {
    id: "app-2",
    name: "oabprev-integration-suite",
    repositoryUrl: "https://github.com/OABPrev/oabprev-integration-suite",
    branch: "main",
    workdir: ".",
    runtime: "go",
    config: {},
    status: "active",
    webhookId: null,
    appVersion: "0.6.0",
    serverId: "server-2",
    lastDeployedAt: "2026-10-28T08:00:00Z",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-10-28T08:00:00Z",
    lastDeployment: {
      id: "dep-2",
      status: "success",
      commitSha: "2f1a9b0",
      durationMs: 51000,
    },
  },
];

vi.mock("../../hooks/use-apps", () => ({
  useApps: () => ({
    data: mockApps,
    isLoading: false,
    error: null,
  }),
  usePurgeApp: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("@/features/servers/hooks/use-servers", () => ({
  useServers: () => ({
    data: [
      { id: "server-1", name: "Outcoders VPS" },
      { id: "server-2", name: "OABPrev Prod" },
    ],
  }),
}));

vi.mock("@/contexts/auth-context", () => ({
  useAuth: () => ({
    user: { id: "user-1", email: "test@example.com", name: "Rafael" },
    isAdmin: true,
  }),
}));

function renderAppList() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AppList />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AppList", () => {
  it("renders search input, server filter, and view mode controls", () => {
    renderAppList();

    expect(
      screen.getByPlaceholderText("Search applications..."),
    ).toBeInTheDocument();
    expect(screen.getByRole("combobox")).toBeInTheDocument();
    expect(screen.getByTitle("List view")).toBeInTheDocument();
    expect(screen.getByTitle("Grid view")).toBeInTheDocument();
  });

  it("filters applications when search query is typed", () => {
    renderAppList();

    expect(screen.getByText("connexto-nexo-assist-api")).toBeInTheDocument();
    expect(screen.getByText("oabprev-integration-suite")).toBeInTheDocument();

    const searchInput = screen.getByPlaceholderText("Search applications...");
    fireEvent.change(searchInput, { target: { value: "oabprev" } });

    expect(
      screen.queryByText("connexto-nexo-assist-api"),
    ).not.toBeInTheDocument();
    expect(screen.getByText("oabprev-integration-suite")).toBeInTheDocument();
  });

  it("renders empty state when search returns no results", () => {
    renderAppList();

    const searchInput = screen.getByPlaceholderText("Search applications...");
    fireEvent.change(searchInput, { target: { value: "non-existent-app" } });

    expect(screen.getByText("No applications found")).toBeInTheDocument();
  });
});
