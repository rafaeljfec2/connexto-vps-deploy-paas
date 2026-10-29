import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "../status-badge";

describe("StatusBadge", () => {
  it("renders Deployed label for success status", () => {
    render(<StatusBadge status="success" />);
    expect(screen.getByText("Deployed")).toBeInTheDocument();
  });

  it("renders Failed label for failed status", () => {
    render(<StatusBadge status="failed" />);
    expect(screen.getByText("Failed")).toBeInTheDocument();
  });

  it("renders Building / Deploying label for running status", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("Building")).toBeInTheDocument();
  });

  it("renders Pending label for pending status", () => {
    render(<StatusBadge status="pending" />);
    expect(screen.getByText("Pending")).toBeInTheDocument();
  });
});
