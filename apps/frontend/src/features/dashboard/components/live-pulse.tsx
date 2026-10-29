import { useEffect, useState } from "react";
import { useRelativeTick } from "@/hooks/use-relative-tick";
import { formatRelativeTime } from "@/lib/format";

interface LivePulseProps {
  readonly isLoading: boolean;
  /**
   * Real SSE connection state. When provided, the indicator honestly reflects
   * whether the server-sent events stream is open (green "Live"), reconnecting
   * (amber "Reconnecting…"), or still loading initial data (muted "Syncing…").
   * When omitted, the component falls back to the legacy "always connected
   * once loaded" behavior to preserve backward compatibility.
   */
  readonly isSSEConnected?: boolean;
}

export function LivePulse({ isLoading, isSSEConnected }: LivePulseProps) {
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  useRelativeTick();

  useEffect(() => {
    if (!isLoading) {
      setLastUpdate(new Date());
    }
  }, [isLoading]);

  if (isLoading || !lastUpdate) {
    return (
      <span
        className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground"
        aria-live="polite"
      >
        <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/60" />
        Syncing…
      </span>
    );
  }

  if (isSSEConnected === false) {
    return (
      <span
        className="inline-flex items-center gap-1.5 text-[11px] text-status-running-text"
        aria-live="polite"
      >
        <span className="h-1.5 w-1.5 rounded-full bg-status-running" />
        Reconnecting…
      </span>
    );
  }

  return (
    <span
      className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground"
      aria-live="polite"
    >
      <span className="relative flex h-1.5 w-1.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-status-success/70" />
        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-status-success" />
      </span>
      <span>Live · updated {formatRelativeTime(lastUpdate)}</span>
    </span>
  );
}
