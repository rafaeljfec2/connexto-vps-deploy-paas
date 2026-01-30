import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface IconTextProps {
  readonly icon: LucideIcon;
  readonly children: React.ReactNode;
  readonly className?: string;
  readonly iconClassName?: string;
  readonly as?: "span" | "div";
}

export function IconText({
  icon: Icon,
  children,
  className,
  iconClassName,
  as: Component = "div",
}: IconTextProps) {
  return (
    <Component
      className={cn(
        "flex items-center gap-2 text-sm text-muted-foreground",
        className,
      )}
    >
      <Icon className={cn("h-4 w-4 shrink-0", iconClassName)} />
      {children}
    </Component>
  );
}
