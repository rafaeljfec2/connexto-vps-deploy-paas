import { HeartPulse, Play, RefreshCw, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import { HealthDetail } from "@/components/health-indicator";
import { CollapsibleSection } from "@/features/apps/components/collapsible-section";
import type { useAppActions } from "@/features/apps/hooks/use-app-actions";
import type { useAppHealth } from "@/hooks/use-sse";
import { getHealthTextColor } from "@/lib/status";

interface ContainerHealthSectionProps {
  readonly appId: string;
  readonly health: ReturnType<typeof useAppHealth>["data"];
  readonly actions: ReturnType<typeof useAppActions>;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

export function ContainerHealthSection({
  appId,
  health,
  actions,
  expanded,
  onToggle,
}: ContainerHealthSectionProps) {
  return (
    <CollapsibleSection
      title="Container Health"
      icon={HeartPulse}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <div className="flex items-center gap-2">
          {health?.status === "running" ? (
            <>
              <span className={getHealthTextColor(health.health)}>
                {health.health}
              </span>
              {health.uptime && (
                <span className="text-muted-foreground">
                  • Uptime: {health.uptime}
                </span>
              )}
            </>
          ) : (
            <span className="text-muted-foreground">
              {health?.status ?? "unknown"}
            </span>
          )}
        </div>
      }
      actions={
        <>
          <Button
            variant="outline"
            size="sm"
            onClick={() => actions.restartContainer.mutate(appId)}
            disabled={
              actions.restartContainer.isPending ||
              health?.status === "not_found"
            }
          >
            <RefreshCw
              className={`h-4 w-4 sm:mr-1 ${actions.restartContainer.isPending ? "animate-spin" : ""}`}
            />
            <span className="hidden sm:inline">Restart</span>
          </Button>
          {health?.status === "running" ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => actions.stopContainer.mutate(appId)}
              disabled={actions.stopContainer.isPending}
            >
              <Square className="h-4 w-4 sm:mr-1" />
              <span className="hidden sm:inline">Stop</span>
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => actions.startContainer.mutate(appId)}
              disabled={
                actions.startContainer.isPending ||
                health?.status === "not_found"
              }
            >
              <Play className="h-4 w-4 sm:mr-1" />
              <span className="hidden sm:inline">Start</span>
            </Button>
          )}
        </>
      }
    >
      <HealthDetail health={health} />
    </CollapsibleSection>
  );
}
