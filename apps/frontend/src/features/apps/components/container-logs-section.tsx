import { Terminal } from "lucide-react";
import { ContainerLogsViewer } from "@/components/container-logs-viewer";
import { CollapsibleSection } from "./collapsible-section";

interface ContainerLogsSectionProps {
  readonly appId: string;
  readonly appName: string;
  readonly health: { status?: string } | null | undefined;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

export function ContainerLogsSection({
  appId,
  appName,
  health,
  expanded,
  onToggle,
}: ContainerLogsSectionProps) {
  return (
    <CollapsibleSection
      title="Container Logs"
      icon={Terminal}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        health?.status === "running" ? (
          <span className="text-status-success">Container running</span>
        ) : (
          <span className="text-muted-foreground">
            Container {health?.status ?? "unknown"}
          </span>
        )
      }
    >
      <ContainerLogsViewer appId={appId} appName={appName} />
    </CollapsibleSection>
  );
}
