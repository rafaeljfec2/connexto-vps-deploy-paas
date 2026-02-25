import type { ElementType } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface StatCardProps {
  readonly icon: ElementType;
  readonly title: string;
  readonly value: string | number;
  readonly subtitle?: string;
  readonly accentColor?: "default" | "success" | "warning" | "destructive";
  readonly isLoading?: boolean;
}

const accentClasses: Record<
  NonNullable<StatCardProps["accentColor"]>,
  string
> = {
  default: "text-primary",
  success: "text-emerald-500",
  warning: "text-yellow-500",
  destructive: "text-red-500",
};

export function StatCard({
  icon: Icon,
  title,
  value,
  subtitle,
  accentColor = "default",
  isLoading = false,
}: StatCardProps) {
  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 rounded-lg" />
            <div className="space-y-1.5">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-6 w-10" />
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="transition-colors hover:bg-accent/30">
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
            <Icon className={cn("h-5 w-5", accentClasses[accentColor])} />
          </div>
          <div className="min-w-0">
            <p className="text-xs font-medium text-muted-foreground">{title}</p>
            <p className="text-2xl font-semibold tracking-tight">{value}</p>
            {subtitle && (
              <p className="truncate text-xs text-muted-foreground/80">
                {subtitle}
              </p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
