import { renderWithProviders } from "@/test/test-utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import {
  CommandPaletteProvider,
  useCommandPalette,
} from "@/hooks/use-command-palette";
import { CommandPalette } from "../command-palette";

const navigateMock = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>(
      "react-router-dom",
    );
  return {
    ...actual,
    useNavigate: () => navigateMock,
  };
});

const setThemeMock = vi.fn();

vi.mock("@/hooks/use-theme", () => ({
  useTheme: () => ({
    theme: "dark",
    resolvedTheme: "dark",
    setTheme: setThemeMock,
    isDark: true,
    isLight: false,
  }),
}));

function Trigger() {
  const { toggle } = useCommandPalette();
  return (
    <button type="button" onClick={toggle}>
      open
    </button>
  );
}

function Harness() {
  return (
    <CommandPaletteProvider>
      <Trigger />
      <CommandPalette />
    </CommandPaletteProvider>
  );
}

describe("CommandPalette", () => {
  it("navigates to /servers when the Servers entry is selected", async () => {
    navigateMock.mockClear();
    const user = userEvent.setup();
    renderWithProviders(<Harness />);

    await user.click(screen.getByRole("button", { name: /open/i }));
    const serversOption = await screen.findByRole("option", {
      name: /Servers/i,
    });
    await user.click(serversOption);

    expect(navigateMock).toHaveBeenCalledWith("/servers");
  });

  it("switches theme when the action is selected", async () => {
    setThemeMock.mockClear();
    const user = userEvent.setup();
    renderWithProviders(<Harness />);

    await user.click(screen.getByRole("button", { name: /open/i }));
    const themeOption = await screen.findByRole("option", {
      name: /Switch to light theme/i,
    });
    await user.click(themeOption);

    expect(setThemeMock).toHaveBeenCalledWith("light");
  });
});
