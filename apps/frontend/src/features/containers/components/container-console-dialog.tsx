import { useCallback, useEffect, useRef, useState } from "react";
import "@xterm/xterm/css/xterm.css";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api } from "@/services/api";

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

        requestAnimationFrame(() => {
          fitAddon.fit();
          setTimeout(() => fitAddon.fit(), 100);
        });

        const wsUrl = api.containers.consoleUrl(containerId);
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.binaryType = "arraybuffer";

        ws.onopen = () => {
          if (mounted) setStatus("connected");
        };

        ws.onmessage = (event) => {
          if (termRef.current && typeof event.data === "string") {
            termRef.current.terminal.write(event.data);
          }
        };

        ws.onerror = () => {
          if (mounted) setError("WebSocket connection failed");
        };

        ws.onclose = () => {
          if (mounted) {
            setStatus("closed");
            if (termRef.current) {
              termRef.current.terminal.writeln(
                "\r\n\x1b[33mConnection closed.\x1b[0m",
              );
            }
          }
        };

        terminal.onData((data) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(data);
          }
        });

        const resizeObserver = new ResizeObserver(() => {
          fitAddon.fit();
        });
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
        <div
          ref={terminalRef}
          className="flex-1 min-h-0 p-4 overflow-hidden"
          style={{ height: "calc(80vh - 120px)" }}
        />
      </DialogContent>
    </Dialog>
  );
}
