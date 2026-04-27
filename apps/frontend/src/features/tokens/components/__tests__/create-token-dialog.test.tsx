import { renderWithProviders, screen, waitFor } from "@/test/test-utils";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CreateTokenDialog } from "../create-token-dialog";

const fetchMock = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  fetchMock.mockReset();
});

function mockSuccessfulCreate(plaintext = "pdp_live_demoplaintexttoken") {
  fetchMock.mockResolvedValueOnce({
    ok: true,
    status: 201,
    json: async () => ({
      success: true,
      error: null,
      meta: {},
      data: {
        token: {
          id: "tok-1",
          name: "demo",
          tokenPrefix: "pdp_live_demoplain",
          scopes: ["read"],
          createdAt: "2026-04-26T00:00:00Z",
        },
        plaintextToken: plaintext,
      },
    }),
  });
}

describe("CreateTokenDialog", () => {
  it("submits form payload and reveals plaintext token only once", async () => {
    const user = userEvent.setup();
    mockSuccessfulCreate("pdp_live_freshlygenerated");
    const onOpenChange = vi.fn();

    renderWithProviders(
      <CreateTokenDialog open={true} onOpenChange={onOpenChange} />,
    );

    await user.type(screen.getByLabelText(/name/i), "demo");
    await user.click(screen.getByRole("button", { name: /create token/i }));

    await waitFor(
      () => {
        expect(
          screen.getByText("pdp_live_freshlygenerated"),
        ).toBeInTheDocument();
      },
      { timeout: 3000 },
    );

    expect(screen.getByText(/copy your token/i)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const firstCall = fetchMock.mock.calls[0];
    if (!firstCall) {
      throw new Error("expected fetch to be called");
    }
    const init = firstCall[1] as { body: string };
    const body = JSON.parse(init.body);
    expect(body.name).toBe("demo");
    expect(body.scopes).toEqual(["read"]);
    expect(typeof body.expiresAt).toBe("string");
  });

  it("blocks submission when name is too short", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <CreateTokenDialog open={true} onOpenChange={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(/name/i), "ab");
    await user.click(screen.getByRole("button", { name: /create token/i }));

    expect(
      await screen.findByText(/name must be between 3 and 120 characters/i),
    ).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("rejects when no scopes are selected", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <CreateTokenDialog open={true} onOpenChange={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(/name/i), "valid-name");
    await user.click(screen.getByRole("checkbox", { name: /read/i }));

    await user.click(screen.getByRole("button", { name: /create token/i }));

    expect(
      await screen.findByText(/select at least one scope/i),
    ).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
