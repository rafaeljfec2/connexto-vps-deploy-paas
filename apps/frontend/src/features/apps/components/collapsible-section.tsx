import { ChevronDown, ChevronRight } from "lucide-react";
import { CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface CollapsibleSectionProps {
  readonly title: string;
  readonly icon: React.ComponentType<{ className?: string }>;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly summary?: React.ReactNode;
  readonly actions?: React.ReactNode;
  readonly children: React.ReactNode;
}

export function CollapsibleSection({
  title,
  icon: Icon,
  expanded,
  onToggle,
  summary,
  actions,
  children,
}: CollapsibleSectionProps) {
  return (
    <div>
      <CardHeader
        className={cn(
          "flex flex-col sm:flex-row sm:items-center justify-between cursor-pointer select-none transition-colors hover:bg-muted/50 gap-2 sm:gap-3 p-4",
          !expanded && "pb-4",
        )}
        onClick={onToggle}
      >
        <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0">
          <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
            {expanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
            <Icon className="h-4 w-4 text-muted-foreground" />
          </div>
          <CardTitle className="text-sm sm:text-base">{title}</CardTitle>
          {!expanded && summary && (
            <div className="hidden sm:flex items-center gap-2 text-sm text-muted-foreground ml-2 truncate">
              <span className="text-muted-foreground/50">—</span>
              {summary}
            </div>
          )}
        </div>
        {actions && (
          <div
            className="flex items-center gap-2 ml-7 sm:ml-0"
            onPointerDown={(e) => e.stopPropagation()}
          >
            {actions}
          </div>
        )}
      </CardHeader>
      <div
        className={cn(
          "overflow-hidden transition-all duration-300",
          expanded ? "max-h-[1000px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <CardContent className="pt-0 sm:pt-0 px-4 pb-4">{children}</CardContent>
      </div>
    </div>
  );
}
