import { useState } from "react";
import {
  ArrowRightLeft,
  ChevronDown,
  ChevronRight,
  Code,
  Globe,
  Lock,
  Radio,
  Shield,
  Zap,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/services/api";
import type { MigrationContainer, NginxSite, SSLCertificate } from "@/types";
import { useMigrateSiteMutation } from "../hooks/use-migration";

interface NginxSiteCardProps {
  readonly site: NginxSite;
  readonly index: number;
  readonly expanded: boolean;
  readonly onToggle: () => void;
  readonly certificates: readonly SSLCertificate[];
  readonly containers: readonly MigrationContainer[];
}

export function NginxSiteCard({
  site,
  index,
  expanded,
  onToggle,
  certificates,
  containers,
}: NginxSiteCardProps) {
  const [traefikPreview, setTraefikPreview] = useState<string | null>(null);
  const [selectedContainer, setSelectedContainer] = useState<string>("");
  const migrateMutation = useMigrateSiteMutation();

  const loadTraefikPreview = async () => {
    const preview = await api.migration.getTraefikConfig(index);
    setTraefikPreview(preview.yaml);
  };

  const serverNames = site.serverNames ?? [];
  const locations = site.locations ?? [];
  const cert = certificates.find((c) => serverNames.includes(c.domain));
  const uniquePorts = [
    ...new Set(locations.map((l) => l.proxyPort).filter(Boolean)),
  ];

  if (serverNames.length === 0) {
    return null;
  }

  return (
    <Collapsible open={expanded} onOpenChange={onToggle}>
      <div className="border rounded-lg">
        <CollapsibleTrigger className="w-full p-3 sm:p-4 hover:bg-muted/50">
          <div className="flex items-start gap-2 sm:gap-3">
            <div className="flex items-center gap-2 shrink-0 pt-0.5">
              {expanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              <Globe className="h-4 w-4 sm:h-5 sm:w-5 text-blue-500" />
            </div>
            <div className="flex-1 min-w-0 text-left">
              <p className="font-medium text-sm sm:text-base truncate">
                {serverNames[0]}
              </p>
              <div className="flex flex-wrap items-center gap-1 sm:gap-2 text-xs sm:text-sm text-muted-foreground mt-1">
                {site.sslEnabled && (
                  <Badge variant="outline" className="text-green-600 text-xs">
                    <Lock className="h-3 w-3 mr-1" />
                    SSL
                  </Badge>
                )}
                <span>Locations: {locations.length}</span>
                {uniquePorts.length > 0 && (
                  <span className="truncate">
                    Ports: {uniquePorts.join(", ")}
                  </span>
                )}
              </div>
              <div className="flex flex-wrap items-center gap-1 mt-1 sm:hidden">
                {site.hasSSE && (
                  <Badge variant="secondary" className="text-xs">
                    <Radio className="h-3 w-3 mr-1" />
                    SSE
                  </Badge>
                )}
                {site.hasWebSocket && (
                  <Badge variant="secondary" className="text-xs">
                    <Zap className="h-3 w-3 mr-1" />
                    WS
                  </Badge>
                )}
              </div>
            </div>
            <div className="hidden sm:flex items-center gap-2 shrink-0">
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
                    Expires: {new Date(cert.expiresAt).toLocaleDateString()} (
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
                {locations.map((loc, i) => (
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

            <div className="flex flex-col sm:flex-row gap-2 pt-2">
              <Dialog>
                <DialogTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full sm:w-auto justify-start sm:justify-center"
                  >
                    <Code className="h-4 w-4 mr-2" />
                    <span className="sm:hidden">Nginx Config</span>
                    <span className="hidden sm:inline">View Raw Config</span>
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-[95vw] sm:max-w-3xl">
                  <DialogHeader>
                    <DialogTitle className="text-sm sm:text-base truncate">
                      Nginx - {serverNames[0]}
                    </DialogTitle>
                  </DialogHeader>
                  <ScrollArea className="h-[60vh] sm:h-[500px]">
                    <pre className="text-xs sm:text-sm p-2 sm:p-4 bg-muted rounded overflow-x-auto">
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
                    className="w-full sm:w-auto justify-start sm:justify-center"
                    onClick={loadTraefikPreview}
                  >
                    <Shield className="h-4 w-4 mr-2" />
                    <span className="sm:hidden">Traefik Labels</span>
                    <span className="hidden sm:inline">
                      View Traefik Config
                    </span>
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-[95vw] sm:max-w-3xl">
                  <DialogHeader>
                    <DialogTitle className="text-sm sm:text-base truncate">
                      Traefik - {serverNames[0]}
                    </DialogTitle>
                  </DialogHeader>
                  <ScrollArea className="h-[60vh] sm:h-[500px]">
                    <pre className="text-xs sm:text-sm p-2 sm:p-4 bg-muted rounded overflow-x-auto">
                      {traefikPreview ?? "Loading..."}
                    </pre>
                  </ScrollArea>
                </DialogContent>
              </Dialog>
            </div>

            <div className="flex flex-col sm:flex-row gap-2 pt-4 border-t mt-4">
              <Select
                value={selectedContainer}
                onValueChange={setSelectedContainer}
              >
                <SelectTrigger className="w-full sm:w-[280px]">
                  <SelectValue placeholder="Select container" />
                </SelectTrigger>
                <SelectContent>
                  {containers.map((container) => (
                    <SelectItem key={container.id} value={container.id}>
                      {container.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Button
                size="sm"
                className="w-full sm:w-auto"
                disabled={!selectedContainer || migrateMutation.isPending}
                onClick={() =>
                  migrateMutation.mutate({
                    siteIndex: index,
                    containerId: selectedContainer,
                  })
                }
              >
                <ArrowRightLeft className="h-4 w-4 mr-2" />
                {migrateMutation.isPending ? "Migrating..." : "Migrate"}
              </Button>
            </div>

            {migrateMutation.isSuccess && (
              <div className="mt-2 p-2 bg-green-500/10 border border-green-500/20 rounded text-sm text-green-600">
                Migration successful! Container{" "}
                {migrateMutation.data.containerName} now has Traefik labels.
              </div>
            )}

            {migrateMutation.isError &&
              migrateMutation.error instanceof Error && (
                <div className="mt-2 p-2 bg-red-500/10 border border-red-500/20 rounded text-sm text-red-600">
                  Migration failed: {migrateMutation.error.message}
                </div>
              )}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
