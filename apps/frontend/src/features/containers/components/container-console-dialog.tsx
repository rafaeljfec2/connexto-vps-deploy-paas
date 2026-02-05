import { useCallback, useEffect, useRef, useState } from "react";
import "@xterm/xterm/css/xterm.css";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api } from "@/services/api";

function scheduleTerminalFit(
  fitAddon: import("@xterm/addon-fit").FitAddon,
): void {
  requestAnimationFrame(() => fitAddon.fit());
  setTimeout(() => fitAddon.fit(), 100);
}

function keyEventToData(event: KeyboardEvent): string {
  if (event.ctrlKey || event.altKey || event.metaKey) {
    if (event.ctrlKey && event.key.length === 1) {
      const code = (event.key.toUpperCase().codePointAt(0) ?? 0) - 64;
      if (code >= 1 && code <= 26) return String.fromCodePoint(code);
    }
    return "";
  }
  switch (event.key) {
    case "Enter":
      return "\r";
    case "Backspace":
      return "\x7f";
    case "Tab":
      return "\t";
    case "ArrowUp":
      return "\x1b[A";
    case "ArrowDown":
      return "\x1b[B";
    case "ArrowRight":
      return "\x1b[C";
    case "ArrowLeft":
      return "\x1b[D";
    default:
      return event.key.length === 1 ? event.key : "";
  }
}

function scheduleTerminalFocus(
  terminalRef: React.RefObject<HTMLDivElement | null>,
): void {
  setTimeout(() => terminalRef.current?.focus(), 300);
  setTimeout(() => terminalRef.current?.focus(), 600);
}

interface ContainerConsoleDialogProps {
  readonly containerId: string | null;
  readonly containerName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function ContainerConsoleDialog({
  containerId,
  containerName,
  open,
  onOpenChange,
}: ContainerConsoleDialogProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<"connecting" | "connected" | "closed">(
    "closed",
  );
  const wsRef = useRef<WebSocket | null>(null);
  const termRef = useRef<{
    terminal: import("@xterm/xterm").Terminal;
    fit: import("@xterm/addon-fit").FitAddon;
  } | null>(null);

  const cleanup = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    if (termRef.current) {
      termRef.current.terminal.dispose();
      termRef.current = null;
    }
    setStatus("closed");
  }, []);

  useEffect(() => {
    if (!open || !containerId) return cleanup;

    let mounted = true;

    function handleWsOpen(): void {
      if (!mounted) return;
      setStatus("connected");
      scheduleTerminalFocus(terminalRef);
      setTimeout(() => terminalRef.current?.focus(), 100);
    }

    function handleWsMessage(event: MessageEvent): void {
      if (termRef.current && typeof event.data === "string") {
        termRef.current.terminal.write(event.data);
      }
    }

    function handleWsError(): void {
      if (mounted) setError("WebSocket connection failed");
    }

    function handleWsClose(): void {
      if (!mounted) return;
      setStatus("closed");
      if (termRef.current) {
        termRef.current.terminal.writeln(
          "\r\n\x1b[33mConnection closed.\x1b[0m",
        );
      }
    }

    function createOnDataHandler(ws: WebSocket): (data: string) => void {
      return (data) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(data);
      };
    }

    function createResizeCallback(
      fit: import("@xterm/addon-fit").FitAddon,
    ): () => void {
      return () => fit.fit();
    }

    const connect = async () => {
      setError(null);
      setStatus("connecting");

      try {
        const { Terminal } = await import("@xterm/xterm");
        const { FitAddon } = await import("@xterm/addon-fit");

        if (!terminalRef.current || !mounted) return;

        const terminal = new Terminal({
          cursorBlink: true,
          theme: {
            background: "#0f172a",
            foreground: "#e2e8f0",
            cursor: "#94a3b8",
          },
          fontFamily: "ui-monospace, monospace",
          fontSize: 14,
        });

        const fitAddon = new FitAddon();
        terminal.loadAddon(fitAddon);
        terminal.open(terminalRef.current);
        termRef.current = { terminal, fit: fitAddon };
        scheduleTerminalFit(fitAddon);
        scheduleTerminalFocus(terminalRef);

        const wsUrl = api.containers.consoleUrl(containerId);
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.binaryType = "arraybuffer";
        ws.onopen = handleWsOpen;
        ws.onmessage = handleWsMessage;
        ws.onerror = handleWsError;
        ws.onclose = handleWsClose;

        terminal.onData(createOnDataHandler(ws));

        const resizeObserver = new ResizeObserver(
          createResizeCallback(fitAddon),
        );
        resizeObserver.observe(terminalRef.current);
      } catch (err) {
        if (mounted) {
          setError(
            err instanceof Error ? err.message : "Failed to load terminal",
          );
        }
      }
    };

    connect();

    return () => {
      mounted = false;
      cleanup();
    };
  }, [open, containerId, cleanup]);

  useEffect(() => {
    if (!open || status !== "connected") return;

    const handleGlobalKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;

      if (
        event.ctrlKey &&
        ["r", "w", "t", "n"].includes(event.key.toLowerCase())
      )
        return;
      if (
        event.metaKey &&
        ["r", "w", "t", "n"].includes(event.key.toLowerCase())
      )
        return;

      if (wsRef.current?.readyState !== WebSocket.OPEN) return;

      event.preventDefault();
      event.stopPropagation();

      const data = keyEventToData(event);
      if (data !== "") wsRef.current.send(data);
    };

    globalThis.addEventListener("keydown", handleGlobalKeyDown, {
      capture: true,
    });

    return () => {
      globalThis.removeEventListener("keydown", handleGlobalKeyDown, {
        capture: true,
      });
    };
  }, [open, status]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) cleanup();
      onOpenChange(next);
    },
    [cleanup, onOpenChange],
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-4xl h-[80vh] flex flex-col p-0">
        <DialogHeader className="px-6 pt-6 pb-2">
          <DialogTitle className="flex items-center gap-2">
            Console - {containerName}
            {status === "connecting" && (
              <span className="text-sm font-normal text-muted-foreground">
                Connecting...
              </span>
            )}
            {status === "connected" && (
              <span className="text-sm font-normal text-green-600">
                Connected
              </span>
            )}
          </DialogTitle>
        </DialogHeader>
        {error && (
          <div className="px-6 py-2 text-sm text-destructive">{error}</div>
        )}
        {/* Focus target; keys forwarded to WebSocket via onKeyDown */}
        <div // NOSONAR jsx-a11y/no-static-element-interactions - focus target for terminal (role=application)
          ref={terminalRef}
          role="application"
          aria-label="Container shell"
          tabIndex={0} // NOSONAR - focus target for embedded terminal (role=application)
          className="flex-1 min-h-0 p-4 overflow-hidden cursor-text outline-none"
          style={{ height: "calc(80vh - 120px)" }}
          onPointerDownCapture={(e) => {
            e.preventDefault();
            terminalRef.current?.focus();
          }}
          onKeyDown={(e) => {
            // Fallback se o listener global nao capturar
            if (
              wsRef.current?.readyState !== WebSocket.OPEN ||
              !termRef.current
            )
              return;
            e.preventDefault();
            e.stopPropagation();
            const data = keyEventToData(e.nativeEvent);
            if (data !== "") wsRef.current.send(data);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
