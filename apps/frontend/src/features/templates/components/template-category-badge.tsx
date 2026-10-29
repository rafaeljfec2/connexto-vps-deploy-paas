import { Badge } from "@/components/ui/badge";

interface TemplateCategoryBadgeProps {
  readonly category: string;
}

const categoryStyles: Record<string, string> = {
  database:
    "bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30",
  webserver:
    "bg-status-success/10 text-status-success-text border-status-success/30",
  development:
    "bg-purple-500/20 text-purple-600 dark:text-purple-400 border-purple-500/30",
  monitoring:
    "bg-orange-500/20 text-orange-600 dark:text-orange-400 border-orange-500/30",
  messaging:
    "bg-status-running/10 text-status-running-text border-status-running/30",
  storage: "bg-cyan-500/20 text-cyan-600 dark:text-cyan-400 border-cyan-500/30",
};

export function TemplateCategoryBadge({
  category,
}: TemplateCategoryBadgeProps) {
  const style =
    categoryStyles[category] ??
    "bg-gray-500/20 text-gray-400 border-gray-500/30";

  return (
    <Badge variant="outline" className={`text-[10px] ${style}`}>
      {category}
    </Badge>
  );
}
