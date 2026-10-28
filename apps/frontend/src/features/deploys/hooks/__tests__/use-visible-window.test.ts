import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PAGE_SIZE, useVisibleWindow } from "../use-visible-window";

describe("useVisibleWindow", () => {
  it("starts at PAGE_SIZE", () => {
    const { result } = renderHook(() =>
      useVisibleWindow({ totalCount: 20, resetKey: "a" }),
    );

    expect(PAGE_SIZE).toBe(5);
    expect(result.current.visibleCount).toBe(5);
    expect(result.current.hasMore).toBe(true);
  });

  it("loadMore increases by PAGE_SIZE", () => {
    const { result } = renderHook(() =>
      useVisibleWindow({ totalCount: 20, resetKey: "a" }),
    );

    act(() => {
      result.current.loadMore();
    });

    expect(result.current.visibleCount).toBe(10);
    expect(result.current.hasMore).toBe(true);
  });

  it("does not exceed totalCount", () => {
    const { result } = renderHook(() =>
      useVisibleWindow({ totalCount: 7, resetKey: "a" }),
    );

    act(() => {
      result.current.loadMore();
    });

    expect(result.current.visibleCount).toBe(7);
    expect(result.current.hasMore).toBe(false);

    act(() => {
      result.current.loadMore();
    });

    expect(result.current.visibleCount).toBe(7);
  });

  it("resets to PAGE_SIZE when resetKey changes", () => {
    const { result, rerender } = renderHook(
      ({ resetKey }) => useVisibleWindow({ totalCount: 20, resetKey }),
      { initialProps: { resetKey: "a" } },
    );

    act(() => {
      result.current.loadMore();
    });
    expect(result.current.visibleCount).toBe(10);

    rerender({ resetKey: "b" });

    expect(result.current.visibleCount).toBe(5);
    expect(result.current.hasMore).toBe(true);
  });

  it("clamps visibleCount when totalCount shrinks below window", () => {
    const { result, rerender } = renderHook(
      ({ totalCount }) => useVisibleWindow({ totalCount, resetKey: "stable" }),
      { initialProps: { totalCount: 20 } },
    );

    act(() => {
      result.current.loadMore();
      result.current.loadMore();
    });
    expect(result.current.visibleCount).toBe(15);

    rerender({ totalCount: 8 });

    expect(result.current.visibleCount).toBe(8);
    expect(result.current.hasMore).toBe(false);
  });
});
