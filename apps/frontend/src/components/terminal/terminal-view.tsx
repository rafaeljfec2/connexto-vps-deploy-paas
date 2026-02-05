import { useEffect } from "react";
import "@xterm/xterm/css/xterm.css";
import { cn } from "@/lib/utils";
import { isBrowserShortcut, keyEventToData } from "./terminal-utils";
import type { TerminalStatus, TerminalTheme } from "./types";
import { useTerminal } from "./use-terminal";

export interface TerminalViewProps {
  readonly wsUrl: string;
  readonly autoConnect?: boolean;
  readonly className?: string;
  readonly onStatusChange?: (status: TerminalStatus) => void;
  readonly theme?: TerminalTheme;
}

export function TerminalView({
  wsUrl,
  autoConnect = true,
  className,
  onStatusChange,
  theme,
}: TerminalViewProps) {
  const { status, error, containerRef, connect, disconnect, sendInput } =
    useTerminal({
      wsUrl,
      onStatusChange,
      theme,
    });

  useEffect(() => {
    if (autoConnect) connect();
    return disconnect;
  }, [autoConnect, connect, disconnect]);

  useEffect(() => {
    if (status !== "connected") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;
      if (isBrowserShortcut(event)) return;

      const data = keyEventToData(event);
      if (data === "") return;

      event.preventDefault();
      event.stopPropagation();
      sendInput(data);
    };

    globalThis.addEventListener("keydown", handleKeyDown, { capture: true });
    return () =>
      globalThis.removeEventListener("keydown", handleKeyDown, {
        capture: true,
      });
  }, [status, sendInput]);

  return (
    <div className={cn("relative", className)}>
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/80">
          <span className="text-destructive">{error}</span>
        </div>
      )}
      <div
        ref={containerRef}
        role="application"
        aria-label="Terminal"
        tabIndex={0} // NOSONAR jsx-a11y/no-noninteractive-tabindex - focus target for terminal key input
        className="h-full w-full outline-none"
        onPointerDownCapture={(e) => {
          e.preventDefault();
          containerRef.current?.focus();
        }}
      />
    </div>
  );
}
