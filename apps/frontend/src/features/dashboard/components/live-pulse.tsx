import { useEffect, useRef } from "react";
import { useRelativeTick } from "@/hooks/use-relative-tick";
import { formatRelativeTime } from "@/lib/format";

interface LivePulseProps {
  readonly isLoading: boolean;
}

export function LivePulse({ isLoading }: LivePulseProps) {
  const lastUpdateRef = useRef<Date | null>(null);
  useRelativeTick();

  useEffect(() => {
    if (!isLoading) {
      lastUpdateRef.current = new Date();
    }
  }, [isLoading]);

  if (isLoading || !lastUpdateRef.current) {
    return (
      <span className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
        <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/60" />
        Syncing…
      </span>
    );
  }

  return (
    <span
      className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground"
      aria-live="polite"
    >
      <span className="relative flex h-1.5 w-1.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-500/70" />
        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
      </span>
      <span>Live · updated {formatRelativeTime(lastUpdateRef.current)}</span>
    </span>
  );
}
