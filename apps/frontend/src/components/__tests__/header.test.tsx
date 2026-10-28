import { ROUTES } from "@/constants/routes";
import type { User } from "@/contexts/auth-context";
import { renderWithProviders, screen, within } from "@/test/test-utils";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Header } from "../header";

interface MockAuthState {
  readonly user: User | null;
  readonly isAuthenticated: boolean;
  readonly logout: () => Promise<void>;
}

const auth = vi.hoisted<{
  initialValue: MockAuthState;
  value: MockAuthState;
}>(() => {
  const initialValue: MockAuthState = {
    user: {
      id: "user-1",
      name: "Ada Lovelace",
      email: "ada@example.com",
      role: "admin",
      authProvider: "email",
      createdAt: "2024-01-01T00:00:00Z",
    },
    isAuthenticated: true,
    logout: vi.fn(async () => undefined),
  };

  return { initialValue, value: initialValue };
});

vi.mock("@/contexts/auth-context", () => ({
  useAuth: () => auth.value,
}));

function getMainNavigation(): HTMLElement {
  return screen.getByRole("navigation", { name: /main navigation/i });
}

beforeEach(() => {
  auth.value = auth.initialValue;
});

describe("Header", () => {
  it("renders the main navigation entries with their routes", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.HOME });

    const nav = within(getMainNavigation());

    expect(nav.getByRole("link", { name: /servers/i })).toHaveAttribute(
      "href",
      ROUTES.SERVERS,
    );
    expect(nav.getByRole("link", { name: /containers/i })).toHaveAttribute(
      "href",
      ROUTES.CONTAINERS,
    );
    expect(nav.getByRole("link", { name: /images/i })).toHaveAttribute(
      "href",
      ROUTES.IMAGES,
    );
    expect(nav.getByRole("link", { name: /audit/i })).toHaveAttribute(
      "href",
      ROUTES.AUDIT,
    );
  });

  it("exposes accessible names for the icon-only navigation state", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.HOME });

    const nav = within(getMainNavigation());

    for (const label of ["Servers", "Containers", "Images", "Audit"]) {
      expect(nav.getByRole("link", { name: label })).toHaveAccessibleName(
        label,
      );
    }
  });

  it("marks the servers entry as the current page", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.SERVERS });

    const nav = within(getMainNavigation());

    expect(nav.getByRole("link", { name: /servers/i })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(nav.getByRole("link", { name: /containers/i })).not.toHaveAttribute(
      "aria-current",
    );
  });

  it("keeps the servers entry active on server detail routes", () => {
    renderWithProviders(<Header />, { initialRoute: "/servers/server-1" });

    expect(
      within(getMainNavigation()).getByRole("link", { name: /servers/i }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("marks containers as current on the containers route", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.CONTAINERS });

    expect(
      within(getMainNavigation()).getByRole("link", { name: /containers/i }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("does not mark containers as current on the templates route", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.TEMPLATES });

    expect(
      within(getMainNavigation()).getByRole("link", { name: /containers/i }),
    ).not.toHaveAttribute("aria-current");
  });

  it("marks images as current on the images route", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.IMAGES });

    expect(
      within(getMainNavigation()).getByRole("link", { name: /images/i }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("marks audit as current on the audit route", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.AUDIT });

    expect(
      within(getMainNavigation()).getByRole("link", { name: /audit/i }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("does not mark any entry as current on the dashboard", () => {
    renderWithProviders(<Header />, { initialRoute: ROUTES.HOME });

    const nav = within(getMainNavigation());

    for (const label of ["Servers", "Containers", "Images", "Audit"]) {
      expect(nav.getByRole("link", { name: label })).not.toHaveAttribute(
        "aria-current",
      );
    }
  });

  it("keeps the management shortcuts in the account menu", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Header />, { initialRoute: ROUTES.HOME });

    await user.click(screen.getByRole("button", { name: /account menu/i }));

    const menu = within(await screen.findByRole("menu"));

    for (const label of [
      /^images$/i,
      /^audit logs$/i,
      /^servers$/i,
      /^helper$/i,
      /^settings$/i,
      /^sign out$/i,
    ]) {
      expect(menu.getByRole("menuitem", { name: label })).toBeInTheDocument();
    }
  });

  it("shows only the sign in action for anonymous visitors", () => {
    auth.value = {
      user: null,
      isAuthenticated: false,
      logout: vi.fn(async () => undefined),
    };

    renderWithProviders(<Header />, { initialRoute: ROUTES.LANDING });

    expect(screen.getByRole("link", { name: /sign in/i })).toBeInTheDocument();
    expect(
      screen.queryByRole("navigation", { name: /main navigation/i }),
    ).toBeNull();
  });
});
