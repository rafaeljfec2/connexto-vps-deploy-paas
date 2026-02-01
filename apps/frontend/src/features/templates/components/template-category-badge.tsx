import { Badge } from "@/components/ui/badge";

interface TemplateCategoryBadgeProps {
  readonly category: string;
}

const categoryStyles: Record<string, string> = {
  database: "bg-blue-500/20 text-blue-400 border-blue-500/30",
  webserver: "bg-green-500/20 text-green-400 border-green-500/30",
  development: "bg-purple-500/20 text-purple-400 border-purple-500/30",
  monitoring: "bg-orange-500/20 text-orange-400 border-orange-500/30",
  messaging: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  storage: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
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
