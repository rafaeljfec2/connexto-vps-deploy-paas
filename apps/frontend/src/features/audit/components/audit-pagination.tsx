import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface AuditPaginationProps {
  readonly total: number;
  readonly offset: number;
  readonly limit: number;
  readonly currentPage: number;
  readonly totalPages: number;
  readonly label: string;
  readonly onPrevPage: () => void;
  readonly onNextPage: () => void;
}

export function AuditPagination({
  total,
  offset,
  limit,
  currentPage,
  totalPages,
  label,
  onPrevPage,
  onNextPage,
}: Readonly<AuditPaginationProps>) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">
        Showing {offset + 1} - {Math.min(offset + limit, total)} of {total}{" "}
        {label}
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onPrevPage}
          disabled={offset === 0}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <span className="text-sm">
          Page {currentPage} of {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={onNextPage}
          disabled={offset + limit >= total}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
