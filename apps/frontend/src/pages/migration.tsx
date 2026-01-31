import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  CheckCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  Code,
  Container,
  Download,
  Globe,
  Lock,
  Play,
  Radio,
  RefreshCw,
  Server,
  Shield,
  Square,
  XCircle,
  Zap,
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
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { api } from "@/services/api";
import type { MigrationContainer, NginxSite, SSLCertificate } from "@/types";

export function MigrationPage() {
  const queryClient = useQueryClient();
  const [selectedContainers, setSelectedContainers] = useState<Set<string>>(
    new Set(),
  );
  const [expandedSites, setExpandedSites] = useState<Set<number>>(new Set());

  const {
    data: status,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["migration-status"],
    queryFn: () => api.migration.status(),
    refetchInterval: 10000,
  });

  const backupMutation = useMutation({
    mutationFn: () => api.migration.backup(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["migration-status"] });
    },
  });

  const stopContainersMutation = useMutation({
    mutationFn: (ids: readonly string[]) => api.migration.stopContainers(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["migration-status"] });
      setSelectedContainers(new Set());
    },
  });

  const startContainersMutation = useMutation({
    mutationFn: (ids: readonly string[]) => api.migration.startContainers(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["migration-status"] });
      setSelectedContainers(new Set());
    },
  });

  const stopNginxMutation = useMutation({
    mutationFn: () => api.migration.stopNginx(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["migration-status"] });
    },
  });

  const toggleSiteExpanded = (index: number) => {
    const newExpanded = new Set(expandedSites);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedSites(newExpanded);
  };

  const toggleContainer = (id: string) => {
    const newSelected = new Set(selectedContainers);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedContainers(newSelected);
  };

  const toggleAllContainers = () => {
    if (!status) return;
    const containers = status.containers ?? [];
    if (selectedContainers.size === containers.length) {
      setSelectedContainers(new Set());
    } else {
      setSelectedContainers(new Set(containers.map((c) => c.id)));
    }
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
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Migration Center</h1>
          <p className="text-muted-foreground">
            Migrate from Nginx to Traefik and manage your server configuration
          </p>
        </div>
        <Button variant="outline" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

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
                <Badge variant={status.proxy.running ? "default" : "secondary"}>
                  {status.proxy.type.toUpperCase()}
                </Badge>
                {status.proxy.version && (
                  <span className="text-sm text-muted-foreground">
                    v{status.proxy.version}
                  </span>
                )}
                <Badge variant={status.proxy.running ? "default" : "outline"}>
                  {status.proxy.running ? "Running" : "Stopped"}
                </Badge>
              </div>
            </div>
            <div className="space-y-1 text-right">
              <p className="text-sm font-medium">Traefik</p>
              <Badge variant={status.traefikReady ? "default" : "outline"}>
                {status.traefikReady ? "Ready" : "Not Running"}
              </Badge>
            </div>
          </div>

          {status.proxy.type === "nginx" && status.proxy.running && (
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => backupMutation.mutate()}
                disabled={backupMutation.isPending}
              >
                <Download className="h-4 w-4 mr-2" />
                {backupMutation.isPending
                  ? "Creating Backup..."
                  : "Create Backup"}
              </Button>
              <Button
                variant="destructive"
                onClick={() => stopNginxMutation.mutate()}
                disabled={stopNginxMutation.isPending}
              >
                <Square className="h-4 w-4 mr-2" />
                {stopNginxMutation.isPending ? "Stopping..." : "Stop Nginx"}
              </Button>
            </div>
          )}

          {status.lastBackupPath && (
            <p className="text-sm text-muted-foreground">
              Last backup: {status.lastBackupPath}
            </p>
          )}
        </CardContent>
      </Card>

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
                    onClick={() =>
                      stopContainersMutation.mutate([...selectedContainers])
                    }
                    disabled={stopContainersMutation.isPending}
                  >
                    <Square className="h-4 w-4 mr-2" />
                    Stop Selected
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      startContainersMutation.mutate([...selectedContainers])
                    }
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

interface NginxSiteCardProps {
  readonly site: NginxSite;
  readonly index: number;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly certificates: readonly SSLCertificate[];
}

function NginxSiteCard({
  site,
  index,
  expanded,
  onToggle,
  certificates,
}: NginxSiteCardProps) {
  const [traefikPreview, setTraefikPreview] = useState<string | null>(null);

  const loadTraefikPreview = async () => {
    const preview = await api.migration.getTraefikConfig(index);
    setTraefikPreview(preview.yaml);
  };

  const cert = certificates.find((c) => site.serverNames.includes(c.domain));
  const uniquePorts = [
    ...new Set(site.locations.map((l) => l.proxyPort).filter(Boolean)),
  ];

  return (
    <Collapsible open={expanded} onOpenChange={onToggle}>
      <div className="border rounded-lg">
        <CollapsibleTrigger className="w-full p-4 flex items-center justify-between hover:bg-muted/50">
          <div className="flex items-center gap-3">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <Globe className="h-5 w-5 text-blue-500" />
            <div className="text-left">
              <p className="font-medium">{site.serverNames[0]}</p>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                {site.sslEnabled && (
                  <Badge variant="outline" className="text-green-600">
                    <Lock className="h-3 w-3 mr-1" />
                    SSL
                  </Badge>
                )}
                <span>Locations: {site.locations.length}</span>
                {uniquePorts.length > 0 && (
                  <span>Ports: {uniquePorts.join(", ")}</span>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {site.hasSSE && (
              <Badge variant="secondary">
                <Radio className="h-3 w-3 mr-1" />
                SSE
              </Badge>
            )}
            {site.hasWebSocket && (
              <Badge variant="secondary">
                <Zap className="h-3 w-3 mr-1" />
                WebSocket
              </Badge>
            )}
          </div>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="px-4 pb-4 space-y-4 border-t">
            {cert && (
              <div className="pt-4">
                <h4 className="text-sm font-medium mb-2">SSL Certificate</h4>
                <div className="text-sm space-y-1 text-muted-foreground">
                  <p>Provider: {cert.provider}</p>
                  <p>
                    Expires: {new Date(cert.expiresAt).toLocaleDateString()}(
                    {cert.daysUntilExpiry} days)
                  </p>
                  {cert.autoRenew && (
                    <Badge variant="outline">Auto-renew</Badge>
                  )}
                </div>
              </div>
            )}

            <div>
              <h4 className="text-sm font-medium mb-2">Locations</h4>
              <div className="space-y-2">
                {site.locations.map((loc, i) => (
                  <div key={i} className="text-sm p-2 bg-muted rounded">
                    <div className="flex items-center justify-between">
                      <code className="text-blue-600">{loc.path}</code>
                      {loc.proxyPort && (
                        <span className="text-muted-foreground">
                          :{loc.proxyPort}
                        </span>
                      )}
                    </div>
                    <div className="flex gap-2 mt-1">
                      {loc.hasSSE && (
                        <Badge variant="secondary" className="text-xs">
                          SSE
                        </Badge>
                      )}
                      {loc.hasWebSocket && (
                        <Badge variant="secondary" className="text-xs">
                          WebSocket
                        </Badge>
                      )}
                      {loc.isRegex && (
                        <Badge variant="outline" className="text-xs">
                          Regex
                        </Badge>
                      )}
                    </div>
                    {loc.sseConfig && (
                      <div className="mt-2 text-xs text-muted-foreground">
                        <p>
                          Buffering: {loc.sseConfig.bufferingOff ? "off" : "on"}
                        </p>
                        {loc.sseConfig.readTimeout && (
                          <p>Read Timeout: {loc.sseConfig.readTimeout}</p>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <div className="flex gap-2 pt-2">
              <Dialog>
                <DialogTrigger asChild>
                  <Button variant="outline" size="sm">
                    <Code className="h-4 w-4 mr-2" />
                    View Raw Config
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-3xl">
                  <DialogHeader>
                    <DialogTitle>
                      Nginx Configuration - {site.serverNames[0]}
                    </DialogTitle>
                  </DialogHeader>
                  <ScrollArea className="h-[500px]">
                    <pre className="text-sm p-4 bg-muted rounded">
                      {site.rawConfig}
                    </pre>
                  </ScrollArea>
                </DialogContent>
              </Dialog>

              <Dialog>
                <DialogTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={loadTraefikPreview}
                  >
                    <Shield className="h-4 w-4 mr-2" />
                    View Traefik Config
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-3xl">
                  <DialogHeader>
                    <DialogTitle>
                      Traefik Labels - {site.serverNames[0]}
                    </DialogTitle>
                  </DialogHeader>
                  <ScrollArea className="h-[500px]">
                    <pre className="text-sm p-4 bg-muted rounded">
                      {traefikPreview ?? "Loading..."}
                    </pre>
                  </ScrollArea>
                </DialogContent>
              </Dialog>
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

interface ContainerRowProps {
  readonly container: MigrationContainer;
  readonly selected: boolean;
  readonly onToggle: () => void;
}

function ContainerRow({ container, selected, onToggle }: ContainerRowProps) {
  const isRunning = container.state === "running";

  return (
    <div className="flex items-center gap-4 p-3 border rounded-lg hover:bg-muted/50">
      <Checkbox checked={selected} onCheckedChange={onToggle} />
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium">{container.name}</span>
          <Badge variant={isRunning ? "default" : "secondary"}>
            {isRunning ? (
              <CheckCircle className="h-3 w-3 mr-1" />
            ) : (
              <XCircle className="h-3 w-3 mr-1" />
            )}
            {container.state}
          </Badge>
        </div>
        <div className="text-sm text-muted-foreground flex gap-4">
          <span>{container.image}</span>
          {container.ports && container.ports.length > 0 && (
            <span>Ports: {container.ports.join(", ")}</span>
          )}
          {container.uptime && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {container.uptime}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

interface SSLCertificateRowProps {
  readonly certificate: SSLCertificate;
}

function SSLCertificateRow({ certificate }: SSLCertificateRowProps) {
  const isExpiringSoon = certificate.daysUntilExpiry <= 30;
  const isExpired = certificate.isExpired;

  return (
    <div className="flex items-center justify-between p-3 border rounded-lg">
      <div className="flex items-center gap-3">
        <Lock
          className={`h-5 w-5 ${isExpired ? "text-red-500" : isExpiringSoon ? "text-yellow-500" : "text-green-500"}`}
        />
        <div>
          <p className="font-medium">{certificate.domain}</p>
          <p className="text-sm text-muted-foreground">
            Provider: {certificate.provider}
            {certificate.autoRenew && " • Auto-renew enabled"}
          </p>
        </div>
      </div>
      <div className="text-right">
        <Badge
          variant={
            isExpired ? "destructive" : isExpiringSoon ? "outline" : "secondary"
          }
        >
          {isExpired ? "Expired" : `${certificate.daysUntilExpiry} days left`}
        </Badge>
        <p className="text-xs text-muted-foreground mt-1">
          {new Date(certificate.expiresAt).toLocaleDateString()}
        </p>
      </div>
    </div>
  );
}
