import { Key } from "lucide-react";
import { CollapsibleSection } from "@/features/apps/components/collapsible-section";
import { EnvVarsManager } from "@/features/apps/components/env-vars-manager";

interface EnvVarsSectionProps {
  readonly appId: string;
  readonly envVarsCount: number;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

export function EnvVarsSection({
  appId,
  envVarsCount,
  expanded,
  onToggle,
}: EnvVarsSectionProps) {
  return (
    <CollapsibleSection
      title="Environment Variables"
      icon={Key}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span>
          {envVarsCount} variable{envVarsCount === 1 ? "" : "s"} configured
        </span>
      }
    >
      <EnvVarsManager appId={appId} embedded />
    </CollapsibleSection>
  );
}
