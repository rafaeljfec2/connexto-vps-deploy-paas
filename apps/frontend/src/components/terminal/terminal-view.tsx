import { useCallback, useEffect } from "react";
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
  const { status, error, containerRef, connect, disconnect, sendInput, focus } =
    useTerminal({
      wsUrl,
      onStatusChange,
      theme,
    });

  useEffect(() => {
    if (autoConnect) connect();
    return disconnect;
  }, [autoConnect, connect, disconnect]);

  // Schedule focus after terminal renders
  useEffect(() => {
    if (status !== "connected") return;
    const timers = [setTimeout(focus, 100), setTimeout(focus, 300)];
    return () => timers.forEach(clearTimeout);
  }, [status, focus]);

  // Global key listener for when dialog focus trap interferes
  useEffect(() => {
    if (status !== "connected") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      // Allow xterm's hidden textarea to handle input directly
      if (target.tagName === "INPUT") return;
      // Skip browser shortcuts
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

  // Fallback handler for keydown on container div
  const handleContainerKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (status !== "connected") return;
      if (isBrowserShortcut(e.nativeEvent)) return;

      const data = keyEventToData(e.nativeEvent);
      if (data === "") return;

      e.preventDefault();
      e.stopPropagation();
      sendInput(data);
    },
    [status, sendInput],
  );

  return (
    <div className={cn("relative", className)}>
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/80">
          <span className="text-destructive">{error}</span>
        </div>
      )}
      <div // NOSONAR jsx-a11y - terminal input container with role="application"
        ref={containerRef}
        role="application"
        aria-label="Terminal"
        tabIndex={0}
        className="h-full w-full outline-none cursor-text"
        onPointerDown={() => {
          focus();
        }}
        onKeyDown={handleContainerKeyDown}
      />
    </div>
  );
}
