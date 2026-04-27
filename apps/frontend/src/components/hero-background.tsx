import { cn } from "@/lib/utils";

interface HeroBackgroundProps {
  readonly className?: string;
  readonly glowClassName?: string;
  readonly dotOpacityClassName?: string;
}

export function HeroBackground({
  className,
  glowClassName,
  dotOpacityClassName,
}: HeroBackgroundProps) {
  return (
    <>
      <div
        aria-hidden="true"
        className={cn(
          "pointer-events-none absolute inset-0",
          dotOpacityClassName ?? "opacity-60 dark:opacity-30",
          className,
        )}
        style={{
          backgroundImage:
            "radial-gradient(circle, hsl(var(--muted-foreground) / 0.07) 1px, transparent 1px)",
          backgroundSize: "24px 24px",
        }}
      />
      <div
        aria-hidden="true"
        className={cn(
          "pointer-events-none absolute -right-32 -top-32 h-[420px] w-[420px] rounded-full bg-emerald-400/10 blur-3xl animate-glow-pulse dark:bg-emerald-500/10",
          glowClassName,
        )}
      />
    </>
  );
}
