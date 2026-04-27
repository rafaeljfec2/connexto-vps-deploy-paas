import type { ElementType } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { KpiTone } from "../lib/kpi-tone";

type KpiEmphasis = "default" | "primary";

interface KpiChipProps {
  readonly icon: ElementType;
  readonly label: string;
  readonly value: string | number;
  readonly subtitle?: string;
  readonly tone?: KpiTone;
  readonly emphasis?: KpiEmphasis;
  readonly isLoading?: boolean;
}

const toneClasses: Record<KpiTone, string> = {
  default: "text-primary",
  success: "text-emerald-500 dark:text-emerald-400",
  warning: "text-yellow-500 dark:text-yellow-400",
  destructive: "text-red-500 dark:text-red-400",
};

const toneBorder: Record<KpiTone, string> = {
  default: "border-border/60",
  success: "border-border/60",
  warning: "border-yellow-500/40",
  destructive: "border-red-500/50",
};

const toneIconBg: Record<KpiTone, string> = {
  default: "bg-muted",
  success: "bg-emerald-500/10",
  warning: "bg-yellow-500/10",
  destructive: "bg-red-500/10",
};

export function KpiChip({
  icon: Icon,
  label,
  value,
  subtitle,
  tone = "default",
  emphasis = "default",
  isLoading = false,
}: KpiChipProps) {
  const isPrimary = emphasis === "primary";

  const containerClass = cn(
    "flex items-center gap-3 rounded-lg border bg-background/60 backdrop-blur transition-colors hover:bg-background/80",
    toneBorder[tone],
    isPrimary ? "min-w-[14rem] px-4 py-3" : "min-w-[9rem] px-3 py-2",
  );

  if (isLoading) {
    return (
      <div className={containerClass} aria-busy="true">
        <Skeleton
          className={cn("rounded-md", isPrimary ? "h-10 w-10" : "h-8 w-8")}
        />
        <div className="space-y-1.5">
          <Skeleton className="h-3 w-14" />
          <Skeleton className={cn(isPrimary ? "h-6 w-14" : "h-4 w-10")} />
        </div>
      </div>
    );
  }

  return (
    <div className={containerClass}>
      <div
        className={cn(
          "flex shrink-0 items-center justify-center rounded-md",
          toneIconBg[tone],
          isPrimary ? "h-10 w-10" : "h-8 w-8",
        )}
      >
        <Icon
          className={cn(isPrimary ? "h-5 w-5" : "h-4 w-4", toneClasses[tone])}
          aria-hidden="true"
        />
      </div>
      <div className="min-w-0">
        <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        <p className="flex items-baseline gap-1.5">
          <span
            className={cn(
              "font-semibold leading-none tracking-tight",
              isPrimary ? "text-2xl" : "text-lg",
            )}
          >
            {value}
          </span>
          {subtitle && (
            <span className="truncate text-[11px] text-muted-foreground/80">
              {subtitle}
            </span>
          )}
        </p>
      </div>
    </div>
  );
}
