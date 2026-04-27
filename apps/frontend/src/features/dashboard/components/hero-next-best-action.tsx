import { Link } from "react-router-dom";
import { AlertTriangle, ArrowRight } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import type { NextBestAction } from "../lib/hero-insights";

interface HeroNextBestActionProps {
  readonly action: NextBestAction | null;
}

const titleByseverity: Record<NextBestAction["severity"], string> = {
  destructive: "Action required",
  warning: "Needs your attention",
};

export function HeroNextBestAction({ action }: HeroNextBestActionProps) {
  if (!action) return null;

  const isDestructive = action.severity === "destructive";

  return (
    <Alert
      variant={isDestructive ? "destructive" : "default"}
      className={cn(
        "mb-5 flex items-start gap-3",
        !isDestructive &&
          "border-yellow-500/40 bg-yellow-500/5 text-yellow-700 dark:border-yellow-500/30 dark:text-yellow-300",
      )}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="flex-1 space-y-1">
        <AlertTitle className="text-sm font-semibold">
          {titleByseverity[action.severity]}
        </AlertTitle>
        <AlertDescription className="text-xs sm:text-sm">
          {action.message}
        </AlertDescription>
      </div>
      <Link
        to={action.ctaHref}
        className="inline-flex shrink-0 items-center gap-1 self-center text-xs font-medium underline-offset-4 hover:underline"
      >
        {action.ctaLabel}
        <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
      </Link>
    </Alert>
  );
}
