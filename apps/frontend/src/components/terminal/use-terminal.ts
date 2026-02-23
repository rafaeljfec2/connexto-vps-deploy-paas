import { useCallback, useEffect, useRef, useState } from "react";
import { DEFAULT_THEME } from "./terminal-utils";
import type {
  TerminalOptions,
  TerminalStatus,
  UseTerminalReturn,
} from "./types";

type Terminal = import("@xterm/xterm").Terminal;
type FitAddon = import("@xterm/addon-fit").FitAddon;

function writeTerminalData(terminal: Terminal, data: unknown): void {
  if (typeof data === "string") {
    terminal.write(data);
    return;
  }
  if (data instanceof ArrayBuffer) {
    terminal.write(new Uint8Array(data));
    return;
  }
  if (data instanceof Blob) {
    data.arrayBuffer().then((buf) => terminal.write(new Uint8Array(buf)));
  }
}

export function useTerminal(options: TerminalOptions): UseTerminalReturn {
  const { onStatusChange } = options;
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const resizeCleanupRef = useRef<(() => void) | null>(null);

  const [status, setStatus] = useState<TerminalStatus>("idle");
  const [error, setError] = useState<string | null>(null);

  const cleanup = useCallback(() => {
    resizeCleanupRef.current?.();
    resizeCleanupRef.current = null;
    wsRef.current?.close();
    wsRef.current = null;
    terminalRef.current?.dispose();
    terminalRef.current = null;
    fitAddonRef.current = null;
  }, []);

  const connect = useCallback(async () => {
    cleanup();
    setError(null);
    setStatus("connecting");

    const { Terminal } = await import("@xterm/xterm");
    const { FitAddon } = await import("@xterm/addon-fit");

    const container = containerRef.current;
    if (!container) return;

    const initTerminal = (): void => {
      const el = containerRef.current;
      if (!el) return;

      const terminal = new Terminal({
        cursorBlink: true,
        theme: options.theme ?? DEFAULT_THEME,
        fontSize: options.fontSize ?? 14,
        fontFamily: options.fontFamily ?? "ui-monospace, monospace",
        scrollback: 10000,
        convertEol: true,
      });

      const fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(el);

      terminalRef.current = terminal;
      fitAddonRef.current = fitAddon;

      const doFit = (): void => {
        fitAddon.fit();
      };
      requestAnimationFrame(doFit);
      setTimeout(doFit, 50);
      setTimeout(doFit, 200);

      const separator = options.wsUrl.includes("?") ? "&" : "?";
      const wsUrlWithSize = `${options.wsUrl}${separator}cols=${terminal.cols}&rows=${terminal.rows}`;
      const ws = new WebSocket(wsUrlWithSize);
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus("connected");
        doFit();
        setTimeout(terminal.focus.bind(terminal), 100);
      };

      ws.onmessage = (event) => {
        writeTerminalData(terminal, event.data);
      };

      ws.onerror = () => {
        setError("Connection failed");
        setStatus("error");
      };

      ws.onclose = () => {
        setStatus("closed");
        terminal.writeln("\r\n\x1b[33mConnection closed.\x1b[0m");
      };

      terminal.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(data);
      });

      terminal.onResize(({ cols: c, rows: r }) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        const payload = `\x01${JSON.stringify({ cols: c, rows: r })}`;
        ws.send(payload);
      });

      const resizeObserver = new ResizeObserver(doFit);
      resizeObserver.observe(el);
      resizeCleanupRef.current = () => resizeObserver.disconnect();
    };

    if (container.clientHeight > 0 && container.clientWidth > 0) {
      initTerminal();
      return;
    }

    const resizeObserver = new ResizeObserver(() => {
      if (!containerRef.current) return;
      if (
        containerRef.current.clientHeight > 0 &&
        containerRef.current.clientWidth > 0
      ) {
        resizeObserver.disconnect();
        resizeCleanupRef.current = null;
        initTerminal();
      }
    });
    resizeObserver.observe(container);
    resizeCleanupRef.current = () => resizeObserver.disconnect();
  }, [
    options.wsUrl,
    options.theme,
    options.fontSize,
    options.fontFamily,
    cleanup,
  ]);

  const disconnect = useCallback(() => {
    cleanup();
    setStatus("idle");
  }, [cleanup]);

  const write = useCallback((data: string) => {
    terminalRef.current?.write(data);
  }, []);

  const sendInput = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data);
    }
  }, []);

  const clear = useCallback(() => {
    terminalRef.current?.clear();
  }, []);

  const focus = useCallback(() => {
    terminalRef.current?.focus();
  }, []);

  useEffect(() => cleanup, [cleanup]);

  useEffect(() => {
    onStatusChange?.(status);
  }, [status, onStatusChange]);

  return {
    status,
    error,
    containerRef,
    connect,
    disconnect,
    write,
    sendInput,
    clear,
    focus,
  };
}
