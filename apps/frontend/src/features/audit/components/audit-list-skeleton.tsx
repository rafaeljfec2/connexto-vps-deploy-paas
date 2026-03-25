import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function AuditListSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex gap-4">
        <Skeleton className="h-10 w-[200px]" />
        <Skeleton className="h-10 w-[200px]" />
      </div>
      <Card>
        <div className="p-4 space-y-3">
          {Array.from({ length: 10 }).map((_, i) => (
            <Skeleton
              key={`audit-skeleton-${i.toString()}`}
              className="h-12 w-full"
            />
          ))}
        </div>
      </Card>
    </div>
  );
}
