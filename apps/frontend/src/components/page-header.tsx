import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";

interface PageHeaderProps {
  readonly title: string;
  readonly description?: React.ReactNode;
  readonly backTo?: string;
  readonly actions?: React.ReactNode;
  readonly titleSuffix?: React.ReactNode;
}

export function PageHeader({
  title,
  description,
  backTo,
  actions,
  titleSuffix,
}: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex items-start gap-3 sm:gap-4">
        {backTo && (
          <Button
            asChild
            variant="ghost"
            size="icon"
            className="shrink-0 -ml-2 sm:ml-0"
          >
            <Link to={backTo}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
        )}
        <div className="flex-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <h1
              className={
                backTo
                  ? "text-xl sm:text-2xl font-bold truncate"
                  : "text-2xl sm:text-3xl font-bold tracking-tight"
              }
            >
              {title}
            </h1>
            {titleSuffix}
          </div>
          {description && (
            <div className="text-sm sm:text-base text-muted-foreground mt-1">
              {description}
            </div>
          )}
        </div>
      </div>
      {actions && (
        <div className="flex flex-wrap gap-2 -mx-1 sm:mx-0">{actions}</div>
      )}
    </div>
  );
}
