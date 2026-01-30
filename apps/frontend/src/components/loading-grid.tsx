import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface LoadingGridProps {
  readonly count?: number;
  readonly itemHeight?: string;
  readonly columns?: 1 | 2 | 3 | 4;
  readonly gap?: "sm" | "md" | "lg";
  readonly className?: string;
}

const columnClasses = {
  1: "grid-cols-1",
  2: "grid-cols-1 md:grid-cols-2",
  3: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
  4: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4",
};

const gapClasses = {
  sm: "gap-2",
  md: "gap-4",
  lg: "gap-6",
};

export function LoadingGrid({
  count = 6,
  itemHeight = "h-40",
  columns = 3,
  gap = "md",
  className,
}: LoadingGridProps) {
  return (
    <div
      className={cn("grid", columnClasses[columns], gapClasses[gap], className)}
    >
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} className={itemHeight} />
      ))}
    </div>
  );
}
