import {
  CheckCircle,
  Cpu,
  Globe,
  HardDrive,
  HeartPulse,
  Link2,
  Link2Off,
  Network,
  Settings,
  XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { NetworksManager, VolumesManager } from "@/features/resources";
import { CollapsibleSection } from "./collapsible-section";
import { DomainManager } from "./domain-manager";

interface AppConfigData {
  readonly hostPort: number;
  readonly port: number;
  readonly resources: { memory: string; cpu: string };
  readonly healthcheck: {
    path: string;
    interval: string;
    timeout: string;
    retries: number;
  };
  readonly domains?: readonly string[];
}

interface WebhookActions {
  readonly setupWebhook: {
    isPending: boolean;
    isError: boolean;
    error: unknown;
  };
  readonly removeWebhook: { isPending: boolean };
  readonly handleSetupWebhook: () => void;
  readonly handleRemoveWebhook: () => void;
}

interface WebhookStatusData {
  readonly active?: boolean;
  readonly configuredUrl?: string | null;
}

interface DeploymentConfigSectionProps {
  readonly appConfig: AppConfigData;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function DeploymentConfigSection({
  appConfig,
  expanded,
  onToggle,
}: DeploymentConfigSectionProps) {
  const portDisplay =
    appConfig.hostPort === appConfig.port
      ? appConfig.port
      : `${appConfig.hostPort}:${appConfig.port}`;

  return (
    <CollapsibleSection
      title="Deployment Config"
      icon={Settings}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span>
          Port {portDisplay} • {appConfig.resources.memory} RAM •{" "}
          {appConfig.resources.cpu} CPU
        </span>
      }
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            <Network className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">Port:</span>
            <span className="font-mono">{portDisplay}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <HeartPulse className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">Health Check:</span>
            <span className="font-mono">{appConfig.healthcheck.path}</span>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="ml-6">
              Interval: {appConfig.healthcheck.interval} | Timeout:{" "}
              {appConfig.healthcheck.timeout} | Retries:{" "}
              {appConfig.healthcheck.retries}
            </span>
          </div>
        </div>
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            <HardDrive className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">Memory:</span>
            <span className="font-mono">{appConfig.resources.memory}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Cpu className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">CPU:</span>
            <span className="font-mono">{appConfig.resources.cpu}</span>
          </div>
          {appConfig.domains && appConfig.domains.length > 0 && (
            <div className="flex items-start gap-2 text-sm">
              <Globe className="h-4 w-4 text-muted-foreground mt-0.5" />
              <div>
                <span className="font-medium">Domains:</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {appConfig.domains.map((domain) => (
                    <span
                      key={domain}
                      className="px-2 py-0.5 bg-muted rounded text-xs font-mono"
                    >
                      {domain}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </CollapsibleSection>
  );
}

interface WebhookSectionProps {
  readonly webhookId: number | null;
  readonly webhookStatus: WebhookStatusData | null | undefined;
  readonly actions: WebhookActions;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function WebhookSection({
  webhookId,
  webhookStatus,
  actions,
  expanded,
  onToggle,
}: WebhookSectionProps) {
  const isConfigured = Boolean(webhookId);

  return (
    <CollapsibleSection
      title="GitHub Webhook"
      icon={Link2}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        isConfigured ? (
          <span className="text-green-500 flex items-center gap-1">
            <CheckCircle className="h-3 w-3" />
            Configured
            {webhookStatus?.active === false && (
              <span className="text-yellow-500 ml-1">(inactive)</span>
            )}
          </span>
        ) : (
          <span className="text-muted-foreground flex items-center gap-1">
            <XCircle className="h-3 w-3" />
            Not configured
          </span>
        )
      }
      actions={
        isConfigured ? (
          <Button
            variant="outline"
            size="sm"
            onClick={actions.handleRemoveWebhook}
            disabled={actions.removeWebhook.isPending}
          >
            <Link2Off className="h-4 w-4 sm:mr-1" />
            <span className="hidden sm:inline">Remove</span>
          </Button>
        ) : (
          <Button
            size="sm"
            onClick={actions.handleSetupWebhook}
            disabled={actions.setupWebhook.isPending}
          >
            <Link2 className="h-4 w-4 sm:mr-1" />
            <span className="hidden sm:inline">Setup</span>
          </Button>
        )
      }
    >
      <div className="flex flex-col gap-3">
        {actions.setupWebhook.isError && (
          <p className="text-sm text-destructive">
            {actions.setupWebhook.error instanceof Error
              ? actions.setupWebhook.error.message
              : "Failed to setup webhook"}
          </p>
        )}
        <div className="flex items-center gap-3">
          {isConfigured ? (
            <>
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <p className="font-medium">Webhook configured</p>
                <p className="text-sm text-muted-foreground">
                  Auto-deploy enabled for push events
                  {webhookStatus?.active === false && (
                    <span className="text-yellow-500 ml-2">(inactive)</span>
                  )}
                </p>
              </div>
            </>
          ) : (
            <>
              <XCircle className="h-5 w-5 text-muted-foreground" />
              <div>
                <p className="font-medium">Webhook not configured</p>
                <p className="text-sm text-muted-foreground">
                  Configure to enable auto-deploy on push
                </p>
                {webhookStatus?.configuredUrl && (
                  <p className="text-xs text-muted-foreground font-mono mt-1 break-all">
                    URL: {webhookStatus.configuredUrl}
                  </p>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </CollapsibleSection>
  );
}

interface DomainsSectionProps {
  readonly appId: string;
  readonly expanded: boolean;
  readonly onToggle: () => void;
}

function DomainsSection({ appId, expanded, onToggle }: DomainsSectionProps) {
  return (
    <CollapsibleSection
      title="Custom Domains"
      icon={Globe}
      expanded={expanded}
      onToggle={onToggle}
      summary={<span className="text-muted-foreground">Cloudflare DNS</span>}
    >
      <DomainManager appId={appId} />
    </CollapsibleSection>
  );
}

interface NetworksSectionProps {
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly containerId?: string;
  readonly containerNetworks?: readonly string[];
}

function NetworksSection({
  expanded,
  onToggle,
  containerId,
  containerNetworks,
}: NetworksSectionProps) {
  return (
    <CollapsibleSection
      title="Docker Networks"
      icon={Network}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span className="text-muted-foreground">Manage container networks</span>
      }
    >
      <NetworksManager
        containerId={containerId}
        containerNetworks={containerNetworks}
      />
    </CollapsibleSection>
  );
}

interface VolumesSectionProps {
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly containerVolumes?: readonly string[];
}

function VolumesSection({
  expanded,
  onToggle,
  containerVolumes,
}: VolumesSectionProps) {
  return (
    <CollapsibleSection
      title="Docker Volumes"
      icon={HardDrive}
      expanded={expanded}
      onToggle={onToggle}
      summary={
        <span className="text-muted-foreground">Manage persistent storage</span>
      }
    >
      <VolumesManager containerVolumes={containerVolumes} />
    </CollapsibleSection>
  );
}

export interface AppSettingsSectionProps {
  readonly appConfig: AppConfigData | undefined;
  readonly webhookId: number | null;
  readonly webhookStatus: WebhookStatusData | null | undefined;
  readonly actions: WebhookActions;
  readonly appId: string;
  readonly containerId?: string;
  readonly containerNetworks?: readonly string[];
  readonly containerVolumes?: readonly string[];
  readonly expandedSections: {
    readonly config?: boolean;
    readonly webhook?: boolean;
    readonly domains?: boolean;
    readonly networks?: boolean;
    readonly volumes?: boolean;
  };
  readonly toggleSection: (
    section: "config" | "webhook" | "domains" | "networks" | "volumes",
  ) => void;
}

export function AppSettingsSection({
  appConfig,
  webhookId,
  webhookStatus,
  actions,
  appId,
  containerId,
  containerNetworks,
  containerVolumes,
  expandedSections,
  toggleSection,
}: AppSettingsSectionProps) {
  return (
    <>
      {appConfig && (
        <DeploymentConfigSection
          appConfig={appConfig}
          expanded={expandedSections.config ?? false}
          onToggle={() => toggleSection("config")}
        />
      )}
      <WebhookSection
        webhookId={webhookId}
        webhookStatus={webhookStatus}
        actions={actions}
        expanded={expandedSections.webhook ?? false}
        onToggle={() => toggleSection("webhook")}
      />
      <DomainsSection
        appId={appId}
        expanded={expandedSections.domains ?? false}
        onToggle={() => toggleSection("domains")}
      />
      <NetworksSection
        expanded={expandedSections.networks ?? false}
        onToggle={() => toggleSection("networks")}
        containerId={containerId}
        containerNetworks={containerNetworks}
      />
      <VolumesSection
        expanded={expandedSections.volumes ?? false}
        onToggle={() => toggleSection("volumes")}
        containerVolumes={containerVolumes}
      />
    </>
  );
}
