import type { ElementType } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";

interface PageHeaderProps {
  readonly title: string;
  readonly description?: React.ReactNode;
  readonly backTo?: string;
  readonly actions?: React.ReactNode;
  readonly titleSuffix?: React.ReactNode;
  readonly icon?: ElementType;
}

export function PageHeader({
  title,
  description,
  backTo,
  actions,
  titleSuffix,
  icon: Icon,
}: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col lg:flex-row lg:items-center gap-3 lg:gap-4">
        <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
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
          <div className="flex flex-wrap items-center gap-2 sm:gap-3 min-w-0">
            {Icon && <Icon className="h-6 w-6 text-muted-foreground" />}
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
        </div>
        {actions && (
          <div className="flex flex-wrap gap-2 shrink-0">{actions}</div>
        )}
      </div>
      {description && (
        <div className="text-sm sm:text-base text-muted-foreground ml-0 lg:ml-10">
          {description}
        </div>
      )}
    </div>
  );
}
