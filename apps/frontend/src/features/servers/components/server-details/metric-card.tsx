import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

function getUsageColor(percent: number): string {
  if (percent < 60) return "bg-status-success";
  if (percent < 80) return "bg-status-running";
  return "bg-status-failed";
}

function getUsageTextColor(percent: number): string {
  if (percent < 60) return "text-status-success-text";
  if (percent < 80) return "text-status-running-text";
  return "text-status-failed-text";
}

interface MetricCardProps {
  readonly icon: React.ComponentType<{ className?: string }>;
  readonly title: string;
  readonly value: string;
  readonly subValue?: string;
  readonly percent?: number;
}

export function MetricCard({
  icon: Icon,
  title,
  value,
  subValue,
  percent,
}: MetricCardProps) {
  return (
    <Card>
      <CardContent className="p-3">
        <div className="flex items-center gap-2 text-muted-foreground mb-1.5">
          <span className="bg-muted rounded-full p-1.5">
            <Icon className="h-3.5 w-3.5" />
          </span>
          <span className="text-[11px] font-medium">{title}</span>
        </div>
        <div className="flex items-baseline gap-1.5">
          <span
            className={cn(
              "text-base font-semibold leading-tight",
              percent !== undefined && getUsageTextColor(percent),
            )}
          >
            {value}
          </span>
          {subValue != null && (
            <span className="text-[11px] text-muted-foreground/80">
              {subValue}
            </span>
          )}
        </div>
        {percent !== undefined && (
          <div className="mt-2 h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all",
                getUsageColor(percent),
              )}
              style={{ width: `${Math.min(percent, 100)}%` }}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
