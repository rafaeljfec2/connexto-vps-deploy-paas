import { useEffect, useState } from "react";

const DEFAULT_INTERVAL_MS = 30_000;

export function useRelativeTick(
  intervalMs: number = DEFAULT_INTERVAL_MS,
): number {
  const [now, setNow] = useState<number>(() => Date.now());

  useEffect(() => {
    let timer: number | null = null;

    const start = () => {
      if (timer !== null) return;
      timer = window.setInterval(() => setNow(Date.now()), intervalMs);
    };

    const stop = () => {
      if (timer !== null) {
        window.clearInterval(timer);
        timer = null;
      }
    };

    const handleVisibility = () => {
      if (document.visibilityState === "visible") {
        setNow(Date.now());
        start();
      } else {
        stop();
      }
    };

    if (document.visibilityState === "visible") {
      start();
    }
    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
      stop();
    };
  }, [intervalMs]);

  return now;
}
