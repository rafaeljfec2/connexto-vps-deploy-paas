import { useState } from "react";
import {
  AlertTriangle,
  Container,
  Download,
  Globe,
  Lock,
  Play,
  RefreshCw,
  Server,
  Square,
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { PageHeader } from "@/components/page-header";
import {
  ContainerRow,
  NginxSiteCard,
  SSLCertificateRow,
  useBackupMutation,
  useMigrationStatus,
  useStartContainersMutation,
  useStopContainersMutation,
  useStopNginxMutation,
} from "@/features/migration";

export function MigrationPage() {
  const [selectedContainers, setSelectedContainers] = useState<Set<string>>(
    new Set(),
  );
  const [expandedSites, setExpandedSites] = useState<Set<number>>(new Set());

  const { data: status, isLoading, refetch } = useMigrationStatus();
  const backupMutation = useBackupMutation();
  const stopContainersMutation = useStopContainersMutation();
  const startContainersMutation = useStartContainersMutation();
  const stopNginxMutation = useStopNginxMutation();

  const toggleSiteExpanded = (index: number) => {
    setExpandedSites((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(index)) {
        newSet.delete(index);
      } else {
        newSet.add(index);
      }
      return newSet;
    });
  };

  const toggleContainer = (id: string) => {
    setSelectedContainers((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  const toggleAllContainers = () => {
    if (!status) return;
    const containers = status.containers ?? [];
    setSelectedContainers((prev) => {
      if (prev.size === containers.length) {
        return new Set();
      }
      return new Set(containers.map((c) => c.id));
    });
  };

  const handleStopContainers = () => {
    stopContainersMutation.mutate([...selectedContainers], {
      onSuccess: () => setSelectedContainers(new Set()),
    });
  };

  const handleStartContainers = () => {
    startContainersMutation.mutate([...selectedContainers], {
      onSuccess: () => setSelectedContainers(new Set()),
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!status) {
    return (
      <Alert variant="destructive">
        <AlertDescription>Failed to load migration status</AlertDescription>
      </Alert>
    );
  }

  const warnings = status.warnings ?? [];
  const nginxSites = status.nginxSites ?? [];
  const containers = status.containers ?? [];
  const sslCertificates = status.sslCertificates ?? [];

  return (
    <div className="space-y-4 sm:space-y-6">
      <PageHeader
        backTo="/settings"
        title="Migration Center"
        description="Migrate from Nginx to Traefik"
        icon={Server}
        actions={
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        }
      />

      {warnings.length > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <ul className="list-disc list-inside">
              {warnings.map((warning, i) => (
                <li key={i}>{warning}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      <ProxyStatusCard
        proxy={status.proxy}
        traefikReady={status.traefikReady}
        lastBackupPath={status.lastBackupPath}
        onBackup={() => backupMutation.mutate()}
        onStopNginx={() => stopNginxMutation.mutate()}
        isBackingUp={backupMutation.isPending}
        isStoppingNginx={stopNginxMutation.isPending}
      />

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Nginx Sites Detected ({nginxSites.length})
          </CardTitle>
          <CardDescription>
            Sites configured in your Nginx installation
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {nginxSites.length === 0 ? (
            <p className="text-muted-foreground">No Nginx sites detected</p>
          ) : (
            nginxSites.map((site, index) => (
              <NginxSiteCard
                key={index}
                site={site}
                index={index}
                expanded={expandedSites.has(index)}
                onToggle={() => toggleSiteExpanded(index)}
                certificates={sslCertificates}
                containers={containers}
              />
            ))
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Container className="h-5 w-5" />
                Docker Containers ({containers.length})
              </CardTitle>
              <CardDescription>
                Containers running on your server
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                checked={
                  selectedContainers.size === containers.length &&
                  containers.length > 0
                }
                onCheckedChange={toggleAllContainers}
              />
              <span className="text-sm">Select All</span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {containers.length === 0 ? (
            <p className="text-muted-foreground">No containers detected</p>
          ) : (
            <>
              <div className="space-y-2">
                {containers.map((container) => (
                  <ContainerRow
                    key={container.id}
                    container={container}
                    selected={selectedContainers.has(container.id)}
                    onToggle={() => toggleContainer(container.id)}
                  />
                ))}
              </div>

              {selectedContainers.size > 0 && (
                <div className="flex gap-2 pt-4 border-t">
                  <span className="text-sm text-muted-foreground">
                    {selectedContainers.size} selected
                  </span>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleStopContainers}
                    disabled={stopContainersMutation.isPending}
                  >
                    <Square className="h-4 w-4 mr-2" />
                    Stop Selected
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleStartContainers}
                    disabled={startContainersMutation.isPending}
                  >
                    <Play className="h-4 w-4 mr-2" />
                    Start Selected
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lock className="h-5 w-5" />
            SSL Certificates ({sslCertificates.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {sslCertificates.length === 0 ? (
            <p className="text-muted-foreground">
              No SSL certificates detected
            </p>
          ) : (
            <div className="space-y-2">
              {sslCertificates.map((cert, index) => (
                <SSLCertificateRow key={index} certificate={cert} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface ProxyStatusCardProps {
  readonly proxy: { type: string; running: boolean; version?: string };
  readonly traefikReady: boolean;
  readonly lastBackupPath?: string;
  readonly onBackup: () => void;
  readonly onStopNginx: () => void;
  readonly isBackingUp: boolean;
  readonly isStoppingNginx: boolean;
}

function ProxyStatusCard({
  proxy,
  traefikReady,
  lastBackupPath,
  onBackup,
  onStopNginx,
  isBackingUp,
  isStoppingNginx,
}: ProxyStatusCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Proxy Status
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium">Current Proxy</p>
            <div className="flex items-center gap-2">
              <Badge variant={proxy.running ? "default" : "secondary"}>
                {proxy.type.toUpperCase()}
              </Badge>
              {proxy.version && (
                <span className="text-sm text-muted-foreground">
                  v{proxy.version}
                </span>
              )}
              <Badge variant={proxy.running ? "default" : "outline"}>
                {proxy.running ? "Running" : "Stopped"}
              </Badge>
            </div>
          </div>
          <div className="space-y-1 text-right">
            <p className="text-sm font-medium">Traefik</p>
            <Badge variant={traefikReady ? "default" : "outline"}>
              {traefikReady ? "Ready" : "Not Running"}
            </Badge>
          </div>
        </div>

        {proxy.type === "nginx" && proxy.running && (
          <div className="flex gap-2">
            <Button variant="outline" onClick={onBackup} disabled={isBackingUp}>
              <Download className="h-4 w-4 mr-2" />
              {isBackingUp ? "Creating Backup..." : "Create Backup"}
            </Button>
            <Button
              variant="destructive"
              onClick={onStopNginx}
              disabled={isStoppingNginx}
            >
              <Square className="h-4 w-4 mr-2" />
              {isStoppingNginx ? "Stopping..." : "Stop Nginx"}
            </Button>
          </div>
        )}

        {lastBackupPath && (
          <p className="text-sm text-muted-foreground">
            Last backup: {lastBackupPath}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
