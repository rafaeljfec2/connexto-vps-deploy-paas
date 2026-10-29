import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useTheme } from "../use-theme";

function systemPrefersDark(prefersDark: boolean): void {
  vi.mocked(window.matchMedia).mockImplementation(
    (query: string) =>
      ({
        matches: prefersDark && query.includes("dark"),
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as unknown as MediaQueryList,
  );
}

describe("useTheme", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    systemPrefersDark(true);
  });

  it("defaults to the light theme even when the OS prefers dark", () => {
    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("light");
    expect(result.current.resolvedTheme).toBe("light");
    expect(result.current.isLight).toBe(true);
    expect(document.documentElement.classList.contains("light")).toBe(true);
  });

  it("resolves the stored dark theme from localStorage", () => {
    localStorage.setItem("theme", "dark");

    const { result } = renderHook(() => useTheme());

    expect(result.current.resolvedTheme).toBe("dark");
    expect(result.current.isDark).toBe(true);
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("follows the OS preference when the stored theme is system", () => {
    localStorage.setItem("theme", "system");
    systemPrefersDark(true);

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("system");
    expect(result.current.resolvedTheme).toBe("dark");
  });
});
