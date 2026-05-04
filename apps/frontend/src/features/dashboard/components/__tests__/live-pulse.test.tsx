import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LivePulse } from "../live-pulse";

describe("LivePulse", () => {
  it("renders the syncing state while loading", () => {
    render(<LivePulse isLoading={true} />);
    expect(screen.getByText(/Syncing/i)).toBeInTheDocument();
    expect(screen.queryByText(/Live ·/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Reconnecting/i)).not.toBeInTheDocument();
  });

  it("renders the connected (Live) state after loading completes when SSE is connected", () => {
    const { rerender } = render(
      <LivePulse isLoading={true} isSSEConnected={true} />,
    );
    expect(screen.getByText(/Syncing/i)).toBeInTheDocument();

    rerender(<LivePulse isLoading={false} isSSEConnected={true} />);

    expect(screen.getByText(/Live ·/)).toBeInTheDocument();
    expect(screen.queryByText(/Reconnecting/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Syncing/i)).not.toBeInTheDocument();
  });

  it("renders the reconnecting state after loading completes when SSE is disconnected", () => {
    const { rerender } = render(
      <LivePulse isLoading={true} isSSEConnected={false} />,
    );
    expect(screen.getByText(/Syncing/i)).toBeInTheDocument();

    rerender(<LivePulse isLoading={false} isSSEConnected={false} />);

    expect(screen.getByText(/Reconnecting/i)).toBeInTheDocument();
    expect(screen.queryByText(/Live ·/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Syncing/i)).not.toBeInTheDocument();
  });

  it("falls back to the connected (Live) state when isSSEConnected is omitted (legacy)", () => {
    const { rerender } = render(<LivePulse isLoading={true} />);
    expect(screen.getByText(/Syncing/i)).toBeInTheDocument();

    rerender(<LivePulse isLoading={false} />);

    expect(screen.getByText(/Live ·/)).toBeInTheDocument();
    expect(screen.queryByText(/Reconnecting/i)).not.toBeInTheDocument();
  });

  it("uses an aria-live region so screen readers announce status changes", () => {
    const { container } = render(
      <LivePulse isLoading={false} isSSEConnected={false} />,
    );
    expect(container.querySelector("[aria-live=polite]")).not.toBeNull();
  });
});
