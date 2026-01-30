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
    <div className="flex items-center gap-4">
      {backTo && (
        <Button asChild variant="ghost" size="icon">
          <Link to={backTo}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
      )}
      <div className="flex-1">
        <div className="flex items-center gap-3">
          <h1
            className={
              backTo
                ? "text-2xl font-bold"
                : "text-3xl font-bold tracking-tight"
            }
          >
            {title}
          </h1>
          {titleSuffix}
        </div>
        {description && (
          <p className="text-muted-foreground mt-1">{description}</p>
        )}
      </div>
      {actions && <div className="flex gap-2">{actions}</div>}
    </div>
  );
}
