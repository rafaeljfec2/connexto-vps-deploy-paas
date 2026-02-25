import { Activity } from "lucide-react";
import { ContainerMetrics } from "@/components/container-metrics";
import { CollapsibleSection } from "@/features/apps/components/collapsible-section";
import type { useContainerStats } from "@/features/apps/hooks/use-apps";
import { formatBytes } from "@/lib/format";

interface ResourceUsageSectionProps {
  readonly appId: string;
  readonly containerStats: ReturnType<typeof useContainerStats>["data"];
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

export function ResourceUsageSection({
  appId,
  containerStats,
  expanded,
  onToggle,
}: ResourceUsageSectionProps) {
  return (
    <CollapsibleSection
      title="Resource Usage"
      icon={Activity}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        containerStats && containerStats.cpuPercent > 0 ? (
          <span>
            CPU: {containerStats.cpuPercent.toFixed(1)}% • Memory:{" "}
            {formatBytes(containerStats.memoryUsage)} /{" "}
            {formatBytes(containerStats.memoryLimit)}
          </span>
        ) : (
          <span className="text-muted-foreground">No metrics available</span>
        )
      }
    >
      <ContainerMetrics appId={appId} embedded />
    </CollapsibleSection>
  );
}
